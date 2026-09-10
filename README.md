# go-launcher

<p align="center">
  <img src="build/appicon.png" alt="icon" width="180" height="180" />
  <br>
  <img src="docs/screenshot.webp" alt="go launcher"  />
</p>


A minimal [Wails](https://wails.io/) desktop launcher app for Go. It opens a
window listing files/programs (with icons and accumulated runtime), letting you
run/stop them, drag & drop to add more, and manage entries via a per-item menu.
Each item can be launched as administrator (check *Run as administrator* in the item
editor — it is stored in the data file and shows a shield marker; Windows asks for UAC).
Launcher data lives in `.go-launcher-data/go-launcher-data.json`; edit that file by
hand and the running app reloads it automatically (see [Data file](#data-file)).

## Requirements

- Go 1.26 or later
- [bun](https://bun.sh) (package manager for the frontend)
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Windows**: WebView2 runtime (preinstalled on Windows 10/11). No C compiler needed.

## Run (development)

```sh
wails dev
```

This starts the Vite dev server and rebuilds/relaunches the app on changes.

## Build

```sh
wails build
# or, on Windows:
.\build.ps1
```

The binary is produced at `build/bin/go-launcher.exe` and copied to `go-launcher.exe`.

## Releases

Every push to `master` runs [`.github/workflows/release.yml`](./.github/workflows/release.yml)
on `windows-latest`: it installs the Wails CLI, builds the frontend with bun, runs
`wails build`, then publishes a GitHub Release tagged
`v<version from frontend/package.json>-<run number>` (e.g. `v0.1.2-42`) with `go-launcher.exe`
attached (Windows x64 only — this app is Wails + WebView2, no cross-platform build). Bump
`version` in `frontend/package.json` when you want a new release version. Manual runs are
possible from the Actions tab (`workflow_dispatch`).

## Data file

Everything the launcher knows is kept in `.go-launcher-data/go-launcher-data.json`
(next to the working directory of the executable):

- `apps` — global item pool (`guid` / `name` / `path` / `icon` / `args` / `working_dir` / `runtime_ms` / `run_as_admin`)
- `categories` — the tabs, each with a `slots` array of app guids (`null` = empty grid cell)
- `settings` — `game_mode` / `auto_hide` / `always_on_top` / `absolute_paths`

The file is watched while the app runs (a Windows directory watch — no polling): save a manual
edit and the launcher applies it right away and refreshes the window. A file that is not valid
JSON is ignored (the current data stays loaded) and a warning is shown; deleting the file is also
ignored, and removing an item stops tracking it. `Refresh` in the app menu re-reads the file
immediately (on every platform), and the store is written back on every change and on exit.

## Structure

- `main.go` — Wails app bootstrap (`wails.Run`, embeds `frontend/dist`)
- `app.go` — the `App` struct with the methods bound to the frontend (add/remove/rename, run/stop, icons, etc.) plus the launcher data persistence
- `store_watch.go`, `store_watch_windows.go`, `store_watch_other.go` — watch `go-launcher-data.json` for manual (external) edits and reload the store
- `frontend/` — **Vite + Vue 3 + TypeScript + Tailwind CSS + Headless UI** frontend, managed with bun
  - `src/api.ts` — typed wrappers around the generated `wailsjs` Go bindings
  - `src/composables/useStore.ts` — reactive item list, store persistence, Wails events & file drop
  - `src/components/` — `LauncherRow` / `GridItem` item views, `AppDialog` (Headless UI `Dialog`), `ItemMenu`, `TabBar`, dialogs
  - `eslint.config.mjs` — [@antfu/eslint-config](https://github.com/antfu/eslint-config)
  - scripts: `bun run dev` / `build` / `typecheck` / `lint` / `lint:fix`
- `utils.go`, `launch_*.go`, `icon_*.go` — platform helpers (path normalization, process launching, icon extraction)

## References

- [Icon generation](./docs/ICON.md)
- [Wails documentation](https://wails.io/docs/introduction)
- [Wails on GitHub](https://github.com/wailsapp/wails)
