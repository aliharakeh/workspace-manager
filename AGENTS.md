# Workspace Manager

Uses Go 1.26+ and Wails v3.0.0-beta.34. Install the matching CLI with
`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34`.

One Go module at the repo root with two programs: the Wails desktop app (the
GUI, `main.go`) and the background daemon `wmd` (`cmd/wmd`). The daemon owns the
database, the process runner and the blueprints. The GUI is its client; it also
runs git, the file dialogs and the path from terminal output to the
screen. The React frontend lives in `frontend/`.

## Layout

| Path | Role |
|---|---|
| `main.go` | Wails entry point only (Wails needs the main package at the repo root): embeds the frontend and runs `gui.App`. |
| `cmd/wmd` | The daemon's entry point: opens the database, runs the server. |
| `cmd/wm-mcp` | The MCP server (stdio) for external coding agents. An `agent` client of the daemon; starts it on demand. See its README. |
| `vscode` | The VS Code extension (JS, esbuild): config sets, env vars, templates and run buttons backed by the daemon, as a `gui` client over the TCP endpoint. See its README. |
| `opencode` | OpenCode V2 Desktop GUI plugin (Solid, Vite): reuses `vscode/daemon.js` and `vscode/service.js` in Electron main, with schema-validated IPC to its session side panel. See its README. |
| `gui` | The desktop app's Go side. `app.go` (the `App` struct, event relay), `link.go` (`Link`, the connection to `wmd`: dials, auto-starts it, resubscribes), `bind.go` (the Wails-bound surface; data, runner and blueprint methods proxy to the daemon, the rest runs here), `events.go`, `dialog.go` (Wails events and file pickers). |
| `gui/git` | Git and worktrees: `git.go`, `worktreelinks.go` (the git plumbing) and `operations.go` (the Git panel's operations, reading what they need from the daemon through a `Backend`). |
| `daemon` | Daemon lifecycle: data-folder lock, socket, idle exit, `hello`, method permissions. |
| `daemon/handlers` | One handler per API method; applies the client-kind rules. |
| `daemon/ops` | Every operation of the daemon, with validation. Each write is announced through a `Publisher`. |
| `daemon/services` | Runner, templates, ready URLs, blueprints, notifier. |
| `daemon/db` | SQLite via sqlc: `schema.sql` (run at every startup), `queries.sql`, generated code, explicit upgrades. |
| `shared/api` | Contract both sides use: `methods.go` (the method registry), `events.go` (event types), `types.go` (data folder, socket path, version, errors), `types` (JSON payloads), `lib` (pure helpers). |
| `shared/rpc`, `shared/client` | The wire protocol (server and connection ends) and the dial/handshake/auto-start client. |
| `shared/native` | OS helpers: process, ports, editor, folder, browser, notifications, PTY (`pty*.go`). |
| `frontend/` | The React UI (Vite, TypeScript, Bun). `bindings/` is generated. |
| `build/` | Wails build assets (icons, installer); `build/bin` is the output (`workspace-manager.exe`, `wmd.exe`, `wm-mcp.exe`). |

## Who does what

| `wmd` (daemon) | GUI process only |
|---|---|
| Database, migrations, seed | File and folder pickers, `FsValidatePath`, `OpenExternal` |
| Workspaces, apps, config sets, env vars, templates, run configs, package scripts, ready-URL patterns, ports | — |
| The runner: PTYs, sessions, terminal buffer, ready-URL and idle detection, template apply/restore, OS notifications | Git and worktrees (`gui/git`), reading the app from the daemon |
| Settings (served to the GUI only) | Turning daemon events into Wails events; xterm rendering, resize and catch-up calls |
| Blueprints: CRUD and the runner (the agent that drafts them is GUI-side) | |

## How the pieces talk

- **Protocol.** One JSON message per line over a unix socket in the data folder
  (`wmd.sock`): `{id, method, params}` → `{id, result}` or
  `{id, error:{code, message, kind}}`, plus pushed `{event, data}`
  notifications. Requests run concurrently per connection, so a client can call
  `blueprints.resolve` while its own `blueprints.createApp` is in flight.
- **TCP endpoint.** Node cannot open the unix socket on Windows, so the daemon
  also listens on `127.0.0.1:<port>` and publishes `{addr, token}` in
  `wmd.endpoint` (mode 0600) in the data folder. A TCP connection must put the
  token in its `hello` (`ClientInfo.Token`) before anything else runs; unix
  connections need none. The VS Code extension uses this. AF_UNIX paths are
  limited to ~108 characters, so a long `WM_DATA_DIR` stops the daemon from
  starting.
- **Client kinds.** Every client's first call is `hello`, stating `gui` or
  `agent`; the rpc layer refuses every other call until it has. Methods marked
  `Agent: false` in `shared/api/methods.go` are refused to agent clients
  (settings, the worktree support methods, `configSets.applyChanges`, terminal
  resizes), and env vars/templates marked `private` are blanked for
  agents. Private env vars and templates allow value/content-only
  replacements and deletion, but not renaming or making public. A client that omits its
  kind is an agent, the least-privileged kind, never the GUI. Any new secret-like value needs the same
  treatment in `daemon/handlers`. `wm-mcp` is the agent client; the GUI is the other.
- **Lifecycle.** The daemon takes a lock in the data folder, so there is one
  instance per data folder. `Link` starts it on demand (`client.StartDaemon`
  finds `wmd` next to the app, in `WM_DAEMON`, or on PATH). It exits after ten
  idle minutes with no client connected and nothing running. **Quitting the GUI
  does not stop the apps it started**: the daemon, and the runs, keep going
  while any run is active. `WM_DATA_DIR` moves both ends to another data folder,
  which is how tests and sandboxes stay isolated.
- **Events.** The frontend's contract: `runnerEvent` carries `{appId, event}`,
  `blueprintEvent` a `BlueprintLogEvent`, `dataChanged` an `api.DataChanged`. The daemon
  pushes the first two as `api.Event`s and `App.relay` (`gui/app.go`) re-emits them
  under those names; keep the names and payload shapes. After a reconnect the
  GUI asks the frontend to reload its lists.
- **Writes announce themselves.** Every write goes through `ops`, which reports
  it as a `dataChanged` event (`api.DataChanged{Entity, IDs}`) so the UI
  refreshes: it reloads the lists on `app`/`workspace` and the open app's panels
  when its ids include the selected app. `ops.changed` ids are app ids
  (workspace ids for a workspace), never row or config set ids; resolve them when
  a write only knows a child row. Git writes are announced by the GUI itself
  (`gitBackend.Changed`). Add an entity to `APP_DATA_ENTITIES` in `App.tsx` when
  a new kind of app data appears.
- **The runner's terminal output** is raw bytes, base64-encoded in `LogEvent`;
  `services/terminal.go` keeps a plain-text copy for line-based work such as
  ready-URL detection. The frontend feeds `runnerEvent` into xterm.js
  (`frontend/components/terminal.tsx`) and catches up on a late open with
  `RunnerOutput`.
- **No AI in the GUI.** The desktop app has no AI features; external agents use
  `wm-mcp`. `configSets.applyChanges` stays in the daemon (GUI-only) but nothing
  in the GUI calls it now.

## Rules

- Adding a daemon operation: add the method constant in
  `shared/api/methods.go` (set `Agent`), put the operation in `daemon/ops` (or
  a service), register a handler in `daemon/handlers`, then add the bound
  method in `gui/bind.go` that calls it through `call[...]`. The daemon refuses to
  start if a method has no handler, or a handler has no method, so nothing can
  drift. Param struct tags in the handler must match the map keys `bind.go` sends.
- Adding an MCP tool: implement it in the matching `cmd/wm-mcp/tools_<domain>.go` file and register it in `allTools` in `tools.go`. Each tool calls daemon methods. `wm_call` derives its allowlist from `api.AgentMethods()`, so a new method only needs `Agent: true` to be reachable; GUI-only methods (settings, git support, `configSets.applyChanges`, resizes) are not. Map typed errors with `api.KindOf`.
- The VS Code extension (`vscode/`): `service.js` holds everything per project
  folder and is shared by the UI and the `workspaceManager.agent.*` commands; both
  `daemon.js` and `service.js` must stay free of `vscode` imports so they can be
  tested against a real `wmd` (`npm test`; `npm run test:host` runs it inside VS
  Code). Edit commands and menus in `vscode/gen-manifest.mjs` and run
  `npm run manifest`. When an agent command or its arguments change, update the
  table in `vscode/README.md`. Env vars and new templates are written to the
  active config set, because the daemon's `envVars.*`/`templates.create` work on it.
- Adding a GUI-only operation (git, dialogs): put it in `gui/git` or
  `gui/bind.go`; it reaches data only through daemon calls, never the database.
- The OpenCode Desktop plugin (`opencode/`) bundles the VS Code client/service;
  keep them free of host-specific imports. Its GUI-privileged connection belongs
  in Electron main, never model tools. Add panel operations to the allowlist in
  `src/backend.js` and the IPC contract as needed. Run `npm run typecheck` and
  `npm test` from `opencode/`; rebuild with `npm run build` after shared client edits.
- Changing the bound surface: edit `gui/bind.go`, then run `wails3 generate bindings -ts -clean=true`
  and check `frontend/host.ts`. Keep helper methods that must not be bound on
  other types (see `gitBackend`), since Wails binds every exported method of
  `App`.
- Schema or query changes: edit `daemon/db/schema.sql` /
  `daemon/db/queries.sql`, then run `sqlc generate` from the repo root.
  `schema.sql` runs at every startup with `IF NOT EXISTS` statements, so it only
  creates missing tables: a change to an existing table needs explicit upgrade
  code in `daemon/db/schema.go`. Migrations run under a file lock
  (`daemon/db/migratelock*.go`). Only the daemon opens the database.
- Env var options: a variable can be a choice. `env_vars.options` (JSON
  `[{label, value}]`) and `selected` (a label) sit next to `value`, which always
  equals the selected option's value, so readers of `value` (runner env,
  templates, compare, AI) need no change. Write through `db.EnvStore` /
  `db.EnvStoreValue` so the three columns stay consistent (a value edit on a choice
  edits the selected option; the first option is selected by default). Option values
  are secret-like: `hideEnvVar` blanks them for agents and agents cannot change the
  options or choice of a private variable. Copy-from and bundle export/import carry
  them; `wm_env` takes `options` and `selected`.
- Shared config: a workspace owns config sets of env vars (`workspace_config_sets`,
  `workspace_env_vars`; `daemon/ops/shared.go`). An app config set links to one of
  them by hand (`config_sets.shared_set_id`, `configSets.linkShared`, any set
  whatever the names) and takes chosen variables as ordinary `env_vars` rows that
  carry `shared_var_id`. Those rows are **live and read-only**: every write to a
  workspace variable (`sharedVars.update`/`delete`, which go through `mergeEnvUpdate`
  so option rules stay the same) rewrites or removes them in the same transaction,
  so the runner, templates, compare, AI and export keep reading `env_vars.value`
  unchanged. Never write a shared row through `envVars.*`, the AI patch, an env
  import or `copyFrom` (they refuse or skip it); a new write path to `env_vars`
  must check `SharedVarID`. A rename that collides with a variable in a linked
  config set is refused, naming where. Private shared values follow the same rules
  as app variables (`hideSharedVar` in `daemon/handlers/shared.go`). A bundle
  records the shared sets and each config set's link and keys (version 2; an
  export without shared config stays version 1). Shared writes announce
  `sharedSet` (workspace id) plus `envVar` for the apps they touched; the UI
  listens to `sharedSet` in `App.tsx`. `wm_shared` edits the shared config over MCP.
- A config set holds four command lists of the same shape: **setup**, **dev**,
  **build** and **prod**, told apart by `run_configs.kind`. Existing `run`
  commands migrate to `dev`; `runner.run` remains a legacy alias for dev. An app
  has one runner session at a time (`runner.dev`/`prod`/`build`/`setup` fail
  while any is going) because all apply and restore templates; a build or setup
  skips ready-URL and idle detection.
- Apps are listed by `apps.sort_order` (per workspace), not by name: new apps
  go last and `apps.reorder` rewrites the order. The UI must not re-sort apps
  by name.
- Workspaces are listed by `workspaces.sort_order` (new ones go last, `workspaces.reorder`
  rewrites the order) and carry a `pinned` flag (`workspaces.setPinned`); clients show
  pinned first, then the saved order, and must not re-sort by name. Collapse state in
  the sidebar stays per-device (local storage).
- Blueprints: the runner is in `daemon/services/blueprints.go` (daemon). When
  you change the runner or the placeholder set (`{{app_name}}`,
  `{{folder_name}}`, `{{app_dir}}`), update the preview in
  `frontend/components/new-app-from-blueprint-dialog.tsx`.
- Git and worktrees go through `gui/git` (`git.go`, `worktreelinks.go`,
  `operations.go`): the main
  worktree and the app's own worktree can never be removed; a worktree can run
  the app's configs via `GitWorktreeStart`, which resolves the folders here and
  asks the daemon to start the run (`runner.startIn`); shared paths
  (`worktree_links`, stored by the daemon) link a worktree's loose files back to
  the app's folder and are added to the repo's `info/exclude`.
- Notification setting keys live in both `daemon/services/notifier.go` and
  `frontend/lib/notifications.ts`; keep them in sync. The daemon sends the OS
  notifications.
- Go and TypeScript type names match on the JSON tags only: `shared/api/types`
  is the single source; `frontend/lib/types.ts` mirrors it by hand.

## Commands

From the repo root:

```bash
go build ./...
go test ./...
go vet ./...
sqlc generate                  # after schema/query edits
wails3 task dev                # Vite on port 5174; builds wmd too
wails3 build                   # -> build/bin/workspace-manager.exe + wmd.exe
wails3 generate bindings -ts -clean=true # after changing bind.go (frontend/bindings)
.\build.ps1                    # wmd.exe + wails3 build + size listing
```

`wmd` must sit next to the app (or `wm-mcp`), or be named by `WM_DAEMON`, or be on PATH. For
`wails3 task dev`, the build task creates it next to the app. To use another daemon:

```bash
go build -o build/bin/wmd.exe ./cmd/wmd
WM_DAEMON=$PWD/build/bin/wmd.exe wails3 task dev
```

A stale `wmd` keeps running after you rebuild it; stop it (`wmd.exe` in the task
manager, or the `server.stop` call) so the next launch uses the new build.

From `frontend/`: `bun install`, `bun run dev`, `bun run build`,
`bun run lint`, `bun run typecheck`.
