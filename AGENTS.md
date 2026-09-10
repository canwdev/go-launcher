# AGENTS.md

Guidance for agents working in this repository. Keep it short and true; update it when
conventions change.

## What this is

A minimal [Wails v2](https://wails.io/) desktop launcher for Windows (with `_other.go`
stubs so the module still compiles on other platforms). A Go backend exposes methods on
`App` to a Vite + Vue 3 + TypeScript + Tailwind + Headless UI frontend. The launcher lists
files/programs with icons and accumulated runtime, and can run/stop them.

## Toolchain

- Go 1.26+ (`go.mod` targets `go 1.26.2`)
- [bun](https://bun.sh) for the frontend — `bun install`, `bun run build`. **bun.lock is
  the lockfile; do not add a package-lock.json / pnpm-lock.yaml.**
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows: WebView2 runtime, no C compiler needed. UI manifest: `build/windows/wails.exe.manifest`.

## Commands

```sh
wails dev                      # dev: Vite HMR + rebuild on Go changes
wails build -clean             # production build (embeds frontend/dist)
.\build.ps1                    # same build, then copies build/bin/go-launcher.exe to repo root

cd frontend
bun run build                  # vite build -> frontend/dist (must run before a Go build)
bun run typecheck              # vue-tsc --noEmit
bun run lint                   # eslint (@antfu/eslint-config); lint:fix to autofix

go build ./...                 # verify Go compiles
go test ./...                  # Go tests (repo root package only)
```

`wails.json` wires `frontend:install`/`frontend:build`/`frontend:dev:watcher` to bun, so
`wails dev` and `wails build` install/build the frontend for you.

## Critical gotchas

- **`frontend/dist` is embedded** via `//go:embed all:frontend/dist` in `main.go`. After any
  frontend `src/` change you must run `bun run build` before building/verifying the Go binary,
  otherwise the UI changes are not in the app. `frontend/dist` is gitignored.
- **The Wails bindings in `frontend/wailsjs/` are generated** (from exported `App` methods).
  Adding/renaming an exported Go method requires regenerating them — `wails dev`/`wails build`
  do this; in Wails v2 there is no separate bindings generator, so use `wails build` (or
  `wails dev`) to regenerate them. Then update `frontend/src/api.ts`,
  which re-exports the bindings as the app's typed API.
- **Runtime data lives in `.go-launcher-data/`** (hidden folder next to the executable's
  working directory), not in the program directory:
  - `.go-launcher-data/go-launcher-data.json` — the store (`AppStore`: apps / categories / settings)
  - `.go-launcher-data/window-state.json` — persisted window geometry
  - `.go-launcher-data/cached-icons/` — extracted icon PNGs (`iconsDir`)
  The folder name is the `dataDir` constant in `app.go` and is duplicated as a path check in
  `frontend/src/utils.ts` (`isAutoIcon`) and in the root ignore file. If you rename it, update
  all three and expect existing stores to hold stale icon paths — no automatic migration is done.
- Paths persisted in the store are relative to the working directory (`toStoredPath` /
  `absPath` in `app.go`), unless the "Abs path" setting makes them absolute. `convertItemPaths`
  converts between the two; keep that round-trip consistent when touching item fields.
- **Per-item "Run as administrator"**: `AppItem.RunAsAdmin` (JSON `run_as_admin`, `omitempty`, set
  by the checkbox in `ItemEditDialog.vue`) is threaded as an `elevated bool` through
  `openFile` → `startTracked` / `startUntracked` → `startProcess`, which on Windows picks the
  `runas` ShellExecuteEx verb (`launchVerb`); a cancelled UAC prompt surfaces as
  "elevation was cancelled". Without the flag the old behaviour is unchanged (`open` first,
  `runas` fallback). `launch_other.go` ignores the flag (Windows-only concept) — keep the
  signatures in both files in sync. Items show a shield marker in `LauncherRow` / `GridItem`.
- **The store file is watched for manual edits** (`store_watch.go` + `store_watch_windows.go`):
  a Windows directory watch (`ReadDirectoryChangesW`, async + event-driven — no polling) on
  `.go-launcher-data` notices changes to `go-launcher-data.json`, waits out a short quiet window
  (150 ms) to coalesce one save's event burst, then reloads the file into memory and emits
  `store:reloaded` (frontend replaces its whole store — see `useStore.ts`, which also cancels a
  pending debounced save) or `store:reload-failed` for invalid JSON (the in-memory store is
  kept; a deleted file is ignored rather than wiping data). App writes are recognised by the
  fingerprint (mtime/size + content hash) recorded in `writeStore`, so **every store write must
  go through `writeStore`** (which assumes `a.mu` is held) or the watcher will re-read them.
  `store_watch_other.go` is the non-Windows no-op (the module still builds there); `GetData` runs
  the same fingerprint check, so `Refresh` re-reads the file immediately on any platform. Parse
  and default logic is shared with `loadStore` via `parseStore`.
- `main.go` is the only file that calls `wails.Run`; keep app wiring there and logic in `app.go`.

## Layout

Root Go package (`package main`):

| File | Role |
| --- | --- |
| `main.go` | Wails bootstrap, window options, embeds `frontend/dist` |
| `app.go` | `App` struct + all frontend-bound methods (add/remove/rename, run/stop, icons, store persistence), `dataDir` constants |
| `store_watch.go` | store file fingerprint (mtime/size + hash) + reload/event plumbing for manual edits |
| `store_watch_windows.go` / `store_watch_other.go` | `ReadDirectoryChangesW` directory watch / non-Windows no-op |
| `utils.go` | shared helpers (path normalization, name sanitizing, etc.) |
| `launch_windows.go` / `launch_other.go` | process launching / stop |
| `icon_windows.go` / `icon_other.go` | icon extraction and image saving |
| `lnk_windows.go` / `lnk_other.go` | `.lnk` shortcut parsing |
| `singleton_windows.go` / `singleton_other.go` / `singleton_key.go` | single-instance guard |
| `window_state_windows.go` / `_other.go` / `window_state.go` | window geometry persistence |
| `*_test.go` | Go tests (`singleton_key_test.go`, `window_state_test.go`, `store_test.go`, `store_watch_test.go`, `store_watch_windows_test.go`, `launch_windows_test.go`) |

Frontend (`frontend/`):

- `src/App.vue` — root component: state wiring, global menu callbacks, layout
- `src/api.ts` — re-exports the generated `wailsjs` bindings as the typed API
- `src/menuConfig.ts` — global (app) menu and add-menu definitions, with their ctx interface
- `src/components/` — `LauncherRow`, `GridItem`, `TabBar`, `SearchOverlay`, dialogs, item menu
- `src/composables/` — one concern per file (`useStore`, `useSearch`, `useTheme`, drag, timers,
  dialogs, toast, …); `itemMenu.ts` builds per-item menu entries
- `src/utils.ts` — pure helpers (`isAutoIcon`, `debounce`, …)
- `eslint.config.mjs` — @antfu/eslint-config; VS Code runs `source.fixAll.eslint` on save and
  `editor.formatOnSave` is off, so lint, don't reformat by hand

**Adding a toggle to the global (app) menu** touches four places: the ctx interface + entry in
`src/menuConfig.ts` (`{ toggle: true, checked, action }`), the `buildAppMenu({...})` wiring and
the store setter in `src/App.vue`, and the setting itself. Boolean UI preferences live in
`store.settings` (persisted through `SaveData`), e.g. `auto_hide` / `always_on_top`; window-level
ones can additionally be applied from Go (`app.startup` + a bound method such as
`SetAlwaysOnTop`).

**Context menus**: every menu renders through `components/ItemMenu.vue` (Headless UI `Menu` +
`Teleport` + a fixed overlay). It exposes `open(e?)` so a right-click handler can open it at the
mouse position; keep a component ref and call it from `@contextmenu.prevent`. Headless UI's own
outside-click hook only listens for pointer/mouse/click/touchend, so **right-click does not close
an open menu** — the overlay handles `contextmenu` explicitly (`close()` + `preventDefault`), and
the trigger's `contextmenu` closes the menu while suppressing the click that Chromium synthesizes
afterwards (otherwise it reopens the menu). Keep that behaviour when touching `ItemMenu.vue`.
Item menus come from `buildItemMenu`, grid slot menus from `buildSlotMenu`, tab menus are built
inline in `TabBar.vue` (Rename / Duplicate / Delete, emitted to `App.vue`; `duplicateTab` in
`useStore.ts`). A tab right-click only opens the menu — it must **not** change the active tab, so
the tab element carries `@click.right.prevent` next to `@contextmenu.prevent` (every menu action
targets the clicked tab's guid). Adding a tab action therefore means: entry in `TabBar.vue` + an
`emit` + a `@action` handler on the `<TabBar>` element in `App.vue`.

## Conventions

- Branch `master`; commit subjects follow `feat: ...`, `fix: ...`, `update ...`, `optimize ...`.
  Write descriptive, focused commits; don't bundle unrelated changes.
- Every `master` push publishes a GitHub Release — `.github/workflows/release.yml`
  (`windows-latest`, tag `v<frontend/package.json version>-<run number>`, asset `go-launcher.exe`);
  it runs `bun install --frozen-lockfile` + `bun run build`, then `wails build -clean -s`. Bump
  `version` in `frontend/package.json` to change the release version. The Wails CLI version there
  is pinned to the `go.mod` wails version — bump both together.
- Do not commit runtime data or build output: `.go-launcher-data/`, `build/bin/`,
  `frontend/dist/`, `node_modules/`, `*.exe`, `*.log` (see the repo root `.gitignore`).
- Frontend style is enforced by ESLint (2-space indent, single quotes, no semicolons); run
  `bun run lint` before finishing UI work. `frontend/full_lint.log` is a leftover log.
- `frontend/package.json.md5` is a Wails-generated checksum marker — leave it alone.
- Comments and docs are mixed Chinese/English; match the surrounding file's language.
- Keep the platform split intact: Windows-only code goes in `*_windows.go` with a `*_other.go`
  stub so `go build ./...` still works elsewhere.

## References

- `README.md` — user-facing overview, requirements, structure
- `docs/ICON.md` — icon generation and Windows icon-cache refresh
- `docs/generate-icon.ps1` — must stay UTF-8 **with BOM** (PowerShell 5.1 parsing)
