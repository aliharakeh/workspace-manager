import { defineConfig } from "drizzle-kit"
import { join } from "node:path"
import { dataDir } from "./db/paths"

export default defineConfig({
  dialect: "sqlite",
  schema: "./db/schema.ts",
  out: "./drizzle",
  dbCredentials: {
    url: join(dataDir(), "workspace-manager.sqlite"),
  },
})
