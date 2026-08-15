import { defineConfig } from "drizzle-kit"

export default defineConfig({
  dialect: "sqlite",
  schema: "./src/bun/db/schema.ts",
  out: "./drizzle",
  dbCredentials: {
    url: "./data/workspace-manager.sqlite",
  },
})
