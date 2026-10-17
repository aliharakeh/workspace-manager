# Workspace Manager for VS Code

Manage the **config sets, environment variables, file templates and run commands**
of your folders from the VS Code sidebar. Unlike a standalone extension, the data is
not kept in VS Code: it lives in the Workspace Manager daemon (`wmd`), so the
desktop app, the MCP server and this extension all see and change the same thing,
live.

A VS Code folder maps to the daemon app whose project path is that folder. Folders the
daemon does not track show an **Add** action (pick a workspace, or create one).

## What it does

- **Config sets**: switch (sidebar, or the status bar item), create (empty or copied
  from another set, choosing what to copy), rename, delete, copy parts into the active set.
- **Environment variables** of the active set: add, edit, delete, import a `.env`/YAML
  file. They are injected into integrated terminals of the folder, and used by runs.
- **Snapshot templates**: save any file as a template (Explorer or editor tab
  context menu, or the active editor), edit it in a normal editor tab (`wm-template:`;
  `Ctrl+S` saves to the daemon), use `{{KEY}}` placeholders, and preview it rendered
  by the daemon's renderer.
- **Run**: Setup / Dev / Build / Prod buttons start the active set's commands through
  the daemon, exactly like the desktop app (templates are applied for the run and
  restored afterwards). Status, per-command output in a VS Code terminal (raw
  terminal bytes, with catch-up for late opens), detected URLs, stop and reload. Runs
  belong to the daemon, so they keep going if you close VS Code.
- **Live**: the view refreshes when anything changes, whoever changed it.

Run commands are configured in the desktop app; the extension runs and shows them.

## The daemon

The extension connects to `wmd` and starts it when it is not running. It finds the
binary from the `workspaceManager.daemonPath` setting, then the `WM_DAEMON`
environment variable, then `PATH`. Build it with `go build ./cmd/wmd` (or `.\build.ps1`
in the repo root). `WM_DATA_DIR` / `workspaceManager.dataDir` select the data folder.

Node cannot open the daemon's unix socket on Windows, so the extension uses the
daemon's loopback TCP endpoint (`wmd.endpoint` in the data folder, readable only by
you, with a token every connection must present).

| Setting | Meaning |
|---|---|
| `workspaceManager.daemonPath` | Path to `wmd`. |
| `workspaceManager.dataDir` | Data folder override. |
| `workspaceManager.autoStartDaemon` | Start the daemon when it is not running (default on). |

## Agent commands

For AI agents and scripts, every feature is also a non-interactive command,
`workspaceManager.agent.<verb>` (`vscode.commands.executeCommand(id, args)`). They never
open UI, and always resolve to `{ ok: true, ...result }` or `{ ok: false, error }`.

`args.project` selects the workspace folder (a path or its name); it may be omitted
when exactly one folder is open. `args.set` (where listed) selects a config set by
name and defaults to the active one. Writes of env vars and **new** templates go to the
active set: switch to a set first to add to it.

| Command | Arguments | Result |
|---|---|---|
| `agent.listSets` | | `{ activeSet, sets: [{ name, active }] }` |
| `agent.switchSet` | `name` | `{ activeSet }` |
| `agent.createSet` | `name`, `copyFrom?`, `parts?` (`{ env, templates, setup, dev, build, prod }`, each `true`), `activate?` (default true) | `{ name, active }` |
| `agent.renameSet` | `oldName`, `newName` | `{ name }` |
| `agent.copyFrom` | `sourceSet`, `targetSet?` (default active), `parts?` (default env + templates) | `{ target, source }` |
| `agent.deleteSet` | `name` | `{ deleted, activeSet }` |
| `agent.listEnv` | `set?` | `{ set, env }` |
| `agent.setEnv` | `entries` (`{ KEY: "value" }`) or `key` + `value` | `{ set, env }` |
| `agent.removeEnv` | `keys` (array) or `key` | `{ removed, set, env }` |
| `agent.importEnv` | `file` (inside the project) | `{ set, imported, filePath }` |
| `agent.listTemplates` | `set?` | `{ set, templates: [path] }` |
| `agent.getTemplate` | `file`, `set?` | `{ set, filePath, content }` |
| `agent.setTemplate` | `file`, `content`, `set?` | `{ set, filePath }` |
| `agent.snapshotFile` | `file`, `set?` (saves the file's current disk content) | `{ set, filePath }` |
| `agent.removeTemplate` | `file`, `set?` | `{ set, removed }` |
| `agent.run` | `kind`: `setup`, `dev`, `build` or `prod` | `{ kind, running, sessionId }` |
| `agent.stop` | | `{ running }` |
| `agent.status` | | `{ running, kind, processes: [{ commandId, label, command, status, exitCode, urls }] }` |

Failures include: the folder is not tracked by the daemon, a config set or template
does not exist, a path outside the project, an app that is already running, and a run
kind with no commands.

## Development

```bash
cd vscode
npm install
npm run manifest     # regenerate package.json "contributes" from gen-manifest.mjs
npm run bundle       # esbuild -> out/extension.cjs (run this before F5)
npm test             # service + daemon client against a real wmd
npm run test:host    # the extension inside a real VS Code (the installed one, or VSCODE_PATH)
npm run build        # bundle + .vsix into ../build/bin
```

Edit commands and menus in `gen-manifest.mjs`, not in `package.json`.
Open the repo in VS Code, run `npm run bundle` in `vscode/`, then **F5** with
`--extensionDevelopmentPath=vscode` (an Extension Development Host).

| File | Role |
|---|---|
| `daemon.js` | The daemon client: endpoint file, token handshake, calls, events, auto-start, reconnect. No `vscode` import. |
| `service.js` | Everything per project folder, shared by the UI and the agent commands. No `vscode` import. |
| `extension.js` | Wiring: commands, status bar, terminal env, events. |
| `tree.js` | The sidebar. |
| `runner.js` | Run status and the output terminals. |
| `template-fs.js` | The `wm-template:` editor file system and the rendered preview. |
| `agent.js` | The agent commands. |
