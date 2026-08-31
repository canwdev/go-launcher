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
	"strings"
	"time"

	_ "golang.org/x/image/bmp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
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

var removed = map[string]bool{}

var appModified = map[string]bool{}

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
			menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
			return &tapRow{CanvasObject: container.NewHBox(iconImg, name, layout.NewSpacer(), runtime, runBtn, menuBtn)}
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			path := absPath(ld.LauncherFiles[id].Path)
			row := obj.(*tapRow)
			box := row.CanvasObject.(*fyne.Container)
			iconImg := box.Objects[0].(*canvas.Image)
			name := box.Objects[1].(*widget.Label)
			runtimeLabel := box.Objects[3].(*widget.Label)
			runBtn := box.Objects[4].(*widget.Button)
			menuBtn := box.Objects[5].(*widget.Button)

			title := ld.LauncherFiles[id].Title
			if title == "" {
				title = filepath.Base(path)
			}
			name.SetText(title)

			if ld.LauncherFiles[id].Icon != "" {
				iconImg.File = absPath(ld.LauncherFiles[id].Icon)
				iconImg.Resource = nil
				iconImg.Image = nil
				iconImg.Hidden = false
				iconImg.Refresh()
			} else {
				iconImg.File = ""
				iconImg.Resource = nil
				iconImg.Image = nil
				iconImg.Hidden = false
				iconImg.Refresh()
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

			menuBtn.OnTapped = func() {
				rename := func() {
				entry := widget.NewEntry()
				entry.SetText(title)
				entry.CursorColumn = len([]rune(title))
					d := dialog.NewForm("Rename", "OK", "Cancel",
						[]*widget.FormItem{widget.NewFormItem("Name", entry)},
						func(ok bool) {
							if !ok {
								return
							}
							newTitle := strings.TrimSpace(entry.Text)
							if newTitle == "" {
								return
							}
							ld.LauncherFiles[id].Title = newTitle
							appModified[itemKey(ld.LauncherFiles[id])] = true
							saveLauncherData(ld)
							fileList.Refresh()
						}, w)
					d.Resize(fyne.NewSize(400, d.MinSize().Height))
					d.Show()
					w.Canvas().Focus(entry)
				}

				changeIcon := func() {
					fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						if reader == nil {
							return
						}
						defer reader.Close()
						img, _, err := image.Decode(reader)
						if err != nil {
							dialog.ShowError(fmt.Errorf("unsupported image: %v", err), w)
							return
						}
						if old := ld.LauncherFiles[id].Icon; old != "" {
							_ = os.Remove(absPath(old))
						}
						icon := saveImage(img, filepath.Base(path)+"-custom")
						if icon == "" {
							dialog.ShowError(fmt.Errorf("failed to save icon"), w)
							return
						}
						ld.LauncherFiles[id].Icon = icon
						appModified[itemKey(ld.LauncherFiles[id])] = true
						saveLauncherData(ld)
						fileList.Refresh()
					}, w)
					fd.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}))
					fd.Show()
				}

				details := func() {
					b, err := json.MarshalIndent(ld.LauncherFiles[id], "", "  ")
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					entry := widget.NewEntry()
					entry.SetText(string(b))
					entry.MultiLine = true
					entry.Wrapping = fyne.TextWrapWord
					copyBtn := widget.NewButton("Copy to Clipboard", func() {
						w.Clipboard().SetContent(string(b))
					})
					d := dialog.NewCustom("Details", "Close",
						container.NewBorder(nil, copyBtn, nil, nil, entry), w)
					d.Resize(fyne.NewSize(460, 320))
					d.Show()
				}

				pop := widget.NewPopUpMenu(fyne.NewMenu("",
					fyne.NewMenuItem("Open containing folder", func() {
						if err := revealFile(path); err != nil {
							dialog.ShowError(err, w)
						}
					}),
					fyne.NewMenuItem("Rename", rename),
					fyne.NewMenuItem("Change icon", changeIcon),
					fyne.NewMenuItem("Update icon", func() {
						icon := writeIcon(path)
						if icon == "" {
							dialog.ShowError(fmt.Errorf("failed to regenerate icon"), w)
							return
						}
						if old := ld.LauncherFiles[id].Icon; old != "" {
							_ = os.Remove(absPath(old))
						}
						ld.LauncherFiles[id].Icon = icon
						appModified[itemKey(ld.LauncherFiles[id])] = true
						saveLauncherData(ld)
						fileList.Refresh()
					}),
					fyne.NewMenuItem("Details", details),
					fyne.NewMenuItemSeparator(),
					fyne.NewMenuItem("Delete", func() {
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
								removed[itemKey(ld.LauncherFiles[id])] = true
								delete(appModified, itemKey(ld.LauncherFiles[id]))
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
					}),
				), w.Canvas())
				pop.ShowAtRelativePosition(fyne.NewPos(0, menuBtn.Size().Height), menuBtn)
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

	addBtn := widget.NewButton("Add or Drag & Drop File", func() {
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
			ld.LauncherFiles = append(ld.LauncherFiles, LauncherItem{Path: toStoredPath(path), Title: defaultTitle(path), Icon: writeIcon(path)})
			delete(removed, itemKey(LauncherItem{Path: toStoredPath(path)}))
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
			ld.LauncherFiles = append(ld.LauncherFiles, LauncherItem{Path: toStoredPath(path), Title: defaultTitle(path), Icon: writeIcon(path)})
			delete(removed, itemKey(LauncherItem{Path: toStoredPath(path)}))
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
