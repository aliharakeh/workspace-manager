# Workspace Manager

A Wails desktop app: one Go module at the repo root, a React frontend in
`frontend/`. There is no server and no separate process: the app opens the
SQLite database and owns the process runner, git, blueprints and the AI.

## Layout

| Path | Role |
|---|---|
| `main.go`, `app.go` | Wails entry point and the `App` struct: opens the database, builds the operations, relays the runner's events to the frontend. |
| `bind.go` | The Wails-bound surface the frontend calls. Most methods call `internal/handlers`; the file dialogs, `OpenExternal` and the AI methods are implemented here. |
| `events.go`, `dialog.go` | Wails event emitters and the file/folder pickers. |
| `internal/api` | The contract: `types` (JSON payloads), `methods.go` (the method registry), `lib` (pure helpers), errors and the data folder (`types.go`), event types (`events.go`). |
| `internal/handlers` | One handler per API method, and `Local`, an in-process caller used by `bind.go`. |
| `internal/ops` | Every write operation, with validation. Each write is announced through a `Publisher`. |
| `internal/services` | Runner, templates, git/worktrees, worktree links, ready URLs, blueprints, notifier. |
| `internal/db` | SQLite via sqlc: `schema.sql` (run at every startup), `queries.sql`, generated code, explicit upgrades. |
| `internal/native` | OS helpers: process, ports, editor, folder, browser, notifications, PTY (`pty*.go`). |
| `internal/ai` | AI connections and the two agents (app config, blueprint). |
| `frontend/` | The React UI (Vite, TypeScript, Bun). `wailsjs/` is generated. |
| `build/` | Wails build assets (icons, installer); `build/bin` is the output. |

## How it fits together

- **Calls.** The frontend calls bound methods on `App` (`frontend/host.ts`
  wraps them). `bind.go` forwards most of them to `handlers.Local.Call(method,
  params, &out)`, which runs the registered handler in the same process.
- **Events.** The frontend's contract: `runnerEvent` carries `{appId, event}`,
  `blueprintEvent` a `BlueprintLogEvent`, `appAIEvent` an `AppAIStreamEvent`,
  `dataChanged` an `api.DataChanged`. Keep those names and payload shapes in
  `app.go` and `events.go`.
- **Writes announce themselves.** Every write goes through `ops`, which
  reports it as a `dataChanged` event (`api.DataChanged{Entity, IDs}`) so the UI
  refreshes: it reloads the lists on `app`/`workspace` and the open app's panels
  when its ids include the selected app. `ops.changed` ids are app ids
  (workspace ids for a workspace), never row or config set ids; resolve them
  when a write only knows a child row. Add an entity to `APP_DATA_ENTITIES` in
  `App.tsx` when a new kind of app data appears.
- **The runner's terminal output** is raw bytes, base64-encoded in `LogEvent`;
  `services/terminal.go` keeps a plain-text copy for line-based work such as
  ready-URL detection. The frontend feeds `runnerEvent` into xterm.js
  (`frontend/components/terminal.tsx`).
- **The AI** runs in this process (`internal/ai`). It reads an app and its
  config set through `apps.get`/`configSets.detail`, stages its edits, and the
  user's review applies them with one `configSets.applyChanges` call. Env vars
  and templates with `include_in_ai=false` are never shown to it or changed by it.
- **Quitting the app stops the apps it started** (`App.shutdown` stops the
  runner, which restores their templates).
- `WM_DATA_DIR` moves the database and the AI config to another data folder,
  which is how tests and sandboxes stay isolated.

## Rules

- Adding an operation: add the method constant in `internal/api/methods.go`,
  put the operation in `internal/ops` (or a service), and register a handler in
  `internal/handlers`. `NewLocal` fails if a method has no handler, or a handler
  has no method, so nothing can drift. Then add the bound method in `bind.go`.
- Changing the bound surface: edit `bind.go`, then run `wails generate module`
  and check `frontend/host.ts`.
- Schema or query changes: edit `internal/db/schema.sql` /
  `internal/db/queries.sql`, then run `sqlc generate` from the repo root.
  `schema.sql` runs at every startup with `IF NOT EXISTS` statements, so it only
  creates missing tables: a change to an existing table needs explicit upgrade
  code in `internal/db/schema.go`. Migrations run under a file lock
  (`internal/db/migratelock*.go`).
- A config set holds four command lists of the same shape: **setup**, **dev**,
  **build** and **prod**, told apart by `run_configs.kind`. Existing `run`
  commands migrate to `dev`; `runner.run` remains a legacy alias for dev. An app
  has one runner session at a time (`runner.dev`/`prod`/`build`/`setup` fail
  while any is going) because all apply and restore templates; a build or setup
  skips ready-URL and idle detection.
- Apps are listed by `apps.sort_order` (per workspace), not by name: new apps
  go last and `apps.reorder` rewrites the order. The UI must not re-sort apps
  by name.
- Blueprints: the runner is in `internal/services/blueprints.go`; the agent
  that drafts them is `internal/ai/blueprintai.go` and delivers only through its
  `propose_blueprint` tool. When you change the runner or the placeholder set
  (`{{app_name}}`, `{{folder_name}}`, `{{app_dir}}`), update the agent's system
  prompt and `validateBlueprintProposal`, and the preview in
  `frontend/components/new-app-from-blueprint-dialog.tsx`.
- Git and worktrees go through `internal/services/git.go` and
  `internal/services/worktreelinks.go`: the main worktree and the app's own
  worktree can never be removed; a worktree can run the app's configs via
  `git.worktree.start`; shared paths (`worktree_links`) link a worktree's loose
  files back to the app's folder and are added to the repo's `info/exclude`.
- Notification setting keys live in both `internal/services/notifier.go` and
  `frontend/lib/notifications.ts`; keep them in sync.
- Go and TypeScript type names match on the JSON tags only: `internal/api/types`
  is the single source; `frontend/lib/types.ts` mirrors it by hand.

## Commands

From the repo root:

```bash
go build ./...
go test ./...
go vet ./...
sqlc generate                  # after schema/query edits
wails dev                      # Vite on port 5174
wails build                    # -> build/bin/workspace-manager.exe
wails generate module          # after changing bind.go
.\build.ps1                    # wails build + size listing
```

From `frontend/`: `bun install`, `bun run dev`, `bun run build`,
`bun run lint`, `bun run typecheck`.
