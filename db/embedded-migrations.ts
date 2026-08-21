/**
 * Populated by `bun run compile` with `with { type: "file" }` imports so
 * drizzle migrations ship inside the bun_backend standalone binary.
 * Empty in normal `bun` / `vite` workflows — those read from disk instead.
 */
export const migrationFiles: Record<string, string> = {}
