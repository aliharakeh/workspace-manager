# Workspace Manager

Same product, two shells. `frontend/` and `db/` are shared. Each shell has its own host adapter and a copy of backend logic that must stay in sync.

## Layout

| Path | Role |
|---|---|
| `frontend/` | Shared React UI. Imports `api` / `onRunnerEvent` from `@host` via `frontend/lib/api.ts`. |
| `db/` | Shared SQLite schema (Drizzle). Same user-data DB for all shells. |
| `native/` | Shared OS helpers for the JS shell (process, ports, editor, browser, run, dialog). |
| `bun_backend/` | Bun HTTP server + Vite. UI talks over `fetch` / SSE. |
| `wails_backend/` | Wails v2 desktop app (Go). UI talks over Wails bindings + events. SQLite via sqlc. Native helpers in `wails_backend/internal/native/`. |

`@host` is per-shell: `bun_backend/host.ts` (HTTP/SSE) vs `wails_backend/frontend/host.ts` (Wails `window.go` + `EventsOn`). Both must export the same `api` shape the UI already calls.

## Shared vs duplicated

**Edit once (shared):** `frontend/`. Schema source of truth is `db/schema.ts` + `drizzle/` (JS shell). Wails maps that schema with sqlc — see below.

**Duplicated — change every affected shell in the same task.** Do not copy-paste blindly: keep the behavior identical, but write it in that shell’s style.

| Concern | bun_backend | wails_backend |
|---|---|---|
| Host / `api` | `host.ts` (`fetch`, `EventSource`) | `frontend/host.ts` (Wails bindings + `runnerEvent`) |
| API surface | `server/routes/*.ts` | `bind.go` (exported `App` methods) |
| Runner / templates / ready URLs | `server/services/` | `internal/services/` |
| fs / parse-env / import-formats | `server/lib/` | `internal/lib/` |
| Native OS helpers | `native/` | `internal/native/` (Go ports of `native/`) |
| File/folder pickers | `native/dialog.ts` | Wails `runtime.Open*Dialog` |
| SQLite | Drizzle (`db/`) | sqlc (`internal/db/`, mapped from Drizzle) |
| UI types | `frontend/lib/types.ts` | `internal/types/` (JSON tags match frontend) |

## Dual-update rule

If you change behavior, an endpoint, runner logic, or the `api` surface in **one** shell, update the **other shell** in the same change. Adapt to that backend; do not leave one shell stale.

**bun_backend**

- HTTP JSON routes and status codes in `server/routes/`
- Live logs via SSE (`EventSource` in `host.ts`)
- Imports: `@db`, `@native`
- Dialogs: `@native/dialog`

**wails_backend**

- Bound methods on `App` in `bind.go` (same operations as the HTTP API)
- Live logs via `runtime.EventsEmit` → `EventsOn("runnerEvent")` in `frontend/host.ts`
- After Go method/type changes: `wails generate module` (from `wails_backend/`)
- After Drizzle schema/query changes: copy new `drizzle/*.sql` into `internal/db/migrations/`, update `internal/db/schema.sql` + `internal/db/queries.sql`, then `sqlc generate`
- Dialogs and `openExternal` use the Wails runtime

Frontend-only UI that does not change `api` needs no backend dual update. A `db/` schema change is shared for the JS shell, but Wails sqlc must be updated in the same change if the tables or queries moved.
