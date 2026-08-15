# Workspace Manager

Same product, two shells. `frontend/` and `db/` are shared. Each shell has its own host adapter and a copy of backend logic that must stay in sync.

## Layout

| Path | Role |
|---|---|
| `frontend/` | Shared React UI. Imports `api` / `onRunnerEvent` from `@host` via `frontend/lib/api.ts`. |
| `db/` | Shared SQLite + Drizzle. Same user-data DB for both shells. |
| `native/` | Shared OS helpers (process, ports, editor, browser, run). bun_backend also uses `native/dialog.ts`. |
| `bun_backend/` | Bun HTTP server + Vite. UI talks over `fetch` / SSE. |
| `electrobun_backend/` | Electrobun desktop app. UI talks over RPC. |

`@host` is per-shell: `bun_backend/host.ts` (HTTP/SSE) vs `electrobun_backend/src/host.ts` (RPC). Both must export the same `api` shape the UI already calls.

## Shared vs duplicated

**Edit once (shared):** `frontend/`, `db/`, and most of `native/` (process, ports, editor, browser, run).

**Duplicated — change both shells in the same task.** Do not copy-paste blindly: keep the behavior identical, but write it in that shell’s style.

| Concern | bun_backend | electrobun_backend |
|---|---|---|
| Host / `api` | `host.ts` (`fetch`, `EventSource`) | `src/host.ts` (RPC) |
| API surface | `server/routes/*.ts` | `src/bun/rpc.ts` + `src/shared/rpc.ts` |
| Runner / templates / ready URLs | `server/services/` | `src/bun/services/` |
| fs / parse-env / import-formats | `server/lib/` | `src/bun/lib/` |
| File/folder pickers | `native/dialog.ts` | `src/bun/native/dialog.ts` (Electrobun `Utils.openFileDialog`) |
| UI types | `frontend/lib/types.ts` | also `src/shared/types.ts` (keep aligned with frontend types) |

## Dual-update rule

If you change behavior, an endpoint, runner logic, or the `api` surface in **one** shell, update the **other** shell in the same change. Adapt to that backend; do not leave one shell stale.

**bun_backend**

- HTTP JSON routes and status codes in `server/routes/`
- Live logs via SSE (`EventSource` in `host.ts`)
- Imports: `@db`, `@native`
- Dialogs: `@native/dialog`

**electrobun_backend**

- RPC request handlers in `src/bun/rpc.ts` and matching types in `src/shared/rpc.ts`
- Live logs via `rpc.send.runnerEvent` (not SSE)
- Imports from `src/bun/` are relative (`../../../db`, `../../../native`) — that tree is outside the Electrobun UI tsconfig aliases
- Dialogs: `src/bun/native/dialog.ts`, not `native/dialog.ts`
- `openExternal` goes through RPC, not `window.open`

Frontend-only UI that does not change `api` needs no backend dual update. A `db/` schema change is shared, but if the API shape changes, update both hosts and both backends.
