# Workspace Manager

Four Go modules plus a Wails app, tied together by `go.work`. There is no
background server. The desktop app (`gui`) runs everything: it opens the SQLite
database, owns the process runner, git, blueprints and the AI. The MCP server
(`wm-mcp`) is a second, small program that opens the same database file to give
an AI agent the user's data and a guide for managing it. It edits
configuration but **runs nothing**. The two never talk to each other directly;
they coordinate through the database (see below).

## Layout

| Path | Role |
|---|---|
| `api/` | The contract both programs share: `types` (JSON payloads), `methods` (the method registry with the `Agent` flag), `lib` (pure helpers), errors and the data folder (`types.go`), event types (`events.go`). |
| `core/db` | SQLite via sqlc: `schema.sql` (run at every startup), `queries.sql`, generated code, explicit upgrades, `coord.go` (the change log and session claims). |
| `core/ops` | Every write operation, with validation. Each write is announced through a `Publisher`. |
| `core/handlers` | One handler per API method, and `Local`, an in-process caller: the GUI builds a `gui` Local, the MCP server an `agent` one. It applies the client-kind rules. |
| `core/services` | Runner, templates, git/worktrees, worktree links, ready URLs, blueprints, session claims, notifier. |
| `core/native` | OS helpers: process, ports, editor, folder, browser, notifications, PTY (`pty*.go`). |
| `gui/` | The Wails app. `bind.go` calls `core/handlers` as a `gui` Local; `ai/` holds the AI connections and the two agents; `dialog.go` has the file pickers; `app.go` wires the runner's events and the change-log watcher; `frontend/` is the React UI. |
| `mcp/cmd/wm-mcp` | The MCP server: curated tools (`tools.go`, `run.go`), `exec.go` (runs one prepared command), the embedded `guide.md`. |
| `go.work` | Development workspace tying `api`, `core`, `gui` and `mcp`. |

## How the pieces work together

- **Client kinds.** `handlers.NewLocal(ops, kind)` builds the method table.
  Methods marked `Agent: false` in `api/methods.go` are refused to an agent
  client, and env vars/templates with `include_in_ai=false` are blanked for it
  and refuse its writes. Anything that is not `gui` is an agent, the
  least-privileged kind. The `Agent` flag is the single list of what the MCP
  server may call: git, everything that runs a process, opening folders and
  URLs, killing ports and applying reviewed AI changes are all `Agent: false`.
  Any new secret-like value needs the same treatment in `core/handlers`.
- **Shared database.** Both programs open `the data folder's SQLite file`
  in WAL mode with a busy timeout. `WM_DATA_DIR` moves both to another data
  folder, which is how tests and sandboxes stay isolated. Migrations run under
  a file lock (`core/db/migratelock*.go`).
- **Change log.** The GUI shows what is in the database, but cannot see
  writes the MCP server makes. Every write through `ops` calls
  `Publisher.Changed`; the MCP server's publisher appends it to the `changes`
  table, and the GUI polls that table (`gui/app.go: watchChanges`) and turns
  new rows into `dataChanged` events. The GUI's own writes emit the event
  directly. Ids are app ids (workspace ids for a workspace), never row or
  config set ids; resolve them when a write only knows a child row. Add an
  entity to `APP_DATA_ENTITIES` in `App.tsx` when a new kind of app data
  appears. The MCP server also logs a `session` change when it takes or
  releases a run session.
- **Run sessions.** An app has one session at a time, shared across both
  programs: the `app_sessions` table. The GUI's runner claims a row when it
  starts (`runner.StartIn`) and drops it when the session stops; the MCP
  server's `wm_run prepare` claims one the same way. A row whose process is
  gone is stale: the next claim clears it and restores its templates. While an
  agent holds an app, the GUI reports it as running with `owner: "agent"` and
  hides its run controls.
- **The MCP server never runs anything.** `wm_run plan` describes a run,
  `prepare` takes the session, renders the templates and returns one
  `wm-mcp exec --app --session --command` line per command, `finish` restores
  the files and releases the session. `exec` loads the config set's full
  environment (including values hidden from the agent) into the child it
  runs, so secrets never pass through the agent's conversation. On exit the
  server restores and releases any session it prepared and was never asked to
  finish.
- **The guide.** `mcp/cmd/wm-mcp/guide.md` is what the agent reads: the data
  model, how to run an app, hidden values, etiquette. It is served as the
  server's MCP `instructions`, as the `workspace://guide` resource and by
  `wm_guide`. Keep it in step with the tools.
- **Events.** The frontend's contract: `runnerEvent` carries `{appId, event}`,
  `blueprintEvent` a `BlueprintLogEvent`, `appAIEvent` an `AppAIStreamEvent`,
  `dataChanged` an `api.DataChanged`. Keep those names and payload shapes in
  `gui/app.go` and `gui/events.go`.
- The runner's terminal output is raw bytes, base64-encoded in `LogEvent`;
  `services/terminal.go` keeps a plain-text copy for line-based work such as
  ready-URL detection. The GUI feeds `runnerEvent` into xterm.js
  (`frontend/components/terminal.tsx`).
- Closing the GUI stops the apps it started (`App.shutdown` stops the runner,
  which restores their templates).

## Rules

- Adding an operation: add the method constant in `api/methods.go` (set
  `Agent`), put the operation in `core/ops` (or a service), and register a
  handler in `core/handlers`. `NewLocal` fails if a method has no handler, or a
  handler has no method, so nothing can drift.
- Adding an MCP tool: a tool in `mcp/cmd/wm-mcp/tools.go` that calls an API
  method through `daoCall`. A method it needs must be `Agent: true`. Errors
  come back typed; map them with `api.KindOf` where the message matters.
  Update `guide.md`.
- Changing the GUI's bound surface: edit `gui/bind.go`, then run
  `wails generate module` from `gui/` and check `frontend/host.ts`. Most
  methods call the handlers; only the file dialogs, `OpenExternal` and the AI
  methods are implemented in the GUI. The AI reads an app through
  `apps.get`/`configSets.detail` and writes reviewed changes with one
  `configSets.applyChanges` call.
- Schema or query changes: edit `core/db/schema.sql` / `core/db/queries.sql`,
  then run `sqlc generate` from `core/`. `schema.sql` runs at every startup
  with `IF NOT EXISTS` statements, so it only creates missing tables: a change
  to an existing table needs explicit upgrade code in `core/db/schema.go`.
- A config set holds four command lists of the same shape: **setup**, **dev**,
  **build** and **prod**, told apart by `run_configs.kind`. Existing `run`
  commands migrate to `dev`; `runner.run` remains a legacy alias for dev. A
  build or setup skips ready-URL and idle detection.
- Apps are listed by `apps.sort_order` (per workspace), not by name: new apps
  go last and `apps.reorder` rewrites the order. The UI must not re-sort apps
  by name.
- Blueprints: the runner is in `core/services/blueprints.go`; the agent that
  drafts them is `gui/ai/blueprintai.go` and delivers only through its
  `propose_blueprint` tool. `Plan` (`blueprintplan.go`) resolves a run into
  data for the MCP server. When you change the runner or the placeholder set
  (`{{app_name}}`, `{{folder_name}}`, `{{app_dir}}`), update the agent's
  system prompt and `validateBlueprintProposal`, the plan, and the preview in
  `frontend/components/new-app-from-blueprint-dialog.tsx`.
- Git and worktrees go through `core/services/git.go` and
  `core/services/worktreelinks.go`: the main worktree and the app's own
  worktree can never be removed; a worktree can run the app's configs via
  `git.worktree.start`; shared paths (`worktree_links`) link a worktree's
  loose files back to the app's folder and are added to the repo's
  `info/exclude`. Git is GUI-only: every `git.*` method is `Agent: false`, the
  MCP server has no git tool, and the app AI agent has no git tools.
- Notification setting keys live in both `core/services/notifier.go` and
  `gui/frontend/lib/notifications.ts`; keep them in sync. The GUI sends the OS
  notifications.
- Server-side and client-side type names match on the JSON tags only:
  `api/types` is the single source; `gui/frontend/lib/types.ts` mirrors it by
  hand.

## Commands

From the repo root (note: `go build ./...` does not work at a workspace root):

```bash
go build ./api/... ./core/... ./gui/... ./mcp/...
go test ./api/... ./core/... ./gui/... ./mcp/...
.\build.ps1                     # MCP + GUI -> build/bin
go build -o build/bin/wm-mcp ./mcp/cmd/wm-mcp
sqlc generate                   # from core/, after schema/query edits
```

From `gui/`: `wails dev` (Vite on port 5174),
`wails build -o ../../../build/bin/workspace-manager.exe` (wails joins `-o`
with `gui/build/bin`), `wails generate module`.

From `gui/frontend/`: `bun install`, `bun run dev`, `bun run build`,
`bun run lint`, `bun run typecheck`.
