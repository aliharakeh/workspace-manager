# Workspace Manager MCP server

An MCP server over stdio that lets any coding agent drive workspace-manager:
list workspaces and apps, read and edit config, run apps, scaffold from
blueprints. Git, worktrees, settings and the AI are GUI-only and not exposed to agents.

Run `workspace-manager mcp`. It talks to the background daemon through `shared/client` and starts the daemon on
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
[architecture map](../docs/architecture.md) for the surrounding layers.

From the repo root:

```bash
wails3 build
```

`.\build.ps1` produces `build/bin/workspace-manager.exe`. It provides GUI, MCP and daemon modes; MCP starts the same executable in `daemon` mode on demand.

## Configure a client

Point any MCP client at the binary:

```json
{
  "mcpServers": {
    "workspace-manager": { "command": "C:/path/to/workspace-manager.exe", "args": ["mcp"] }
  }
}
```

Claude Code: `claude mcp add workspace-manager -- C:/path/to/workspace-manager.exe mcp`

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
| `wm_wait` | Wait for process completion or ready URLs; handles parallel commands and selected command IDs |
| `wm_diagnostics` | Detected errors/warnings for the current/latest run; incremental pages and per-problem context |
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


## Diagnostics

Use `wm_diagnostics` before requesting raw logs:

```json
{"app":"my-app"}
```

The default page contains at most 10 problem summaries (maximum `limit`: 20),
with run/config identity, error and warning group counts, and total occurrences.
`commandId` and `severity` filter items; totals describe the whole run.
For retained context/stack traces, pass an item's `id` and the returned `sessionId`.
For the next page or new/updated problems, pass `cursor: nextCursor` and that same
`sessionId`. Repeated problems update their count and revision, so they reappear
in incremental results. An old session id is refused after a new run starts.

Collection runs in the daemon as output arrives, independently of the terminal's
1 MiB buffer. Compiler locations, JSON log levels, common error/warning patterns,
Python tracebacks and Go/JavaScript stacks are recognized. Runner failures are
recorded directly; an intentional Stop is not a process error. Detection is
heuristic: an empty list does not establish health, and a successful rebuild does
not automatically resolve earlier problems in the same run.

Only the latest run per app is retained in SQLite, including after process exit
and daemon restart; the next run replaces it and deleting the app removes it.
Retention is capped at 512 distinct problem groups, 1 KiB per message and 8 KiB of
context per group. `dropped` counts unretained problem occurrences after the cap;
`contextTruncated` and `inputTruncated` identify shortened excerpts and oversized
input. Snapshots are saved at most once per second and flushed when a run ends;
a crash can lose the last second. A `persistenceError` means the live result is
available but its latest snapshot could not be saved. Excerpts contain process
output, with the same access as `wm_run` logs.

GUI launch controls and worktrees always use the active config set without a
second selector. GUI Reload uses the currently active set and keeps the serving
kind and worktree. MCP Reload preserves an explicit per-run config override.


## Waiting for a run

After `wm_run` starts a run, pass its `sessionId` to `wm_wait`:

```json
{"app":"my-app","sessionId":"<from wm_run>","until":"ready","timeoutSeconds":30}
```

The default `until: "ready"` waits until every selected command has a live URL or
has exited successfully, with at least one live URL. This handles parallel
frontend/backend commands without returning just because the first server is
ready. Successful setup commands do not need URLs. For a non-web worker that
never prints a URL, select just the web commands with `commandIds`, or explicitly
use `readiness: "any"` to return when any selected command has a live URL.

Use `until: "exit"` for builds/setup or to wait for all selected commands to finish:

```json
{"app":"my-app","sessionId":"<from wm_run>","until":"exit","commandIds":[12,13],"timeoutSeconds":60}
```

`reason` is `ready`, `completed`, `failed`, `stopped`, or `timeout`. A selected
process failure takes priority over readiness, even if another selected process
already has a URL. Specific command selections ignore unrelated command failures;
the full app status is still included. Each returned URL has a `commandId`,
`label`, and `url`, so agents can choose the correct browser target. Exited/killed
processes do not contribute live URLs.

The tool uses the existing daemon runner events and ready-URL patterns, with a
status refresh after each notification and a one-second fallback. Readiness is
log-detected, not an HTTP health probe; idle output alone never counts as a URL.
An app selection scopes the wait to that app, so simultaneous runs in other apps
cannot supply its URLs. An omitted `sessionId` binds to the current run; a replaced
run is refused rather than silently followed. Unknown command IDs are refused.

The wait defaults to 30 seconds and is capped at 60 seconds per call. A timeout
returns the latest status and leaves every process running. Call again with the
same session id for a longer wait. No raw terminal logs are returned, and the tool
does not open a browser. It only adds an MCP tool using existing daemon methods;
no GUI or daemon API binding changes are needed.
