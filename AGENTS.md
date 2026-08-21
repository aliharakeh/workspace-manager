# Workspace Manager

Same product, three shells. `frontend/` and `db/` are shared. Each shell has its own host adapter and a copy of backend logic that must stay in sync.

## Layout

| Path | Role |
|---|---|
| `frontend/` | Shared React UI. Imports `api` / `onRunnerEvent` from `@host` via `frontend/lib/api.ts`. |
| `db/` | Shared SQLite schema (Drizzle). Same user-data DB for all shells. |
| `native/` | Shared OS helpers for the JS shells (process, ports, editor, browser, run). bun_backend also uses `native/dialog.ts`. |
| `bun_backend/` | Bun HTTP server + Vite. UI talks over `fetch` / SSE. |
| `electrobun_backend/` | Electrobun desktop app. UI talks over RPC. |
| `wails_backend/` | Wails v2 desktop app (Go). UI talks over Wails bindings + events. SQLite via sqlc. Native helpers in `wails_backend/internal/native/`. |

`@host` is per-shell: `bun_backend/host.ts` (HTTP/SSE) vs `electrobun_backend/src/host.ts` (RPC) vs `wails_backend/frontend/host.ts` (Wails `window.go` + `EventsOn`). All three must export the same `api` shape the UI already calls.

## Shared vs duplicated

**Edit once (shared):** `frontend/`. Schema source of truth is `db/schema.ts` + `drizzle/` (JS shells). Wails maps that schema with sqlc — see below.

**Duplicated — change every affected shell in the same task.** Do not copy-paste blindly: keep the behavior identical, but write it in that shell’s style.

| Concern | bun_backend | electrobun_backend | wails_backend |
|---|---|---|---|
| Host / `api` | `host.ts` (`fetch`, `EventSource`) | `src/host.ts` (RPC) | `frontend/host.ts` (Wails bindings + `runnerEvent`) |
| API surface | `server/routes/*.ts` | `src/bun/rpc.ts` + `src/shared/rpc.ts` | `bind.go` (exported `App` methods) |
| Runner / templates / ready URLs | `server/services/` | `src/bun/services/` | `internal/services/` |
| fs / parse-env / import-formats | `server/lib/` | `src/bun/lib/` | `internal/lib/` |
| Native OS helpers | `native/` | `native/` + `src/bun/native/dialog.ts` | `internal/native/` (Go ports of `native/`) |
| File/folder pickers | `native/dialog.ts` | Electrobun `Utils.openFileDialog` | Wails `runtime.Open*Dialog` |
| SQLite | Drizzle (`db/`) | Drizzle (`db/`) | sqlc (`internal/db/`, mapped from Drizzle) |
| UI types | `frontend/lib/types.ts` | also `src/shared/types.ts` | `internal/types/` (JSON tags match frontend) |

## Dual-update rule

If you change behavior, an endpoint, runner logic, or the `api` surface in **one** shell, update the **other shells** in the same change. Adapt to that backend; do not leave one shell stale.

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

**wails_backend**

- Bound methods on `App` in `bind.go` (same operations as the RPC/HTTP API)
- Live logs via `runtime.EventsEmit` → `EventsOn("runnerEvent")` in `frontend/host.ts`
- After Go method/type changes: `wails generate module` (from `wails_backend/`)
- After Drizzle schema/query changes: copy new `drizzle/*.sql` into `internal/db/migrations/`, update `internal/db/schema.sql` + `internal/db/queries.sql`, then `sqlc generate`
- Dialogs and `openExternal` use the Wails runtime

Frontend-only UI that does not change `api` needs no backend dual update. A `db/` schema change is shared for JS shells, but Wails sqlc must be updated in the same change if the tables or queries moved.
