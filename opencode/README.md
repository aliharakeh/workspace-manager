# Workspace Manager for OpenCode Desktop

A native **OpenCode V2 Desktop GUI plugin** for the same daemon (`wmd`) used by
the Workspace Manager app, VS Code extension and MCP server. Open a session's
side-panel **+ → Workspace Manager**, or run **Open Workspace Manager** from
the command palette. The panel follows the session's local directory.

## Install from this repository

Build the daemon from the repository root if it is not already installed:

```powershell
go build -o build/bin/wmd.exe ./cmd/wmd
```

Build the plugin:

```powershell
cd opencode
npm ci
npm run build
```

Add this entry to `plugins` in your **global** OpenCode `opencode.jsonc`
(`~/.config/opencode/opencode.jsonc`, or `$XDG_CONFIG_HOME/opencode/opencode.jsonc`).
Keep your other settings and plugins. Use absolute paths to your checkout and
daemon; forward slashes work on Windows:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "plugins": [
    {
      "package": "C:/Workspace/GitHub/workspace-manager/opencode",
      "options": {
        "daemonPath": "C:/Workspace/GitHub/workspace-manager/build/bin/wmd.exe",
        "autoStartDaemon": true
      }
    }
  ]
}
```

OpenCode watches config and local bundles; use `/reload` if needed. The GUI is
listed under **Settings → Extensions → Plugins**. A project-only plugin entry
does not make its GUI available to the desktop app.

Targets the experimental GUI API in `@opencode/plugin` **2.2.5**. Use a compatible
OpenCode V2 Desktop release. This is not a V1, TUI or web-browser plugin.

## Features

- **Track a folder** in an existing or new Workspace Manager workspace.
- **Config sets:** switch, create empty or copy selected parts, rename, delete,
  and copy env/templates/setup/dev/build/prod into the active set.
- **Environment:** add, edit/rename, delete, import a project-local `.env` or YAML
  file. Values are masked in the list; editing reveals the selected value.
- **Templates:** snapshot a project file, create/edit content with `{{KEY}}`
  placeholders, preview with the daemon's renderer, save and delete. Editing a
  template does not write the project file; the daemon applies it during runs.
- **Runs:** Setup / Dev / Build / Prod, Stop and Reload, command status, exit
  codes, detected HTTP(S) links, and read-only xterm output with ANSI colors and
  late-open catch-up. Click a command to view its output.
- **Live changes:** refreshes after writes from any client and after reconnect.
  Open editor drafts are not overwritten by incoming refreshes. A save refuses
  when the active config set has changed since the editor opened.

Configure run commands in the Workspace Manager app, as with the VS Code
extension. Runs belong to the daemon and survive closing OpenCode.

### Differences from VS Code

Template editing uses an explicit **Save** dialog, not a virtual editor file
system or Explorer context menu. Snapshot/import paths are entered relative to
the current project. The panel does **not** inject environment variables into
OpenCode's own terminals or agent processes; daemon-managed runs do receive
them. Remote OpenCode directories are not supported: `wmd` and the project must
be local to this desktop machine. The panel does not expose agent commands or
tools; use the existing [MCP server](../cmd/wm-mcp/README.md) for agent access.

## Connection and security

| Option | Meaning |
|---|---|
| `daemonPath` | `wmd` executable; defaults to `WM_DAEMON`, then `PATH`. |
| `dataDir` | Data folder override; defaults to `WM_DATA_DIR`, then the OS default. |
| `autoStartDaemon` | Start `wmd` on demand (default `true`). |

The plugin reuses `vscode/daemon.js` and `vscode/service.js`, bundled into the
single distributable. Only Electron's **main process** reads `wmd.endpoint`,
authenticates over loopback TCP, and holds the `gui` connection. Tokens are not
sent to renderer windows. A schema-validated IPC bridge exposes the panel's
operations, not arbitrary daemon calls. No GUI-privileged connection or private
env/template contents are exposed as model tools. As in VS Code, the human GUI
can see private values.

## Development

```powershell
npm ci
npm run dev          # watch and rebuild; OpenCode reloads the local bundle
npm run typecheck
npm test             # bundle smoke tests + backend tests against an isolated real wmd
npm pack             # typecheck/build, then create a distributable .tgz
```

The packed plugin includes `dist/gui-experimental.cjs`, which embeds the shared
VS Code client/service; it does not need the sibling `vscode/` folder installed.
The repository's `build.ps1` packs it into `build/bin` too.

| Path | Role |
|---|---|
| `src/gui.tsx`, `src/view.tsx` | Panel, launcher, commands and editor dialogs. |
| `src/contract.ts` | Schema-validated window ↔ main IPC. |
| `src/main.ts`, `src/backend.js` | Native connection, event relay and allowlisted service operations. |
| `src/output.tsx` | Read-only terminal output, catch-up, resize and cleanup. |
| `test/` | Isolated daemon integration and built-bundle main-process smoke tests. |
