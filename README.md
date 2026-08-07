# React + TypeScript + Vite + shadcn/ui + Bun API

Frontend is the Vite + React + shadcn app at the repo root. Backend is a Bun server in `server/` with SQLite persistence.

## Structure

```text
src/          React frontend (shadcn)
server/       Bun.serve API + SQLite + process runner + static files
dist/         Production frontend build (served by Bun)
data/         Local SQLite DB and template backups (gitignored)
scripts/      Dev / production launchers
```

## Development

API + Vite HMR. Preferred ports `3000` / `5173`; if busy, the next free ports are chosen automatically:

```bash
bun run dev
```

Or separately:

```bash
bun run dev:server
bun run dev:web
```

## Production

Build the frontend and serve it from the Bun server (single process, no Vite):

```bash
bun run start
```

Or build once, then run the server again:

```bash
bun run build
bun run start:server
```

Open the printed URL (API and UI share the same origin; `/api` is handled by Bun, everything else from `dist/`).

## Features

- **Workspaces** — named groups of apps
- **Apps** — name + local `project_path`
- **Env vars** — per-app key/value pairs
- **Templates** — Handlebars templates applied to project files on Run (with backup/restore on Stop)
- **Run config** — one config per app, multiple commands, sequential or parallel
- **Logs** — SSE-streamed stdout/stderr per process tab
