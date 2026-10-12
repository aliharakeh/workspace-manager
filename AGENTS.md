# Workspace Manager

Three Go modules plus a Wails app, tied together by `go.work`. The background
server `wmd` owns the database, the process runner, the blueprints and the AI;
the `wm` CLI and the desktop app are both clients of it and talk JSON over a
local socket. `api/` is the contract every side shares.

## Layout

| Path | Role |
|---|---|
| `api/` | Shared contract: `types` (JSON payloads), `methods` (the method registry), `rpc` (wire protocol and both connection ends), `client` (dial, handshake, auto-start, subscriptions), `lib` (pure helpers). |
| `server/cmd/wmd` | The daemon's entry point: opens the database, runs the server. |
| `server/daemon` | Lifecycle: data-folder lock, socket, idle exit, notifications, method table wiring. |
| `server/handlers` | One handler per API method; applies the client-kind rules (AI is GUI-only, `include_in_ai` values are hidden from the CLI). |
| `server/ops` | Every write operation, ported from the old Wails bindings, with validation. |
| `server/services` | Runner, templates, git/worktrees, worktree links, ready URLs, blueprints, notifier. |
| `server/ai` | AI connections and the two agents. They run in the GUI's process but are served to GUI clients only. |
| `server/db` | SQLite via sqlc: `schema.sql` (run at every startup), `queries.sql`, generated code, explicit upgrades. |
| `server/native` | OS helpers: process, ports, editor, folder, browser, notifications, PTY (`pty*.go`). |
| `cli/cmd/wm` | The CLI: one Cobra command group per domain. |
| `gui/` | The Wails app. `bind.go` proxies to the server; `frontend/` is the React UI. |
| `go.work` | Development workspace tying `api`, `server`, `cli` and `gui`. |

## How the pieces talk

- Protocol: one JSON message per line over a unix socket in the data folder
  (`wmd.sock`). `{id, method, params}` → `{id, result}` or
  `{id, error:{code, message, kind}}`, plus pushed `{event, data}`
  notifications. Requests run concurrently per connection, so a client can
  call `blueprints.resolve` while its own `blueprints.createApp` is still in
  flight.
- Every client's first call is `hello`, which states `cli` or `gui`. That
  decides two things: methods marked `CLI: false` in `api/methods.go` are
  refused to the CLI (the AI), and env vars/templates with
  `include_in_ai=false` are blanked for CLI clients and refuse writes from
  them. Any new secret-like value needs the same treatment in
  `server/handlers`.
- The daemon takes a lock in the data folder, so there is one instance per
  data folder. It exits after ten idle minutes with no client connected and nothing running. Clients
  start it on demand (`api/client.StartDaemon`), and `WM_DATA_DIR` moves both
  the daemon and its clients to another data folder, which is how tests and
  sandboxes stay isolated.
- Events are the frontend's contract: `runnerEvent` carries `{appId, event}`,
  `blueprintEvent` a `BlueprintLogEvent`, `appAIEvent` an `AppAIStreamEvent`.
  Keep those names and payload shapes in `server/daemon` and `gui/app.go`.
- The runner's terminal output is raw bytes, base64-encoded in `LogEvent`;
  `services/terminal.go` keeps a plain-text copy for line-based work such as
  ready-URL detection. The GUI feeds `runnerEvent` into xterm.js
  (`frontend/components/terminal.tsx`).

## Rules

- Adding a server operation: add the method constant in `api/methods.go`
  (set `CLI`), put the operation in `server/ops` (or a service), and register
  a handler in `server/handlers`. The daemon refuses to start if a method has
  no handler, or a handler has no method, so nothing can drift.
- Adding a CLI command: a Cobra command in `cli/cmd/wm` that calls the API
  method. Errors come back typed; map them with `api.KindOf` and the exit
  codes (0 ok, 1 failed, 2 usage, 3 not found, 4 conflict, 5 unavailable).
  `--json` output is written by `printJSON`.
- Changing the GUI's bound surface: edit `gui/bind.go`, then run
  `wails generate module` from `gui/` and check `frontend/host.ts`. Most
  methods proxy to the server; only the file dialogs, `OpenExternal`, and the
  AI methods run in the GUI process, and only the AI methods touch the
  database directly there.
- Schema or query changes: edit `server/db/schema.sql` / `server/db/queries.sql`,
  then run `sqlc generate` from the repo root. `schema.sql` runs at every
  startup with `IF NOT EXISTS` statements, so it only creates missing tables:
  a change to an existing table needs explicit upgrade code in
  `server/db/schema.go`. Because the daemon, the GUI and the CLI all open the
  database, migrations run under a file lock (`server/db/migratelock*.go`).
- A config set holds three command lists of the same shape: a **run** config,
  a **build** config and a **setup** config, told apart by `run_configs.kind`.
  An app has one runner session at a time (`runner.run`/`build`/`setup` fail
  while any is going) because all apply and restore templates; a build or
  setup skips ready-URL and idle detection.
- Apps are listed by `apps.sort_order` (per workspace), not by name: new apps
  go last and `apps.reorder` rewrites the order. The UI must not re-sort apps
  by name.
- Blueprints: the runner is in `server/services/blueprints.go`; the agent that
  drafts them is `server/ai/blueprintai.go` and delivers only through its
  `propose_blueprint` tool. When you change the runner or the placeholder set
  (`{{app_name}}`, `{{folder_name}}`, `{{app_dir}}`), update the agent's
  system prompt and `validateBlueprintProposal`, and the preview in
  `frontend/components/new-app-from-blueprint-dialog.tsx`.
- Git and worktrees go through `server/services/git.go` and
  `server/services/worktreelinks.go`: the main worktree and the app's own
  worktree can never be removed; a worktree can run the app's configs via
  `git.worktree.start`; shared paths (`worktree_links`) link a worktree's
  loose files back to the app's folder and are added to the repo's
  `info/exclude`.
- Notification setting keys live in both `server/services/notifier.go` and
  `gui/frontend/lib/notifications.ts`; keep them in sync. The daemon sends the
  OS notifications.
- Server-side and client-side type names match on the JSON tags only:
  `api/types` is the single source; `gui/frontend/lib/types.ts` mirrors it by
  hand.

## Commands

From the repo root (note: `go build ./...` does not work at a workspace root):

```bash
go build ./api/... ./server/... ./cli/... ./gui/...
go test ./api/... ./server/...
go build -o build/bin/wmd ./server/cmd/wmd
go build -o build/bin/wm  ./cli/cmd/wm
sqlc generate                   # after schema/query edits
```

From `gui/`: `wails dev` (Vite on port 5174), `wails build`,
`wails generate module`.

From `gui/frontend/`: `bun install`, `bun run dev`, `bun run build`,
`bun run lint`, `bun run typecheck`.

Development tip: in `wails dev` the app runs from a temporary build folder, so
`wmd` must be findable — next to the app binary (`gui/build/bin`) or on PATH.
`api/client` looks next to its own executable first, then on PATH.
