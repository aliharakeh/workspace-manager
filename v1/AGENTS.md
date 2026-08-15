# Agent guidance — workspace-manager-v1

## Native OS calls and commands

All systemic native OS integration lives under [`native/`](./native/).

Put new OS-level work there — do **not** scatter `Bun.spawn`, shell wrappers, or platform-specific commands across `server/`, `scripts/`, or `src/`.

### What belongs in `native/`

- Spawning OS tools (`powershell`, `osascript`, `zenity`, `netstat`, `lsof`, `open`, `xdg-open`, `cmd`, `sh`, …)
- Cross-platform helpers that pick an OS command by `process.platform`
- Process lifecycle helpers used to run/kill child processes
- Short-lived command runners that capture stdout/stderr

### What does not belong there

- App/domain orchestration (sessions, logging, HTTP routes) — keep in `server/`
- Dev/prod launchers that only start Bun/Vite (`scripts/dev.ts`, etc.) — those may call `native/` helpers, but stay in `scripts/`
- Ordinary filesystem path math and project-safe reads — use `server/lib/fs.ts`
- Frontend code — call server APIs; never spawn OS commands from `src/`

### Layout

| Module | Role |
|--------|------|
| `native/run.ts` | Capture stdout/stderr from a short-lived command |
| `native/dialog.ts` | Native file/folder pickers |
| `native/browser.ts` | Open default browser; detect standalone binary |
| `native/ports.ts` | List listening ports / find a free TCP port |
| `native/process.ts` | Shell spawn, env merge, process kill |
| `native/index.ts` | Re-exports (prefer importing the specific module) |

### Adding a new native capability

1. Add a focused helper file under `native/` (or extend an existing module if it clearly fits).
2. Keep platform branches inside that helper; export a small, stable API.
3. Import the helper from server/scripts — do not inline `Bun.spawn` for OS tools elsewhere.
4. Prefer `native/run.ts` for fire-and-forget command capture.
