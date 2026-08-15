# App Runner

A desktop-style local dev runner: configure apps, run multiple commands per project, and manage them from a single UI. React + Vite + shadcn/ui frontend, Bun server with SQLite persistence.

## Structure

```text
src/          React frontend (shadcn)
server/       Bun.serve API + SQLite + process runner + static files
native/       Native OS helpers (spawn, dialogs, ports, browser)
dist/         Production frontend build (served by Bun)
data/         Local SQLite DB and template backups (gitignored)
scripts/      Dev / production / compile launchers
release/      Compiled standalone binary (bun run compile)
```

## Development

API + Vite HMR. Preferred ports `3000` / `5173`; if busy, the next free ports are chosen automatically:

```bash
bun run dev
```

Or separately:

```bash
bun run dev:server
bun run dev:web
```

## Production

Build the frontend and serve it from the Bun server (single process, no Vite):

```bash
bun run start
```

Or build once, then run the server again:

```bash
bun run build
bun run start:server
```

Compile everything (frontend + drizzle migrations embedded) into a single standalone binary:

```bash
bun run compile
```

Open the printed URL (API and UI share the same origin; `/api` is handled by Bun, everything else from `dist/`).

## Features

### Workspaces & apps

- **Workspaces** — named groups of apps, managed from the sidebar
- **Apps** — name + local `project_path`, picked via a native folder dialog or validated on entry
- **Live status** — running/idle dots next to apps in the sidebar and workspace view, plus inline Run / Stop / Reload controls and per-app config-set pickers

### Config sets

- **Config sets** — each app has named sets (e.g. dev, staging, prod), each bundling its own env vars, templates, and run config; one is active at a time
- **Switching / rename / delete** — activate any set from a dropdown; the last set can't be deleted
- **Granular copy** — create a new set or replace parts of the current one from another set; pick exactly which env vars, templates, or run commands to copy, with search filters and select-all per category

### Environment variables

- **Per config set** — key/value pairs with unique keys; the active set's vars are injected into every spawned process
- **Import from `.env`** — pick a file with the native dialog and import (or override) its variables

### Templates

- **Handlebars templates** applied to project files on Run, with automatic backup/restore — original files are restored on Stop or exit
- Files are picked via the native picker and stored per config set; paths are confined to the project directory
- **Theme-aware editor** with syntax highlighting that follows the file's extension (TypeScript, JavaScript, JSON, CSS/SCSS, HTML, Java, Python, YAML, SQL, Bash, …), falling back to Handlebars so `{{var}}` still stands out

### Run config & runner

- **Run config** — per config set: multiple labeled commands, run in **parallel** (default) or **sequential** (stops on first non-zero exit)
- **Run / Stop / Reload** — starts a session per app, stops by killing the whole process tree, and reload restarts the session
- **Per-process tabs** in the logs panel with status badges (pending / running / exited / killed / error)

### Logs

- **SSE streaming** — stdout and stderr stream live over Server-Sent Events with heartbeats and a rolling in-memory buffer; ANSI escape codes are stripped before display
- **stdout / stderr split** — separate columns per process tab, auto-scrolled; system messages (commands, exits, template apply/restore) are shown inline

### Ready URLs

- **Auto-detection** — process log lines are matched against configurable regex patterns (named `url` / `port` groups) to surface live links (Vite, Next.js, Spring Boot, .NET, Django, and more)
- **One-click open** — detected URLs appear as clickable links on apps, workspaces, and in the logs panel
- **Pattern manager** — Settings → Log URL patterns: edit, add, or delete patterns; built-in defaults are seeded automatically

### Settings

- **Listening ports** — inspect all listening processes (with PID, port, and process name) and kill one directly from Settings → Listening ports
- **Keyboard shortcuts** — record custom shortcuts for the search palette (default `Ctrl+P`) and the theme toggle (default `D`)

### Navigation & UX

- **Search palette** — `Ctrl+P` (configurable) opens a fuzzy-ish search over workspaces and apps with keyboard navigation and highlighted matches
- **Deep linking** — workspace/app/tab/config-set selection stays in sync with the URL; back/forward buttons work and links land on the same section
- **Light/dark theme** — dark, light, or system (follows `prefers-color-scheme`), toggled from the sidebar header button or the keyboard shortcut (default `D`, rebindable in Settings → Keyboard shortcuts)

### Platform integration

- **Native dialogs** — OS folder/file pickers (PowerShell / osascript / zenity) for app paths, templates, and `.env` imports
- **Smart startup** — picks the next free port when 3000/5173 are busy and opens the default browser automatically (standalone binary)
- **Single binary** — `bun run compile` bundles the frontend and migrations into `release/app-runner(.exe)`
