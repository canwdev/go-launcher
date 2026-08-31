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

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type LauncherItem struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	RuntimeMs int64  `json:"runtime_ms"`
	Icon      string `json:"icon,omitempty"`
}

type LauncherData struct {
	LauncherFiles []LauncherItem `json:"launcher_files"`
}

type LauncherItemView struct {
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
		return LauncherData{}
	}
	var ld LauncherData
	if err := json.Unmarshal(data, &ld); err != nil {
		return LauncherData{}
	}
	for i := range ld.LauncherFiles {
		ld.LauncherFiles[i].Path = normalizePath(ld.LauncherFiles[i].Path)
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

func readDiskData() (LauncherData, bool) {
	data, err := os.ReadFile(saveFile)
	if err != nil {
		return LauncherData{}, false
	}
	var ld LauncherData
	if err := json.Unmarshal(data, &ld); err != nil {
		return LauncherData{}, false
	}
	return ld, true
}

func itemKey(item LauncherItem) string {
	return filepath.Clean(absPath(normalizePath(item.Path)))
}

func openFile(path string) error {
	return openWithDefaultHandler(path)
}

type App struct {
	ctx         context.Context
	mu          sync.Mutex
	ld          LauncherData
	running     map[string]*runningProc
	removed     map[string]bool
	appModified map[string]bool
	iconCache   map[string]string
}

func NewApp() *App {
	return &App{
		ld:          loadLauncherData(),
		running:     map[string]*runningProc{},
		removed:     map[string]bool{},
		appModified: map[string]bool{},
		iconCache:   map[string]string{},
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
	disk, ok := readDiskData()
	if !ok {
		writeLauncherData(a.ld)
		return
	}

	diskByKey := make(map[string]LauncherItem, len(disk.LauncherFiles))
	order := make([]string, 0, len(disk.LauncherFiles))
	for _, item := range disk.LauncherFiles {
		k := itemKey(item)
		if _, exists := diskByKey[k]; !exists {
			order = append(order, k)
		}
		diskByKey[k] = item
	}

	memByKey := make(map[string]LauncherItem, len(a.ld.LauncherFiles))
	for _, item := range a.ld.LauncherFiles {
		memByKey[itemKey(item)] = item
	}

	merged := make([]LauncherItem, 0, len(disk.LauncherFiles)+len(a.ld.LauncherFiles))
	for _, k := range order {
		if a.removed[k] {
			continue
		}
		diskItem := diskByKey[k]
		if memItem, inMem := memByKey[k]; inMem {
			if a.appModified[k] {
				diskItem = memItem
			} else {
				diskItem.RuntimeMs = memItem.RuntimeMs
			}
		}
		merged = append(merged, diskItem)
	}
	for k, memItem := range memByKey {
		if _, inDisk := diskByKey[k]; !inDisk && !a.removed[k] {
			merged = append(merged, memItem)
		}
	}
	writeLauncherData(LauncherData{LauncherFiles: merged})
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

func (a *App) launch(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	path := absPath(a.ld.LauncherFiles[id].Path)
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
		a.mu.Unlock()
		a.emitUpdated()
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
			if p.cleanup != nil {
				p.cleanup()
			}
			a.saveLauncherData()
			a.mu.Unlock()
			a.emitUpdated()
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

func (a *App) Launch(id int) error {
	return a.launch(id)
}

func (a *App) Stop(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	path := absPath(a.ld.LauncherFiles[id].Path)
	a.mu.Unlock()
	err := a.stopProcess(path)
	if err == nil {
		a.emitUpdated()
	}
	return err
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
			Path:  toStoredPath(path),
			Title: defaultTitle(path),
			Icon:  writeIcon(path),
		})
		delete(a.removed, itemKey(LauncherItem{Path: toStoredPath(path)}))
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

func (a *App) RemoveItem(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	item := a.ld.LauncherFiles[id]
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
	a.removed[itemKey(item)] = true
	delete(a.appModified, itemKey(item))
	newItems := make([]LauncherItem, 0, len(a.ld.LauncherFiles)-1)
	for i, it := range a.ld.LauncherFiles {
		if i != id {
			newItems = append(newItems, it)
		}
	}
	a.ld.LauncherFiles = newItems
	a.saveLauncherData()
	a.mu.Unlock()
	a.emitUpdated()
	return nil
}

func (a *App) RenameItem(id int, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("name cannot be empty")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		return fmt.Errorf("invalid item index %d", id)
	}
	a.ld.LauncherFiles[id].Title = title
	a.appModified[itemKey(a.ld.LauncherFiles[id])] = true
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}

func (a *App) ChangeIcon(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	path := absPath(a.ld.LauncherFiles[id].Path)
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
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		return fmt.Errorf("invalid item index %d", id)
	}
	item := &a.ld.LauncherFiles[id]
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.appModified[itemKey(*item)] = true
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}

func (a *App) UpdateIcon(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	path := absPath(a.ld.LauncherFiles[id].Path)
	a.mu.Unlock()

	icon := writeIcon(path)
	if icon == "" {
		return fmt.Errorf("failed to regenerate icon")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		return fmt.Errorf("invalid item index %d", id)
	}
	item := &a.ld.LauncherFiles[id]
	if old := item.Icon; old != "" {
		_ = os.Remove(absPath(old))
		delete(a.iconCache, old)
	}
	item.Icon = icon
	a.appModified[itemKey(*item)] = true
	a.saveLauncherData()
	a.emitUpdated()
	return nil
}

func (a *App) Reveal(id int) error {
	a.mu.Lock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		a.mu.Unlock()
		return fmt.Errorf("invalid item index %d", id)
	}
	path := absPath(a.ld.LauncherFiles[id].Path)
	a.mu.Unlock()
	return revealFile(path)
}

func (a *App) Details(id int) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if id < 0 || id >= len(a.ld.LauncherFiles) {
		return ""
	}
	b, err := json.MarshalIndent(a.ld.LauncherFiles[id], "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
