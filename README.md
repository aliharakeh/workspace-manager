# Workspace Manager

A local desktop tool for running and managing multiple apps from one place. Group projects into workspaces, give each app its own config (env vars, file templates, start commands), then run, stop, and restart them while watching live logs.

Same product, three shells:

| | **bun_backend** (`bun_backend/`) | **electrobun_backend** (`electrobun_backend/`) | **wails_backend** (`wails_backend/`) |
|---|---|---|---|
| What it is | Compiled **Bun** server + React UI | **Electrobun** desktop app | **Wails** v2 desktop app (Go) |
| How you use it | HTTP server in the browser (or a standalone compiled app) | Native window; Bun is the main process | Native window; Go is the main process |
| Native work | Shared helpers under `native/` | Same `native/` helpers, plus Electrobun dialogs | Go ports in `wails_backend/internal/native/` (Wails dialogs / browser) |
| UI | Shared `frontend/` + `bun_backend/host.ts` (`fetch` / SSE) | Shared `frontend/` + `electrobun_backend/src/host.ts` (RPC) | Shared `frontend/` + `wails_backend/frontend/host.ts` (Wails bindings) |
| Live logs | Server-Sent Events (`EventSource`) | Electrobun RPC (`runnerEvent`) | Wails events (`runnerEvent`) |
| SQLite | Drizzle (`db/`) | Drizzle (`db/`) | sqlc mapped from the same Drizzle schema |
| Ship it | `bun run bun_backend:compile:{win,mac,linux}` | `bun run electrobun_backend:build:stable` on that OS | `bun run wails_backend:build` on that OS |

All three share one React UI in `frontend/` and one SQLite database. Each shell provides a small host adapter so the UI talks to one `api` surface.

The database file lives in the OS user-data directory (same path for every shell): Windows `%LOCALAPPDATA%\workspace-manager`, macOS `~/Library/Application Support/workspace-manager`, Linux `$XDG_DATA_HOME/workspace-manager` or `~/.local/share/workspace-manager`.

## bun_backend vs electrobun_backend vs wails_backend

**bun_backend** is a Bun HTTP server that serves the shared React app and a JSON/SSE API. In development it sits next to Vite; in production it serves the built frontend from the same origin. `bun run bun_backend:compile` embeds the UI and Drizzle migrations and produces a standalone binary (cross-compile with `bun_backend:compile:win` / `bun_backend:compile:mac` / `bun_backend:compile:linux`). Native bits (folder pickers, process spawn/kill, ports, opening the browser) live in shared `native/` and are invoked by the server. The UI reaches them through `bun_backend/host.ts` (`fetch` + `EventSource`).

**electrobun_backend** is an [Electrobun](https://blackboard.sh/electrobun/docs/) app (not Electron). The Bun main process owns the window, SQLite, process runner, and OS integration. The same `frontend/` UI talks to it through `electrobun_backend/src/host.ts` (Electrobun RPC) instead of `fetch` / SSE. Shared native helpers (spawn, kill, ports, open-in-editor) live in `native/`; Electrobun file/folder dialogs stay in `electrobun_backend/src/bun/native/`.

**wails_backend** is a [Wails](https://wails.io/) v2 app. Go owns the window, SQLite (sqlc), process runner, and OS integration. The same `frontend/` UI talks to it through `wails_backend/frontend/host.ts` (generated Wails bindings + `EventsOn("runnerEvent")`). Native helpers are Go versions of `native/` under `wails_backend/internal/native/`; file/folder pickers and opening URLs use the Wails runtime.

## Updating one shell without drifting the others

You can work on a single backend, but anything the UI or the database cares about has to land in **all** shells that implement it.

| You changed… | Also update… |
|---|---|
| Shared UI only (`frontend/`, no `api` change) | Nothing else |
| `api` shape / host methods | `bun_backend/host.ts`, `electrobun_backend/src/host.ts`, `wails_backend/frontend/host.ts`, plus that shell’s routes / RPC / `bind.go` |
| Runner, templates, ready URLs, fs/env import | `bun_backend/server/`, `electrobun_backend/src/bun/`, `wails_backend/internal/` |
| JS native helpers (`native/*.ts`) | The matching file in `wails_backend/internal/native/` (and Electrobun dialogs if you touched pickers) |
| Drizzle schema or `db/*.ts` queries | `bun run db:generate`, then copy new `drizzle/*.sql` into `wails_backend/internal/db/migrations/`, update `wails_backend/internal/db/schema.sql` + `queries.sql`, run `bun run wails_backend:sqlc` |
| Wails bound methods or Go JSON types | `cd wails_backend && wails generate module` so `frontend/wailsjs/` matches |

JS shells share `db/` at runtime. Wails applies the same Drizzle SQL (hash-compatible `__drizzle_migrations`) so all three can open the same user-data SQLite file.

## Features

### Workspaces and apps

- **Workspaces** — named groups of apps (optional icon), managed from the sidebar
- **Apps** — name + local project folder, picked with a native folder dialog
- **Live status** — running / idle indicators, plus Run / Stop / Reload on the app and workspace views
- **Open in editor** — opens the app’s project path in your local editor (`$VISUAL` / `$EDITOR`, or the OS file manager)

### Config sets

Each app has named **config sets** (e.g. dev, staging, prod). One is active at a time. A set bundles:

- environment variables
- file templates
- run commands

You can switch, rename, or delete sets (the last one stays). You can also copy another set into a new or existing one and pick exactly which env vars, templates, or commands to take.

### Environment variables

- Key/value pairs on the active config set
- Injected into every process that set starts
- Import from a `.env` or `.yaml` / `.yml` file via the native file picker (YAML nests flatten to dot-notation keys)

### Templates

Handlebars templates written over project files when you hit **Run**. Originals are backed up and restored on **Stop** or exit.

- Paths stay inside the project directory
- Theme-aware editor with syntax highlighting (TS, JS, JSON, CSS, HTML, Python, YAML, …), with Handlebars `{{var}}` still visible

### Run config

- Multiple labeled commands per config set
- **Parallel** (default) or **sequential** (stops on the first non-zero exit)
- **Run / Stop / Reload** — one session per app; stop kills the process tree; reload restarts it
- Per-process tabs in the logs panel (pending / running / exited / killed / error)

### Live logs

Stdout and stderr stream as they arrive. bun_backend uses SSE; electrobun_backend and wails_backend push the same events over RPC / Wails events. ANSI codes are stripped. stdout and stderr are split per process tab, with system lines (commands, exits, template apply/restore) inline.

### Ready URLs

Log lines are matched against configurable regex patterns (named `url` / `port` groups) so Vite, Next.js, Spring Boot, .NET, Django, and similar servers show up as clickable links on the app, workspace, and logs. Defaults are seeded; you can add or edit patterns under Settings → Log URL patterns.

### Settings and UX

- **Listening ports** — list local listeners (PID, port, name) and kill one from Settings
- **Command palette** — search workspaces and apps (`Ctrl+P` by default)
- **Keyboard shortcuts** — rebind the palette and theme toggle
- **Deep linking** — workspace / app / tab / config set stay in the URL
- **Theme** — light, dark, or system

## Run

From the repo root, `bun install` once (one `package.json`, one `node_modules`). Then:

**bun_backend**

```bash
bun run bun_backend:dev              # API + Vite
bun run bun_backend:start            # production (build + serve)
bun run bun_backend:compile          # this machine → bun_backend/release/workspace-manager(.exe)
bun run bun_backend:compile:win      # Windows x64 → bun_backend/release/workspace-manager-windows.exe
bun run bun_backend:compile:mac      # macOS arm64 → bun_backend/release/workspace-manager-macos
bun run bun_backend:compile:linux    # Linux x64 → bun_backend/release/workspace-manager-linux
```

**electrobun_backend**

Electrobun packages the OS you run the command on (no cross-compile).

```bash
bun run electrobun_backend:dev:hmr          # Electrobun + Vite HMR
bun run electrobun_backend:dev              # Electrobun without HMR
bun run electrobun_backend:build:stable     # production build for this machine
```

**wails_backend**

Needs [Wails](https://wails.io/) v2 and Go. Packages the OS you run the command on (no cross-compile).

```bash
bun run wails_backend:dev       # Wails + Vite (port 5174)
bun run wails_backend:build     # production native app → wails_backend/build/bin
bun run wails_backend:sqlc      # regenerate Go from sqlc after schema/query edits
```

| | **bun_backend** (`bun run bun_backend:compile:win`) | **electrobun_backend** / **wails_backend** |
|---|---|---|
| How the OS is chosen | Bun `--target` (cross-compile) | Host OS |
| Build Windows from a Mac | Yes | No — run the command on Windows |
| Build Mac from Windows | Yes | No — run it on a Mac |
