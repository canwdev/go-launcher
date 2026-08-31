# go-launcher

A minimal [Wails](https://wails.io/) desktop launcher demo app for Go. It opens a
window listing files/programs (with icons and accumulated runtime), letting you
run/stop them, drag & drop to add more, and manage entries via a per-item menu.

## Requirements

- Go 1.26 or later
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Windows**: WebView2 runtime (preinstalled on Windows 10/11). No C compiler needed.

## Run (development)

```sh
wails dev
```

## Build

```sh
wails build
# or, on Windows:
.\build.ps1
```

The binary is produced at `build/bin/go-launcher.exe` and copied to `go-launcher.exe`.

## Structure

- `main.go` — Wails app bootstrap (`wails.Run`, embedded frontend assets)
- `app.go` — the `App` struct with the methods bound to the frontend (add/remove/rename, run/stop, icons, etc.) plus the launcher data persistence
- `frontend/` — plain HTML + CSS + JS UI (no build step or framework)
- `utils.go`, `launch_*.go`, `icon_*.go` — platform helpers (path normalization, process launching, icon extraction)

## References

- [Wails documentation](https://wails.io/docs/introduction)
- [Wails on GitHub](https://github.com/wailsapp/wails)
