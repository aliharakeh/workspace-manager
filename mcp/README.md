# wm-mcp: the workspace-manager MCP server

An MCP server over stdio that gives an AI agent the user's workspace-manager
data and a guide for managing it. It reads and edits configuration (workspaces,
apps, config sets, env vars, templates, command lists, blueprints, settings).
It **runs nothing itself**: for a run it hands the agent the commands, and the
agent runs them. Git and the app's AI are not exposed.

It opens the same SQLite database as the desktop app, so the app does not need
to be running. When it is, the app refreshes by itself after an edit made here
(through the `changes` table), and shows an app the agent is running as "by
agent". Values with `include_in_ai=false` stay hidden: blanked in output, and
writes to them are refused.

## Build

From the repo root:

```bash
go build -o build/bin/wm-mcp ./mcp/cmd/wm-mcp
```

`.\build.ps1` builds it alongside the GUI.

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
`WM_DATA_DIR` to isolate it (same variable as the app).

## The guide

`guide.md` (embedded) tells the agent how the data model fits together, how to
run an app, why some values are hidden and how to behave. The server sends it as
its MCP `instructions`, as the `workspace://guide` resource, and through the
`wm_guide` tool.

## Tools

| Tool | What it does |
|---|---|
| `wm_guide` | The guide. Read it first. |
| `wm_overview` | Workspaces, their apps, and which apps have a live run session. |
| `wm_workspace` | list, create, update, delete |
| `wm_app` | list, get (app + config + session), create, update, delete, reorder |
| `wm_config` | Config sets: list, detail, create, rename, activate, copy, delete; command lists: get/save, package scripts |
| `wm_env` | Env vars: list, set (upsert), delete, import |
| `wm_template` | File templates: list, add, edit, delete |
| `wm_run` | How to run an app: `plan` (what a run does), `prepare` (reserve the app, write its templates, return the commands to run), `finish` (restore the files, release the app), `status` |
| `wm_blueprint` | Blueprints: list, show, create, update, delete, `plan_app` (the resolved commands for a new app, for the agent to run) |
| `wm_ports` | List listening ports |
| `wm_system` | Settings, path validation, reading app files |

Apps, workspaces and config sets accept an id or a name; ambiguity is reported
instead of guessed.

## Running an app

`wm_run prepare` returns one line per command, such as
`wm-mcp exec --app 3 --session 3-1700000000000-agent --command 12`. The agent
runs those lines in its own shell. `wm-mcp exec` loads the config set's full
environment, including values hidden from the agent, into the command it runs,
so secrets reach the process and never the conversation. `wm_run finish`
restores the files the templates replaced. An app has one session at a time,
shared with the desktop app; if the app is running there, `prepare` fails.
