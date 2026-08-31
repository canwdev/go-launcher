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

type LauncherItem struct {
	GUID      string `json:"guid"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	RuntimeMs int64  `json:"runtime_ms"`
	Icon      string `json:"icon,omitempty"`
}

type LauncherData struct {
	LauncherFiles []LauncherItem `json:"launcher_files"`
	Settings      Settings       `json:"settings"`
}

type Settings struct {
	AutoMinimize bool `json:"auto_minimize"`
}

type LauncherItemView struct {
	GUID      string `json:"guid"`
	Title     string `json:"title"`
	IconURL   string `json:"iconURL"`
	RuntimeMs int64  `json:"runtime_ms"`
	Running   bool   `json:"running"`
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

func loadLauncherData() LauncherData {
	data, err := os.ReadFile(saveFile)
	if err != nil {
		return LauncherData{Settings: Settings{AutoMinimize: true}}
	}
	var ld LauncherData
	if err := json.Unmarshal(data, &ld); err != nil {
		return LauncherData{Settings: Settings{AutoMinimize: true}}
	}
	for i := range ld.LauncherFiles {
		ld.LauncherFiles[i].Path = normalizePath(ld.LauncherFiles[i].Path)
		if ld.LauncherFiles[i].GUID == "" {
			ld.LauncherFiles[i].GUID = uuid.NewString()
		}
		if ld.LauncherFiles[i].Title == "" {
			ld.LauncherFiles[i].Title = defaultTitle(absPath(ld.LauncherFiles[i].Path))
		}
	}
	return ld
}

func writeLauncherData(ld LauncherData) {
	data, _ := json.MarshalIndent(ld, "", "  ")
	_ = os.WriteFile(saveFile, data, 0644)
}

func (a *App) findItem(guid string) *LauncherItem {
	for i := range a.ld.LauncherFiles {
		if a.ld.LauncherFiles[i].GUID == guid {
			return &a.ld.LauncherFiles[i]
		}
	}
	return nil
}

func openFile(path string) error {
	return openWithDefaultHandler(path)
}

type App struct {
	ctx       context.Context
	mu        sync.Mutex
	ld        LauncherData
	running   map[string]*runningProc
	iconCache map[string]string
}

func NewApp() *App {
	return &App{
		ld:        loadLauncherData(),
		running:   map[string]*runningProc{},
		iconCache: map[string]string{},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.OnFileDrop(ctx, func(_x, _y int, paths []string) {
		_ = a.AddPaths(paths)
	})
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
	for path, p := range a.running {
		for i := range a.ld.LauncherFiles {
			if absPath(a.ld.LauncherFiles[i].Path) == path {
				a.ld.LauncherFiles[i].RuntimeMs += time.Since(p.start).Milliseconds()
				break
			}
		}
	}
	a.saveLauncherData()
}

func (a *App) emitUpdated() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "items:updated")
}

func (a *App) saveLauncherData() {
	writeLauncherData(a.ld)
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

func (a *App) GetItems() []LauncherItemView {
	a.mu.Lock()
	defer a.mu.Unlock()
	items := make([]LauncherItemView, 0, len(a.ld.LauncherFiles))
	for i := range a.ld.LauncherFiles {
		item := a.ld.LauncherFiles[i]
		path := absPath(item.Path)
		view := LauncherItemView{
			GUID:      item.GUID,
			Title:     item.Title,
			IconURL:   a.iconURL(item.Icon),
			RuntimeMs: item.RuntimeMs,
		}
		if view.Title == "" {
			view.Title = filepath.Base(path)
		}
		if p, ok := a.running[path]; ok {
			view.Running = true
			view.RuntimeMs += time.Since(p.start).Milliseconds()
		}
		items = append(items, view)
	}
	return items
}

func (a *App) launch(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	if _, ok := a.running[path]; ok {
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	if isExecutable(path) {
		p := &runningProc{}
		if err := startTracked(path, p); err != nil {
			return err
		}
		a.mu.Lock()
		a.running[path] = p
		autoMinimize := a.ld.Settings.AutoMinimize
		a.mu.Unlock()
		a.emitUpdated()
		if autoMinimize {
			runtime.WindowMinimise(a.ctx)
		}
		go func() {
			_ = p.wait()
			elapsed := time.Since(p.start).Milliseconds()
			a.mu.Lock()
			for i := range a.ld.LauncherFiles {
				if absPath(a.ld.LauncherFiles[i].Path) == path {
					a.ld.LauncherFiles[i].RuntimeMs += elapsed
					break
				}
			}
			delete(a.running, path)
			restore := a.ld.Settings.AutoMinimize && len(a.running) == 0
			if p.cleanup != nil {
				p.cleanup()
			}
			a.saveLauncherData()
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

func (a *App) stopProcess(path string) error {
	a.mu.Lock()
	p, ok := a.running[path]
	a.mu.Unlock()
	if ok && p.stop != nil {
		return p.stop()
	}
	return nil
}

func (a *App) Launch(guid string) error {
	return a.launch(guid)
}

func (a *App) Stop(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	path := ""
	if item != nil {
		path = absPath(item.Path)
	}
	a.mu.Unlock()
	if path == "" {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	err := a.stopProcess(path)
	if err == nil {
		a.emitUpdated()
	}
	return err
}

func (a *App) GetAutoMinimize() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ld.Settings.AutoMinimize
}

func (a *App) SetAutoMinimize(enabled bool) error {
	a.mu.Lock()
	a.ld.Settings.AutoMinimize = enabled
	a.saveLauncherData()
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "settings:updated")
	}
	return nil
}

func (a *App) AddFiles() error {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Add Files",
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	return a.AddPaths(files)
}

func (a *App) AddPaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	a.mu.Lock()
	added := false
	for _, raw := range paths {
		path := normalizePath(raw)
		if path == "" {
			continue
		}
		dup := false
		for _, item := range a.ld.LauncherFiles {
			if absPath(item.Path) == path {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		a.ld.LauncherFiles = append(a.ld.LauncherFiles, LauncherItem{
			GUID:  uuid.NewString(),
			Path:  toStoredPath(path),
			Title: defaultTitle(path),
			Icon:  writeIcon(path),
		})
		added = true
	}
	if added {
		a.saveLauncherData()
	}
	a.mu.Unlock()
	if added {
		a.emitUpdated()
	}
	return nil
}

func (a *App) RemoveItem(guid string) error {
	a.mu.Lock()
	found := -1
	for i := range a.ld.LauncherFiles {
		if a.ld.LauncherFiles[i].GUID == guid {
			found = i
			break
		}
	}
	if found < 0 {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	item := a.ld.LauncherFiles[found]
	path := absPath(item.Path)
	a.mu.Unlock()

	if err := a.stopProcess(path); err != nil {
		return err
	}

	a.mu.Lock()
	if item.Icon != "" {
		_ = os.Remove(absPath(item.Icon))
		delete(a.iconCache, item.Icon)
	}
	a.ld.LauncherFiles = append(a.ld.LauncherFiles[:found], a.ld.LauncherFiles[found+1:]...)
	a.saveLauncherData()
	a.mu.Unlock()
	a.emitUpdated()
	return nil
}

func (a *App) RenameItem(guid, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("name cannot be empty")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	item := a.findItem(guid)
	if item == nil {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	item.Title = title
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}

func (a *App) ChangeIcon(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	a.mu.Unlock()

	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Change Icon",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.gif;*.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp"},
		},
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}
	f, err := os.Open(selection)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("unsupported image: %v", err)
	}
	icon := saveImage(img, filepath.Base(path)+"-custom")
	if icon == "" {
		return fmt.Errorf("failed to save icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}

func (a *App) UpdateIcon(guid string) error {
	a.mu.Lock()
	item := a.findItem(guid)
	if item == nil {
		a.mu.Unlock()
		return fmt.Errorf("invalid item guid %q", guid)
	}
	path := absPath(item.Path)
	a.mu.Unlock()

	icon := writeIcon(path)
	if icon == "" {
		return fmt.Errorf("failed to regenerate icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	item = a.findItem(guid)
	if item == nil {
		return fmt.Errorf("invalid item guid %q", guid)
	}
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.saveLauncherData()
	a.emitUpdated()
	return nil
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

func (a *App) Details(guid string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	item := a.findItem(guid)
	if item == nil {
		return ""
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func (a *App) MoveItem(from, to int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := len(a.ld.LauncherFiles)
	if from < 0 || from >= n || to < 0 || to >= n {
		return fmt.Errorf("invalid index")
	}
	if from == to {
		return nil
	}
	item := a.ld.LauncherFiles[from]
	a.ld.LauncherFiles = append(a.ld.LauncherFiles[:from], a.ld.LauncherFiles[from+1:]...)
	a.ld.LauncherFiles = append(a.ld.LauncherFiles[:to], append([]LauncherItem{item}, a.ld.LauncherFiles[to:]...)...)
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}
