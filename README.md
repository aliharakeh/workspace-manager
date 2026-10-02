# Workspace Manager

A local desktop tool for running and managing multiple apps from one place. Group projects into workspaces, give each app its own config (env vars, file templates, start commands), then run, stop, and restart them while watching live logs.

A [Wails](https://wails.io/) v2 desktop app: Go owns the window, SQLite (sqlc), process runner, and OS integration. The React UI in `frontend/` talks to it through generated Wails bindings (`frontend/host.ts`) and Wails events (`runnerEvent`). File/folder pickers and opening URLs use the Wails runtime.

The database file lives in the OS user-data directory: Windows `%LOCALAPPDATA%\workspace-manager`, macOS `~/Library/Application Support/workspace-manager`, Linux `$XDG_DATA_HOME/workspace-manager` or `~/.local/share/workspace-manager`.

## Features

### Workspaces and apps

- **Workspaces** — named groups of apps (optional icon), managed from the sidebar
- **Apps** — name + local project folder, picked with a native folder dialog
- **Live status** — running / idle indicators, plus Run / Stop / Reload on the app and workspace views
- **Open in editor** — opens the app’s project path in your local editor (`$VISUAL` / `$EDITOR`, or the OS file manager)
- **Blueprints** — reusable recipes for new apps, managed under **Settings → Blueprints** and usable in any workspace: a list of shell commands (with `{{app_name}}`, `{{folder_name}}`, `{{app_dir}}`). **From blueprint** next to **Add app** asks for a parent folder and the app name, then runs the commands either inside a new app folder or in the parent (your choice), streams the output, runs `git init` unless a repo exists, and adds the app

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

Stdout and stderr stream as they arrive over Wails events. ANSI codes are stripped. stdout and stderr are split per process tab, with system lines (commands, exits, template apply/restore) inline.

### Ready URLs

Log lines are matched against configurable regex patterns (named `url` / `port` groups) so Vite, Next.js, Spring Boot, .NET, Django, and similar servers show up as clickable links on the app, workspace, and logs. Defaults are seeded; you can add or edit patterns under Settings → Log URL patterns.

### Settings and UX

- **Listening ports** — list local listeners (PID, port, name) and kill one from Settings
- **Command palette** — search workspaces and apps (`Ctrl+P` by default)
- **Keyboard shortcuts** — rebind the palette and theme toggle
- **Deep linking** — workspace / app / tab / config set stay in the URL
- **Theme** — light, dark, or system

## Run

Needs [Go](https://go.dev/), [Wails](https://wails.io/) v2 and [Bun](https://bun.sh/). From the repo root:

```bash
wails dev                # Wails + Vite (port 5174)
wails build              # production native app → build/bin (builds for the OS you run it on)
wails generate module    # regenerate frontend/wailsjs after changing bound Go methods or types
sqlc generate            # regenerate db after editing schema.sql / queries.sql
go test ./db/... ./lib/... ./native/... ./services/...
```

Frontend scripts (`bun install` once in `frontend/`): `bun run dev`, `bun run build`, `bun run lint`, `bun run typecheck`.

## Layout

| Path | Role |
|---|---|
| `main.go`, `app.go`, `bind.go`, `events.go` | Wails entry point, `App` lifecycle, bound API methods, event emission |
| `services/` | Runner, templates, blueprints, ready URLs, notifier, AI |
| `lib/`, `native/` | fs / env / import helpers; OS helpers (process, ports, editor, browser) |
| `db/` | SQLite schema, queries, sqlc-generated code |
| `types/` | Types shared with the UI (JSON tags match `frontend/lib/types.ts`) |
| `frontend/` | React UI (Vite); `host.ts` is the Wails adapter, `wailsjs/` is generated |
| `build/` | Wails packaging assets |
