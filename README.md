# go-launcher

A minimal [Gio](https://gioui.org/) desktop launcher for Go. It opens a window
that lists executable files (from `go-launcher-data.json`) and lets you run,
stop, and manage them.

## Requirements

- Go 1.23 or later
- No C compiler required on Windows (Gio uses the Direct3D11 backend).

## Run

```sh
go run .
```

## Build

```sh
go build
```

Production build (Windows GUI subsystem, no console window):

```sh
powershell -File build.ps1
```

## Overview

`main.go` holds the data model and persistence logic; `gui.go` is the Gio
user interface.

- `runGUI()` creates the window (`gioui.org/app`) and runs the event loop,
  rendering the list of launcher files each frame.
- Rows show the app icon, title, accumulated runtime, a Run/Stop button and a
  "…" menu (open containing folder, rename, update icon, details, delete).
- Double-click a row to launch it; runtime is tracked per process and saved to
  `go-launcher-data.json` when a process exits or the window closes.
- Dialogs (rename, details, confirm, error) are rendered in-app with an overlay
  stack; the details dialog can copy its JSON to the clipboard.

Platform-specific support lives in `icon_windows.go`/`icon_other.go`
(extracting file icons) and `launch_windows.go`/`launch_other.go` (launching
processes).

## References

- [Gio documentation](https://gioui.org/doc)
- [Gio on sourcehut](https://git.sr.ht/~eliasnaur/gio)
