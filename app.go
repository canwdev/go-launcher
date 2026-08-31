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
	IconURL    string `json:"iconURL,omitempty"`
	Running    bool   `json:"running,omitempty"`
}

type Tab struct {
	GUID  string     `json:"guid"`
	Name  string     `json:"name"`
	Slots []*AppItem `json:"slots"`
}

type Settings struct {
	AutoMinimize bool `json:"auto_minimize"`
}

type AppStore struct {
	Version       string   `json:"version"`
	ActiveTabGUID string   `json:"active_tab_guid"`
	Tabs          []Tab    `json:"tabs"`
	Settings      Settings `json:"settings"`
}

type runningProc struct {
	start   time.Time
	wait    func() error
	stop    func() error
	cleanup func()
}

const saveFile = "go-launcher-data.json"

const iconsDir = "icons"

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
		Version: "1",
		Tabs: []Tab{
			{GUID: uuid.NewString(), Name: "Default", Slots: []*AppItem{}},
		},
		Settings: Settings{AutoMinimize: true},
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
	if store.Version == "" {
		store.Version = "1"
	}
	if len(store.Tabs) == 0 {
		store.Tabs = []Tab{{GUID: uuid.NewString(), Name: "Default", Slots: []*AppItem{}}}
	}
	for i := range store.Tabs {
		if store.Tabs[i].GUID == "" {
			store.Tabs[i].GUID = uuid.NewString()
		}
		if store.Tabs[i].Name == "" {
			store.Tabs[i].Name = "Tab"
		}
		if store.Tabs[i].Slots == nil {
			store.Tabs[i].Slots = []*AppItem{}
		}
		for j := range store.Tabs[i].Slots {
			slot := store.Tabs[i].Slots[j]
			if slot == nil {
				continue
			}
			slot.Path = normalizePath(slot.Path)
			if slot.GUID == "" {
				slot.GUID = uuid.NewString()
			}
			if slot.Name == "" {
				slot.Name = defaultTitle(absPath(slot.Path))
			}
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
	for i := range a.store.Tabs {
		for _, slot := range a.store.Tabs[i].Slots {
			if slot != nil && slot.GUID == guid {
				return slot
			}
		}
	}
	return nil
}

func (a *App) findTab(guid string) *Tab {
	for i := range a.store.Tabs {
		if a.store.Tabs[i].GUID == guid {
			return &a.store.Tabs[i]
		}
	}
	return nil
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
			a.emitUpdated()
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.injectRuntime()
	a.writeStore()
}

func (a *App) emitUpdated() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "items:updated")
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

func (a *App) injectRuntime() {
	for i := range a.store.Tabs {
		for _, slot := range a.store.Tabs[i].Slots {
			if slot == nil {
				continue
			}
			slot.IconURL = a.iconURL(slot.Icon)
			slot.Running = false
			slot.RuntimeMs = a.runtimeStats[slot.GUID]
			if p, ok := a.running[slot.GUID]; ok {
				slot.Running = true
				slot.RuntimeMs += time.Since(p.start).Milliseconds()
			}
		}
	}
}

func (a *App) clearTransient() {
	for i := range a.store.Tabs {
		for _, slot := range a.store.Tabs[i].Slots {
			if slot == nil {
				continue
			}
			slot.IconURL = ""
			slot.Running = false
		}
	}
}

func (a *App) writeStore() {
	_ = writeStoreAtomic(a.store)
}

func (a *App) GetData() AppStore {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.injectRuntime()
	return a.store
}

func (a *App) SaveData(store AppStore) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range store.Tabs {
		for _, slot := range store.Tabs[i].Slots {
			if slot == nil {
				continue
			}
			if val, ok := a.runtimeStats[slot.GUID]; ok {
				slot.RuntimeMs = val
			} else if slot.RuntimeMs > 0 {
				a.runtimeStats[slot.GUID] = slot.RuntimeMs
			}
			slot.IconURL = ""
			slot.Running = false
		}
	}
	a.store = store
	a.writeStore()
	return nil
}

func (a *App) AddFiles() []AppItem {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Add Files",
	})
	if err != nil {
		return nil
	}
	if len(files) == 0 {
		return nil
	}
	return a.AddPaths(files)
}

func (a *App) AddPaths(paths []string) []AppItem {
	if len(paths) == 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]AppItem, 0, len(paths))
	for _, raw := range paths {
		path := normalizePath(raw)
		if path == "" {
			continue
		}
		icon := writeIcon(path)
		items = append(items, AppItem{
			GUID:     uuid.NewString(),
			Name:     defaultTitle(path),
			Path:     toStoredPath(path),
			Icon:     icon,
			IconURL:  a.iconURL(icon),
			Running:  false,
			RuntimeMs: 0,
		})
	}
	return items
}

func (a *App) ChangeIcon(guid string) (string, error) {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return "", fmt.Errorf("invalid item guid %q", guid)
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
		return "", err
	}
	if selection == "" {
		return icon0, nil
	}
	f, err := os.Open(selection)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("unsupported image: %v", err)
	}
	icon := saveImage(img, filepath.Base(path)+"-custom")
	if icon == "" {
		return "", fmt.Errorf("failed to save icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return "", fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.writeStore()
	return icon, nil
}

func (a *App) UpdateIcon(guid string) (string, error) {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return "", fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	a.mu.Unlock()

	icon := writeIcon(path)
	if icon == "" {
		return "", fmt.Errorf("failed to regenerate icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return "", fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.writeStore()
	return icon, nil
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
	a.mu.Unlock()

	if isExecutable(path) {
		p := &runningProc{}
		if err := startTracked(path, p); err != nil {
			return err
		}
		a.mu.Lock()
		a.running[guid] = p
		a.mu.Unlock()
		a.emitUpdated()
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
			a.emitUpdated()
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
		a.emitUpdated()
	}
	return nil
}