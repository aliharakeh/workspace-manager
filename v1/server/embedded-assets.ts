/**
 * Populated by `bun run compile` with `with { type: "file" }` imports so the
 * frontend and drizzle migrations ship inside the standalone binary.
 * Empty in normal `bun` / `vite` workflows — those read from disk instead.
 */
export const staticFiles: Record<string, string> = {}
export const migrationFiles: Record<string, string> = {}
