# wm-mcp: the workspace-manager MCP server

An MCP server over stdio that lets any coding agent drive workspace-manager:
list workspaces and apps, read and edit config, run apps, scaffold from
blueprints. Git, worktrees, settings and the AI are GUI-only and not exposed to agents.

It talks to the `wmd` daemon through `shared/client` and starts the daemon on
demand, so there is nothing to run by hand. It identifies as an `agent` client:
AI methods stay unavailable and values with `include_in_ai=false` stay hidden,
exactly as for any external agent.

## Build

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
| `wm_env` | Env vars: list, set (upsert), delete, import |
| `wm_template` | File templates: list, add, edit, delete |
| `wm_run` | run, build, setup, stop, reload, status, logs |
| `wm_blueprint` | Blueprints: list, show, create, update, delete, create_app (fails fast), resolve, cancel |
| `wm_ports` | List listening ports, stop a process |
| `wm_file` | Read a file inside an app's folder |
| `wm_call` | Raw escape hatch: any daemon method available to an agent client |

Apps, workspaces and config sets accept an id or a name; ambiguity is reported
instead of guessed.
