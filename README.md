# Workspace Manager

**Get your local projects running without the terminal shuffle.**

Workspace Manager brings your projects, startup setup, and live output into one desktop app. Group related apps into a workspace, then start, stop, or restart them from one place.

## The problem it solves

Getting into a project often means opening several terminal tabs, remembering environment settings, starting services in the right order, and hunting through output when something breaks. When you switch between projects or run a frontend, API, and worker together, that setup steals time before you can get to the work.

Workspace Manager saves that setup with each app and gives you one place to run and monitor a workspace.

## Features

### Keep related apps together

Group related apps into a workspace and see all of them on one screen, with each app's status and running count visible at a glance.

### Start each app with one click

Give an app its commands once, then run, stop, or reload it with a single click instead of typing commands into a terminal. Commands inside an app run in parallel or in sequence, depending on what that app needs.

### Keep project setup reusable

Save run commands, environment variables, and file templates in named config sets such as `dev` and `staging`. Switch setups without retyping variables or preparing files by hand.

### See what's running

Check each app's status and follow live, searchable output in one place. When a development server prints its local URL, open it with a click.

### Create new projects from a recipe

Save setup commands as blueprints and use them to create new apps. If a command asks for input, answer it in the built-in terminal.

### Find local port conflicts

See which process is using a listening port and stop it from Settings.

## Development

Needs [Go](https://go.dev/), [Wails](https://wails.io/) v2 and [Bun](https://bun.sh/). From the repo root:

```bash
wails dev                # Wails + Vite (port 5174)
wails build              # production native app → build/bin (builds for the OS you run it on)
wails generate module    # regenerate frontend/wailsjs after changing bound Go methods or types
sqlc generate            # regenerate db after editing schema.sql / queries.sql
go test ./db/... ./lib/... ./native/... ./services/...
```

Frontend scripts (`bun install` once in `frontend/`): `bun run dev`, `bun run build`, `bun run lint`, `bun run typecheck`.

## Project layout

| Path | Role |
|---|---|
| `main.go`, `app.go`, `bind.go`, `events.go` | Wails entry point, `App` lifecycle, bound API methods, event emission |
| `services/` | Runner, templates, blueprints, ready URLs, notifier, AI |
| `lib/`, `native/` | fs / env / import helpers; OS helpers (process, ports, editor, browser) |
| `db/` | SQLite schema, queries, sqlc-generated code |
| `types/` | Types shared with the UI (JSON tags match `frontend/lib/types.ts`) |
| `frontend/` | React UI (Vite); `host.ts` is the Wails adapter, `wailsjs/` is generated |
| `build/` | Wails packaging assets |
