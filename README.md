# React + TypeScript + Vite + shadcn/ui + Bun API

Frontend is the Vite + React + shadcn app at the repo root. Backend is a zero-dependency Bun server in `server/`.

## Structure

```text
src/          React frontend (shadcn)
server/       Bun.serve API
scripts/dev.ts  runs web + API together
```

## Development

Run both (Vite on `:5173`, API on `:3000`, `/api` proxied):

```bash
bun run dev
```

Or separately:

```bash
bun run dev:web
bun run dev:server
```

Smoke-check the API (via the Vite proxy while `dev` is running, or hit Bun directly):

```bash
curl http://localhost:3000/api/health
# → {"ok":true}

# through Vite proxy:
curl http://localhost:5173/api/health
```

From the frontend:

```ts
const res = await fetch("/api/health")
const data = await res.json() // { ok: true }
```

## Adding components

```bash
npx shadcn@latest add button
```

Components land in `src/components`. Import with:

```tsx
import { Button } from "@/components/ui/button"
```
