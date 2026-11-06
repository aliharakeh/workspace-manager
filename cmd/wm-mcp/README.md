# wm-mcp: the workspace-manager MCP server

An MCP server over stdio that lets any coding agent drive workspace-manager:
list workspaces and apps, read and edit config, run apps, scaffold from
blueprints. Git, worktrees, settings and the AI are GUI-only and not exposed to agents.

It talks to the `wmd` daemon through `shared/client` and starts the daemon on
demand, so there is nothing to run by hand. It identifies as an `agent` client:
AI methods stay unavailable and values with `private` stay private,
exactly as for any external agent.

`wm_env`, `wm_template` and `wm_config` (`get_commands`/`save_commands`) take an
optional `set` (id or name) and work on any config set of the app; nothing has to
be activated first. Without `set` they use the active set.

A `wm_env` variable can be a choice: `set` it with `options` (`[{label, value}, ...]`)
and optionally `selected` (a label); the first option is selected by default, and
`value` always equals the selected option's. Passing `value` alone on a choice edits
the selected option, `selected` alone switches it, and `options: []` makes it a plain
value again. A private choice returns its labels but never its values, and its options
and selection cannot be changed through MCP.

`wm_env` can replace or delete a private variable, and `wm_template` can edit or
delete a private template, without reading the previous value.
Omit `private` when updating to preserve the existing flag, or pass `private: true`.
The stored value stays blank in lists, config details and write responses.
Private items cannot be renamed or made public through MCP; use
the GUI for those changes. New variables default to public unless `private: true`.

## Shared config

A workspace can own shared config sets of env vars. `wm_shared` edits them
(`set`, `delete`, `create_set`, ...) and links an app config set to one:
`link` takes `keys` (omit for all) from any shared set, whatever the names, and
`replace: true` lets a shared variable overwrite an own one of the same name;
`unlink` stops (`keep: true` keeps the variables as the app's own). A linked
variable is read-only through `wm_env`, which refuses to change or delete it, and
a `wm_shared` edit reaches every config set using it at once. Private shared values
follow the same rules as private app variables: never returned, replaceable, not
renamable or made public.

## Build

For source changes, `tools.go` holds the registry and schema builders;
`tools_<domain>.go` holds each tool's schema and handler. See the
[architecture map](../../docs/architecture.md) for the surrounding layers.

From the repo root:

```bash
go build -o build/bin/wm-mcp ./cmd/wm-mcp
```

`.\build.ps1` builds it alongside the daemon and the GUI into `build/bin`; `wm-mcp` finds `wmd` next to itself (or via `WM_DAEMON`, or PATH).

## Configure a client

Point any MCP client at the binary:

```json
{
  "mcpServers": {
    "workspace-manager": { "command": "C:/path/to/wm-mcp.exe" }
  }
}
```

Claude Code: `claude mcp add workspace-manager -- C:/path/to/wm-mcp.exe`

Environment is inherited, so the server uses your normal data folder. Set
`WM_DATA_DIR` to isolate it (same variable as the daemon and the GUI), or
`WM_SOCKET` to point at a specific daemon socket.

## Tools

| Tool | What it does |
|---|---|
| `wm_overview` | Workspaces, their apps and runner states. Start here. |
| `wm_app` | list, get (app + config + status), create, update, delete, reorder |
| `wm_config` | Config sets: list, detail, create, rename, activate, copy, delete; command lists: get/save, package scripts |
| `wm_env` | Env vars: list, set (upsert, with `options` and `selected` for a choice variable), delete, import |
| `wm_env_script` | The dynamic environment script of an app config set or a shared set: get, set, clear, run, stop. Output is never returned |
| `wm_shared` | A workspace's shared config: list, get, create_set, rename_set, delete_set, set/delete a variable, and link/unlink an app config set to a shared set |
| `wm_template` | File templates: list, add, edit, delete |
| `wm_run` | run, build, setup, stop, reload, status, logs, startAll, stopAll (a whole workspace) |
| `wm_blueprint` | Blueprints: list, show, create, update, delete, create_app (fails fast), resolve, cancel |
| `wm_ports` | List listening ports, stop a process |
| `wm_file` | Read a file inside an app's folder |
| `wm_call` | Raw escape hatch: any daemon method available to an agent client |

Apps, workspaces and config sets accept an id or a name; ambiguity is reported
instead of guessed.

Pass `configSet` directly to `wm_run` to choose the commands, environment and
templates for that run without changing the app's active config set:

```json
{"action": "dev", "app": "my-app", "configSet": "staging"}
```

This works for `dev`, `prod`, `build`, `setup` and the legacy `run` alias.
Omitting `configSet` uses the app's active set, as the GUI does. Each app still
allows only one session at a time. Reload keeps an explicitly selected set;
stop the app before starting it with a different set. `startAll` continues to
use each app's active set.

Raw daemon calls to `runner.dev`, `runner.prod`, `runner.build`, `runner.setup`
and `runner.run` accept the optional numeric `configSetID` parameter.
