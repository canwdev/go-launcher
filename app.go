package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppItem struct {
	GUID       string `json:"guid"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Icon       string `json:"icon,omitempty"`
	RuntimeMs  int64  `json:"runtime_ms,omitempty"`
	Args       string `json:"args,omitempty"`
	WorkingDir string `json:"working_dir,omitempty"`
}

type CategoryNode struct {
	GUID  string    `json:"guid"`
	Name  string    `json:"name"`
	Slots []*string `json:"slots"` // nil = empty cell, otherwise an app guid
}

type Settings struct {
	AutoMinimize  bool `json:"auto_minimize"`
	AbsolutePaths bool `json:"absolute_paths"`
}

type AppStore struct {
	Apps       map[string]*AppItem `json:"apps"` // global app pool
	Categories []CategoryNode      `json:"categories"`
	Settings   Settings            `json:"settings"`
}

type ItemState struct {
	Running   bool   `json:"running"`
	RuntimeMs int64  `json:"runtime_ms"`
	IconURL   string `json:"icon_url,omitempty"`
}

type AppData struct {
	Store AppStore             `json:"store"`
	State map[string]ItemState `json:"state"`
}

type AddResult struct {
	Items []*AppItem        `json:"items"`
	Icons map[string]string `json:"icons"`
}

type IconResult struct {
	Icon    string `json:"icon"`
	IconURL string `json:"icon_url"`
}

type runningProc struct {
	start   time.Time
	wait    func() error
	stop    func() error
	cleanup func()
}

const dataDir = "go-launcher-data"

const saveFile = dataDir + "/go-launcher-data.json"

const iconsDir = dataDir + "/icons"

var absBase, _ = filepath.Abs(".")

func toStoredPath(abs string) string {
	rel, err := filepath.Rel(absBase, abs)
	if err != nil {
		return abs
	}
	return rel
}

func absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(absBase, p)
}

func saveImage(img image.Image, nameBase string) string {
	if err := os.MkdirAll(filepath.Join(absBase, iconsDir), 0755); err != nil {
		return ""
	}
	name := fmt.Sprintf("%s-%d.png", sanitizeBase(nameBase), time.Now().UnixNano())
	abs := filepath.Join(absBase, iconsDir, name)
	f, err := os.Create(abs)
	if err != nil {
		return ""
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return ""
	}
	return "./" + filepath.ToSlash(filepath.Join(iconsDir, name))
}

func writeIcon(path string) string {
	img, err := iconForFile(path)
	if err != nil {
		return ""
	}
	return saveImage(img, filepath.Base(path))
}

func defaultStore() AppStore {
	return AppStore{
		Apps: map[string]*AppItem{},
		Categories: []CategoryNode{
			{GUID: uuid.NewString(), Name: "Default", Slots: []*string{}},
		},
		Settings: Settings{AutoMinimize: true, AbsolutePaths: true},
	}
}

func loadStore() AppStore {
	data, err := os.ReadFile(saveFile)
	if err != nil {
		return defaultStore()
	}
	var store AppStore
	if err := json.Unmarshal(data, &store); err != nil {
		return defaultStore()
	}
	if store.Apps == nil {
		store.Apps = map[string]*AppItem{}
	}
	if len(store.Apps) == 0 && len(store.Categories) == 0 {
		return defaultStore()
	}
	for guid, app := range store.Apps {
		if app == nil {
			continue
		}
		app.Path = normalizePath(app.Path)
		if app.GUID == "" {
			app.GUID = guid
		}
		if app.Name == "" {
			app.Name = defaultTitle(absPath(app.Path))
		}
	}
	if len(store.Categories) == 0 {
		store.Categories = []CategoryNode{{GUID: uuid.NewString(), Name: "Default", Slots: []*string{}}}
	}
	for i := range store.Categories {
		cat := &store.Categories[i]
		if cat.GUID == "" {
			cat.GUID = uuid.NewString()
		}
		if cat.Name == "" {
			cat.Name = "Category"
		}
		if cat.Slots == nil {
			cat.Slots = []*string{}
		}
	}
	return store
}

func writeStoreAtomic(store AppStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	tmp := saveFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, saveFile)
}

func (a *App) findItem(guid string) *AppItem {
	return a.store.Apps[guid]
}

func openFile(path string) error {
	return openWithDefaultHandler(path)
}

type App struct {
	ctx          context.Context
	mu           sync.Mutex
	store        AppStore
	runtimeStats map[string]int64
	running      map[string]*runningProc
	iconCache    map[string]string
}

func NewApp() *App {
	_ = os.MkdirAll(filepath.Join(absBase, dataDir), 0755)
	return &App{
		store:        loadStore(),
		runtimeStats: map[string]int64{},
		running:      map[string]*runningProc{},
		iconCache:    map[string]string{},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for range tick.C {
			a.emitState()
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.writeStore()
}

func (a *App) emitState() {
	if a.ctx == nil {
		return
	}
	a.mu.Lock()
	state := a.buildState()
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "state:updated", state)
}

func (a *App) iconURL(rel string) string {
	if rel == "" {
		return ""
	}
	if url, ok := a.iconCache[rel]; ok {
		return url
	}
	data, err := os.ReadFile(absPath(rel))
	if err != nil {
		return ""
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	a.iconCache[rel] = url
	return url
}

func (a *App) buildState() map[string]ItemState {
	state := make(map[string]ItemState, len(a.store.Apps))
	for guid, app := range a.store.Apps {
		if app == nil {
			continue
		}
		st := ItemState{IconURL: a.iconURL(app.Icon), RuntimeMs: a.runtimeStats[guid]}
		if p, ok := a.running[guid]; ok {
			st.Running = true
			st.RuntimeMs += time.Since(p.start).Milliseconds()
		}
		state[guid] = st
	}
	return state
}

func (a *App) writeStore() {
	for guid, ms := range a.runtimeStats {
		if app := a.store.Apps[guid]; app != nil {
			app.RuntimeMs = ms
		}
	}
	_ = writeStoreAtomic(a.store)
}

func (a *App) GetData() AppData {
	a.mu.Lock()
	defer a.mu.Unlock()
	return AppData{Store: a.store, State: a.buildState()}
}

func (a *App) SaveData(store AppStore) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for guid, app := range store.Apps {
		if app == nil {
			continue
		}
		if ms, ok := a.runtimeStats[guid]; ok {
			app.RuntimeMs = ms
		} else {
			a.runtimeStats[guid] = app.RuntimeMs
		}
	}
	a.store = store
	a.writeStore()
	return nil
}

func (a *App) AddFiles() AddResult {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Add Files",
	})
	if err != nil {
		return AddResult{}
	}
	if len(files) == 0 {
		return AddResult{}
	}
	return a.AddPaths(files)
}

func (a *App) AddPaths(paths []string) AddResult {
	if len(paths) == 0 {
		return AddResult{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]*AppItem, 0, len(paths))
	icons := make(map[string]string, len(paths))
	for _, raw := range paths {
		path := normalizePath(raw)
		if path == "" {
			continue
		}
		icon := writeIcon(path)
		storedPath := path
		storedWorkDir := filepath.Dir(path)
		if !a.store.Settings.AbsolutePaths {
			storedPath = toStoredPath(path)
			storedWorkDir = toStoredPath(storedWorkDir)
		}
		item := &AppItem{
			GUID:       uuid.NewString(),
			Name:       defaultTitle(path),
			Path:       storedPath,
			WorkingDir: storedWorkDir,
			Icon:       icon,
		}
		items = append(items, item)
		icons[item.GUID] = a.iconURL(icon)
	}
	return AddResult{Items: items, Icons: icons}
}

func (a *App) ConvertToAbsolute() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, app := range a.store.Apps {
		if app == nil {
			continue
		}
		app.Path = absPath(app.Path)
	}
	a.writeStore()
	return nil
}

func (a *App) ConvertToRelative() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, app := range a.store.Apps {
		if app == nil {
			continue
		}
		app.Path = toStoredPath(absPath(app.Path))
	}
	a.writeStore()
	return nil
}

func (a *App) PickFile(initialDir string) (string, error) {
	if initialDir == "" {
		initialDir = absBase
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select File",
		DefaultDirectory: initialDir,
	})
}

func (a *App) PickDirectory(initialDir string) (string, error) {
	if initialDir == "" {
		initialDir = absBase
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Folder",
		DefaultDirectory: initialDir,
	})
}

func (a *App) OpenDirectory(path string) error {
	if path == "" {
		path = absBase
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return openWithDefaultHandler(path)
}

func (a *App) ChangeIcon(guid string) (IconResult, error) {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return IconResult{}, fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	icon0 := item.Icon
	a.mu.Unlock()

	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Change Icon",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.gif;*.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp"},
		},
	})
	if err != nil {
		return IconResult{}, err
	}
	if selection == "" {
		return IconResult{Icon: icon0, IconURL: a.iconURL(icon0)}, nil
	}
	f, err := os.Open(selection)
	if err != nil {
		return IconResult{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return IconResult{}, fmt.Errorf("unsupported image: %v", err)
	}
	icon := saveImage(img, filepath.Base(path)+"-custom")
	if icon == "" {
		return IconResult{}, fmt.Errorf("failed to save icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return IconResult{}, fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.writeStore()
	return IconResult{Icon: icon, IconURL: a.iconURL(icon)}, nil
}

func (a *App) UpdateIcon(guid string) (IconResult, error) {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return IconResult{}, fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	a.mu.Unlock()

	icon := writeIcon(path)
	if icon == "" {
		return IconResult{}, fmt.Errorf("failed to regenerate icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return IconResult{}, fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.writeStore()
	return IconResult{Icon: icon, IconURL: a.iconURL(icon)}, nil
}

func (a *App) Reveal(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	a.mu.Unlock()
	return revealFile(path)
}

func (a *App) launch(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	if _, ok := a.running[guid]; ok {
		a.mu.Unlock()
		return nil
	}
	autoMinimize := a.store.Settings.AutoMinimize
	workingDir := ""
	if item.WorkingDir != "" {
		workingDir = absPath(item.WorkingDir)
	}
	args := splitArgs(item.Args)
	a.mu.Unlock()

	if isExecutable(path) {
		p := &runningProc{}
		if err := startTracked(path, args, workingDir, p); err != nil {
			return err
		}
		a.mu.Lock()
		a.running[guid] = p
		a.mu.Unlock()
		a.emitState()
		if autoMinimize {
			runtime.WindowMinimise(a.ctx)
		}
		go func() {
			_ = p.wait()
			elapsed := time.Since(p.start).Milliseconds()
			a.mu.Lock()
			a.runtimeStats[guid] += elapsed
			delete(a.running, guid)
			restore := a.store.Settings.AutoMinimize && len(a.running) == 0
			if p.cleanup != nil {
				p.cleanup()
			}
			a.writeStore()
			a.mu.Unlock()
			a.emitState()
			if restore {
				runtime.WindowUnminimise(a.ctx)
			}
		}()
		return nil
	}
	return openFile(path)
}

func (a *App) Launch(guid string) error {
	return a.launch(guid)
}

func (a *App) Stop(guid string) error {
	a.mu.Lock()
	p, ok := a.running[guid]
	a.mu.Unlock()
	if ok && p.stop != nil {
		if err := p.stop(); err != nil {
			return err
		}
		a.emitState()
	}
	return nil
}
