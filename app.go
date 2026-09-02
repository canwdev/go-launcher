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
	"strings"
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
	GameMode      bool `json:"game_mode"`
	AbsolutePaths bool `json:"absolute_paths"`
}

type AppStore struct {
	Apps       map[string]*AppItem `json:"apps"` // global app pool
	Categories []CategoryNode      `json:"categories"`
	Settings   Settings            `json:"settings"`
}

type ItemState struct {
	Running   bool   `json:"running"`
	RuntimeMs int64  `json:"runtime_ms"`           // persisted baseline (accumulated)
	StartAt   int64  `json:"start_at,omitempty"`   // unix ms of process start, 0 when not running
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
		Settings: Settings{GameMode: true, AbsolutePaths: true},
	}
}

// settingsMissingKey reports whether the raw JSON's settings object lacks the
// given key (or the whole settings block is missing).
func settingsMissingKey(data []byte, key string) bool {
	var probe struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(data, &probe); err != nil || len(probe.Settings) == 0 {
		return true
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(probe.Settings, &keys); err != nil {
		return true
	}
	_, ok := keys[key]
	return !ok
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
	// Files predating the game_mode key (renamed from auto_minimize) carry no
	// game_mode setting; default it to ON so the runtime display stays visible
	// on startup. An explicit "game_mode": false is always respected.
	if settingsMissingKey(data, "game_mode") {
		store.Settings.GameMode = true
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

func openFile(path string, args []string, workDir string) error {
	// Executables are launched through the real process-run mechanism
	// (startUntracked) with the frontend-supplied args and working directory,
	// but without full tracking: no process handle is kept for Stop and no
	// runtime is accumulated.
	if isExecutable(path) {
		return startUntracked(path, args, workDir)
	}
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
	app := &App{
		store:        loadStore(),
		runtimeStats: map[string]int64{},
		running:      map[string]*runningProc{},
		iconCache:    map[string]string{},
	}
	// Seed in-memory runtime stats from the persisted store so GetData/buildState
	// return the real accumulated time on startup instead of 0 (which previously
	// made the frontend show no real runtime on every fresh launch).
	for guid, item := range app.store.Apps {
		if item != nil {
			app.runtimeStats[guid] = item.RuntimeMs
		}
	}
	app.pruneIconFiles()
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Frontend drives its own 30s display refresh (useAutoRuntime); the old
	// backend 30s state tick was redundant and has been removed. State is still
	// pushed on launch/stop/icon changes via emitState.
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
	url := "data:" + mimeTypeForIcon(rel) + ";base64," + base64.StdEncoding.EncodeToString(data)
	a.iconCache[rel] = url
	return url
}

func mimeTypeForIcon(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/png"
	}
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
			// Frontend computes the live elapsed (now - start_at); the backend only
			// reports the process start time and the accumulated baseline.
			st.StartAt = p.start.UnixMilli()
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
	a.pruneIconFiles()
	return nil
}

func (a *App) pruneIconFiles() {
	referenced := make(map[string]bool)
	for _, app := range a.store.Apps {
		if app != nil && app.Icon != "" {
			referenced[filepath.Base(app.Icon)] = true
		}
	}
	entries, err := os.ReadDir(filepath.Join(absBase, iconsDir))
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if referenced[e.Name()] {
			continue
		}
		_ = os.Remove(filepath.Join(absBase, iconsDir, e.Name()))
	}
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

func convertItemPaths(item *AppItem, absolute bool) {
	if item == nil {
		return
	}
	if absolute {
		if item.Path != "" {
			item.Path = absPath(item.Path)
		}
		if item.WorkingDir != "" {
			item.WorkingDir = absPath(item.WorkingDir)
		}
		if item.Icon != "" {
			item.Icon = absPath(item.Icon)
		}
		return
	}
	if item.Path != "" {
		item.Path = toStoredPath(absPath(item.Path))
	}
	if item.WorkingDir != "" {
		item.WorkingDir = toStoredPath(absPath(item.WorkingDir))
	}
	if item.Icon != "" {
		item.Icon = toStoredPath(absPath(item.Icon))
	}
}

func (a *App) ConvertToAbsolute() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, app := range a.store.Apps {
		convertItemPaths(app, true)
	}
	a.writeStore()
	return nil
}

func (a *App) ConvertToRelative() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, app := range a.store.Apps {
		convertItemPaths(app, false)
	}
	a.writeStore()
	return nil
}

func (a *App) ConvertItemToAbsolute(guid string) error {
	return a.convertItem(guid, true)
}

func (a *App) ConvertItemToRelative(guid string) error {
	return a.convertItem(guid, false)
}

func (a *App) convertItem(guid string, absolute bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	item := a.findItem(guid)
	if item == nil {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	convertItemPaths(item, absolute)
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

func (a *App) PickImageFile(initialDir string) (string, error) {
	if initialDir == "" {
		initialDir = absBase
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Image",
		DefaultDirectory: initialDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.svg)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.svg"},
		},
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
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.svg)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.svg"},
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
	gameMode := a.store.Settings.GameMode
	workingDir := ""
	if item.WorkingDir != "" {
		workingDir = absPath(item.WorkingDir)
	}
	args := splitArgs(item.Args)
	a.mu.Unlock()

	// Game mode off: plain open only - no tracking, no timing, no window control.
	if !gameMode {
		return openFile(path, args, workingDir)
	}

	if isExecutable(path) {
		p := &runningProc{}
		if err := startTracked(path, args, workingDir, p); err != nil {
			return err
		}
		a.mu.Lock()
		a.running[guid] = p
		a.mu.Unlock()
		a.emitState()
		runtime.WindowMinimise(a.ctx)
		go func() {
			_ = p.wait()
			elapsed := time.Since(p.start).Milliseconds()
			a.mu.Lock()
			a.runtimeStats[guid] += elapsed
			delete(a.running, guid)
			restore := len(a.running) == 0
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
	return openFile(path, args, workingDir)
}

func (a *App) Launch(guid string) error {
	return a.launch(guid)
}

// Open launches an item without any process tracking (no runtime tracking, no
// Stop handle, no auto-minimize). Used for items that opt into manual-only
// timing (autoTimer), e.g. programs the launcher cannot track as a process.
func (a *App) Open(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	args := splitArgs(item.Args)
	workDir := ""
	if item.WorkingDir != "" {
		workDir = absPath(item.WorkingDir)
	}
	a.mu.Unlock()
	return openFile(path, args, workDir)
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

// SetRuntimeMs overrides the cumulative runtime for an item. It writes the
// authoritative runtimeStats directly so the normal SaveData merge cannot
// clobber a user-edited value. Negative input is clamped to 0.
func (a *App) SetRuntimeMs(guid string, ms int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	item := a.findItem(guid)
	if item == nil {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	if ms < 0 {
		ms = 0
	}
	a.runtimeStats[guid] = ms
	item.RuntimeMs = ms
	a.writeStore()
	return nil
}
