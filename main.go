package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type LauncherItem struct {
	Path      string `json:"path"`
	RuntimeMs int64  `json:"runtime_ms"`
}

type LauncherData struct {
	LauncherFiles []LauncherItem `json:"launcher_files"`
}

type runningProc struct {
	cmd   *exec.Cmd
	start time.Time
}

const saveFile = "go_launcher_data.json"

var running = map[string]*runningProc{}

func loadLauncherData() LauncherData {
	data, err := os.ReadFile(saveFile)
	if err != nil {
		return LauncherData{}
	}
	var ld LauncherData
	if err := json.Unmarshal(data, &ld); err != nil {
		return LauncherData{}
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
	s := total % 60
	parts := make([]string, 0, 4)
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	parts = append(parts, fmt.Sprintf("%ds", s))
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
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "", `"`+path+`"`)
		return cmd.Start()
	case "darwin":
		cmd := exec.Command("open", path)
		return cmd.Start()
	default:
		cmd := exec.Command("xdg-open", path)
		return cmd.Start()
	}
}

func main() {
	a := app.New()
	w := a.NewWindow("Go Launcher")

	ld := loadLauncherData()

	var fileList *widget.List

	startProcess := func(path string) error {
		cmd := exec.Command(path)
		if err := cmd.Start(); err != nil {
			return err
		}
		p := &runningProc{cmd: cmd, start: time.Now()}
		running[path] = p
		go func() {
			_ = cmd.Wait()
			elapsed := time.Since(p.start).Milliseconds()
			fyne.Do(func() {
				for i := range ld.LauncherFiles {
					if ld.LauncherFiles[i].Path == path {
						ld.LauncherFiles[i].RuntimeMs += elapsed
						break
					}
				}
				delete(running, path)
				saveLauncherData(ld)
				fileList.Refresh()
			})
		}()
		return nil
	}

	stopProcess := func(path string) {
		if p, ok := running[path]; ok {
			_ = p.cmd.Process.Kill()
		}
	}

	fileList = widget.NewList(
		func() int { return len(ld.LauncherFiles) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("placeholder")
			runtime := widget.NewLabel("")
			runBtn := widget.NewButton("Run", nil)
			delBtn := widget.NewButton("Delete", nil)
			return container.NewHBox(name, layout.NewSpacer(), runtime, runBtn, delBtn)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			path := ld.LauncherFiles[id].Path
			box := obj.(*fyne.Container)
			name := box.Objects[0].(*widget.Label)
			runtimeLabel := box.Objects[2].(*widget.Label)
			runBtn := box.Objects[3].(*widget.Button)
			delBtn := box.Objects[4].(*widget.Button)

			name.SetText(filepath.Base(path))

			displayed := formatRuntime(ld.LauncherFiles[id].RuntimeMs)
			if p, ok := running[path]; ok {
				displayed = formatRuntime(ld.LauncherFiles[id].RuntimeMs + time.Since(p.start).Milliseconds())
			}
			runtimeLabel.SetText(displayed)

			if _, ok := running[path]; ok {
				runBtn.SetText("Stop")
				runBtn.Importance = widget.DangerImportance
				runBtn.OnTapped = func() { stopProcess(path) }
			} else {
				runBtn.SetText("Run")
				runBtn.Importance = widget.MediumImportance
				runBtn.OnTapped = func() {
					if isExecutable(path) {
						if err := startProcess(path); err != nil {
							dialog.ShowError(err, w)
						}
					} else if err := openFile(path); err != nil {
						dialog.ShowError(err, w)
					}
				}
			}

			delBtn.OnTapped = func() {
				dialog.ShowConfirm("Confirm Delete",
					fmt.Sprintf("Delete \"%s\"?", filepath.Base(path)),
					func(ok bool) {
						if !ok {
							return
						}
						stopProcess(path)
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
		tick := time.NewTicker(time.Second)
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
			path := reader.URI().Path()
			for _, item := range ld.LauncherFiles {
				if item.Path == path {
					dialog.ShowInformation("Notice", "File is already in the list", w)
					return
				}
			}
			ld.LauncherFiles = append(ld.LauncherFiles, LauncherItem{Path: path})
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
	w.Resize(fyne.NewSize(640, 400))
	w.ShowAndRun()
}
