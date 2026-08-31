package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
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

var running = map[string]*runningProc{}

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

func writeIcon(path string) string {
	img, err := iconForFile(path)
	if err != nil {
		return ""
	}
	if err := os.MkdirAll(filepath.Join(absBase, iconsDir), 0755); err != nil {
		return ""
	}
	base := strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		}
		return r
	}, filepath.Base(path))
	if len(base) > 40 {
		base = base[:40]
	}
	name := fmt.Sprintf("%s-%d.png", base, time.Now().UnixNano())
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

type tapRow struct {
	fyne.CanvasObject
	onDouble func()
}

func (t *tapRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.CanvasObject)
}

func (t *tapRow) DoubleTapped(_ *fyne.PointEvent) {
	if t.onDouble != nil {
		t.onDouble()
	}
}

func normalizePath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	p := strings.ReplaceAll(path, "/", `\`)
	if len(p) >= 3 && p[0] == '\\' && p[2] == ':' &&
		((p[1] >= 'A' && p[1] <= 'Z') || (p[1] >= 'a' && p[1] <= 'z')) {
		p = p[1:]
	}
	return p
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
			ld.LauncherFiles[i].Title = filepath.Base(absPath(ld.LauncherFiles[i].Path))
		}
	}
	return ld
}

func saveLauncherData(ld LauncherData) {
	data, _ := json.MarshalIndent(ld, "", "  ")
	_ = os.WriteFile(saveFile, data, 0644)
}

func formatRuntime(ms int64) string {
	total := ms / 1000
	d := total / 86400
	h := (total % 86400) / 3600
	m := (total % 3600) / 60
	parts := make([]string, 0, 3)
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if len(parts) == 0 {
		return "1m"
	}
	return strings.Join(parts, " ")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".com"
	}
	return info.Mode()&0111 != 0
}

func openFile(path string) error {
	return openWithDefaultHandler(path)
}

func main() {
	a := app.New()
	w := a.NewWindow("Go Launcher")

	ld := loadLauncherData()

	w.SetCloseIntercept(func() {
		for path, p := range running {
			for i := range ld.LauncherFiles {
				if absPath(ld.LauncherFiles[i].Path) == path {
					ld.LauncherFiles[i].RuntimeMs += time.Since(p.start).Milliseconds()
					break
				}
			}
		}
		saveLauncherData(ld)
		w.Close()
	})

	var fileList *widget.List

	startProcess := func(path string) error {
		p := &runningProc{}
		if err := startTracked(path, p); err != nil {
			return err
		}
		running[path] = p
		go func() {
			p.wait()
			elapsed := time.Since(p.start).Milliseconds()
			fyne.Do(func() {
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
				fileList.Refresh()
			})
		}()
		return nil
	}

	stopProcess := func(path string) {
		if p, ok := running[path]; ok && p.stop != nil {
			if err := p.stop(); err != nil {
				dialog.ShowError(err, w)
			}
		}
	}

	fileList = widget.NewList(
		func() int { return len(ld.LauncherFiles) },
		func() fyne.CanvasObject {
			iconImg := canvas.NewImageFromFile("")
			iconImg.SetMinSize(fyne.NewSize(24, 24))
			iconImg.FillMode = canvas.ImageFillContain
			name := widget.NewLabel("placeholder")
			runtime := widget.NewLabel("")
			runBtn := widget.NewButton("Run", nil)
			delBtn := widget.NewButton("Delete", nil)
			return &tapRow{CanvasObject: container.NewHBox(iconImg, name, layout.NewSpacer(), runtime, runBtn, delBtn)}
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			path := absPath(ld.LauncherFiles[id].Path)
			row := obj.(*tapRow)
			box := row.CanvasObject.(*fyne.Container)
			iconImg := box.Objects[0].(*canvas.Image)
			name := box.Objects[1].(*widget.Label)
			runtimeLabel := box.Objects[3].(*widget.Label)
			runBtn := box.Objects[4].(*widget.Button)
			delBtn := box.Objects[5].(*widget.Button)

			title := ld.LauncherFiles[id].Title
			if title == "" {
				title = filepath.Base(path)
			}
			name.SetText(title)

			if ld.LauncherFiles[id].Icon != "" {
				iconImg.File = absPath(ld.LauncherFiles[id].Icon)
				iconImg.Hidden = false
				iconImg.Refresh()
			} else {
				iconImg.File = ""
				iconImg.Hidden = true
			}

			var displayed string
			if p, ok := running[path]; ok {
				displayed = formatRuntime(ld.LauncherFiles[id].RuntimeMs + time.Since(p.start).Milliseconds())
			} else if ld.LauncherFiles[id].RuntimeMs > 0 {
				displayed = formatRuntime(ld.LauncherFiles[id].RuntimeMs)
			}
			runtimeLabel.SetText(displayed)

			launch := func() {
				if isExecutable(path) {
					if err := startProcess(path); err != nil {
						dialog.ShowError(err, w)
						return
					}
					fileList.Refresh()
				} else if err := openFile(path); err != nil {
					dialog.ShowError(err, w)
				}
			}

			if _, ok := running[path]; ok {
				runBtn.Importance = widget.DangerImportance
				runBtn.SetText("Stop")
				runBtn.OnTapped = func() {
					stopProcess(path)
					fileList.Refresh()
				}
			} else {
				runBtn.Importance = widget.MediumImportance
				runBtn.SetText("Run")
				runBtn.OnTapped = launch
			}

			row.onDouble = func() {
				if _, ok := running[path]; ok {
					return
				}
				launch()
			}

			delBtn.OnTapped = func() {
				dialog.ShowConfirm("Confirm Delete",
					fmt.Sprintf("Delete \"%s\"?", filepath.Base(path)),
					func(ok bool) {
						if !ok {
							return
						}
						stopProcess(path)
						if item := ld.LauncherFiles[id]; item.Icon != "" {
							_ = os.Remove(absPath(item.Icon))
						}
						newItems := make([]LauncherItem, 0, len(ld.LauncherFiles)-1)
						for i, item := range ld.LauncherFiles {
							if i != id {
								newItems = append(newItems, item)
							}
						}
						ld.LauncherFiles = newItems
						saveLauncherData(ld)
						fileList.Refresh()
					}, w)
			}
		},
	)

	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for range tick.C {
			fyne.Do(func() { fileList.Refresh() })
		}
	}()

	addBtn := widget.NewButton("Add File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			path := normalizePath(reader.URI().Path())
			for _, item := range ld.LauncherFiles {
				if absPath(item.Path) == path {
					dialog.ShowInformation("Notice", "File is already in the list", w)
					return
				}
			}
			ld.LauncherFiles = append(ld.LauncherFiles, LauncherItem{Path: toStoredPath(path), Title: filepath.Base(path), Icon: writeIcon(path)})
			saveLauncherData(ld)
			fileList.Refresh()
		}, w)
	})

	content := container.NewBorder(
		addBtn,
		nil, nil, nil,
		fileList,
	)
	w.SetContent(content)

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		added := false
		for _, uri := range uris {
			path := normalizePath(uri.Path())
			if path == "" {
				continue
			}
			dup := false
			for _, item := range ld.LauncherFiles {
				if absPath(item.Path) == path {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			ld.LauncherFiles = append(ld.LauncherFiles, LauncherItem{Path: toStoredPath(path), Title: filepath.Base(path), Icon: writeIcon(path)})
			added = true
		}
		if added {
			saveLauncherData(ld)
			fileList.Refresh()
		}
	})

	w.Resize(fyne.NewSize(640, 400))
	w.ShowAndRun()
}
