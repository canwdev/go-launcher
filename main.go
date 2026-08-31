package main

import (
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

type runningProc struct {
	start   time.Time
	wait    func() error
	stop    func() error
	cleanup func()
}

const saveFile = "go-launcher-data.json"

var ld LauncherData

var running = map[string]*runningProc{}

var removed = map[string]bool{}

var appModified = map[string]bool{}

var dataMu sync.Mutex

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

const iconsDir = "icons"

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

func saveLauncherData(ld LauncherData) {
	disk, ok := readDiskData()
	if !ok {
		writeLauncherData(ld)
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

	memByKey := make(map[string]LauncherItem, len(ld.LauncherFiles))
	for _, item := range ld.LauncherFiles {
		memByKey[itemKey(item)] = item
	}

	merged := make([]LauncherItem, 0, len(disk.LauncherFiles)+len(ld.LauncherFiles))
	for _, k := range order {
		if removed[k] {
			continue
		}
		diskItem := diskByKey[k]
		if memItem, inMem := memByKey[k]; inMem {
			if appModified[k] {
				diskItem = memItem
			} else {
				diskItem.RuntimeMs = memItem.RuntimeMs
			}
		}
		merged = append(merged, diskItem)
	}
	for k, memItem := range memByKey {
		if _, inDisk := diskByKey[k]; !inDisk && !removed[k] {
			merged = append(merged, memItem)
		}
	}
	writeLauncherData(LauncherData{LauncherFiles: merged})
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

func startProcess(path string) error {
	p := &runningProc{}
	if err := startTracked(path, p); err != nil {
		return err
	}
	dataMu.Lock()
	running[path] = p
	dataMu.Unlock()
	go func() {
		p.wait()
		elapsed := time.Since(p.start).Milliseconds()
		dataMu.Lock()
		for i := range ld.LauncherFiles {
			if absPath(ld.LauncherFiles[i].Path) == path {
				ld.LauncherFiles[i].RuntimeMs += elapsed
				break
			}
		}
		delete(running, path)
		if p.cleanup != nil {
			p.cleanup()
		}
		saveLauncherData(ld)
		dataMu.Unlock()
		gwinInvalidate()
	}()
	return nil
}

func stopProcess(path string) error {
	dataMu.Lock()
	p, ok := running[path]
	dataMu.Unlock()
	if !ok || p.stop == nil {
		return nil
	}
	return p.stop()
}

func onWindowClose() {
	dataMu.Lock()
	for path, p := range running {
		for i := range ld.LauncherFiles {
			if absPath(ld.LauncherFiles[i].Path) == path {
				ld.LauncherFiles[i].RuntimeMs += time.Since(p.start).Milliseconds()
				break
			}
		}
	}
	saveLauncherData(ld)
	dataMu.Unlock()
}

func main() {
	ld = loadLauncherData()
	runGUI()
}
