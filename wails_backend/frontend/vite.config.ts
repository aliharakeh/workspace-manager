import path from "node:path"
import { fileURLToPath } from "node:url"
import type { Plugin } from "vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(rootDir, "../..")
const frontendDir = path.resolve(repoRoot, "frontend")
const hostFile = path.resolve(rootDir, "host.ts")

function resolveDepsFrom(hostImporter: string): Plugin {
  return {
    name: "resolve-deps-from-host",
    enforce: "pre",
    async resolveId(id, importer, options) {
      if (!id || id.startsWith("\0")) return null
      if (id.startsWith(".") || path.isAbsolute(id)) return null
      if (id === "@host" || id.startsWith("@/")) return null
      if (!importer) return null
      const fromFrontend = path
        .normalize(importer)
        .startsWith(path.normalize(frontendDir))
      if (!fromFrontend) return null
      return this.resolve(id, hostImporter, {
        ...options,
        skipSelf: true,
      })
    },
  }
}

export default defineConfig({
  root: rootDir,
  plugins: [resolveDepsFrom(hostFile), react(), tailwindcss()],
  resolve: {
    alias: {
      "@": frontendDir,
      "@host": hostFile,
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5174,
    strictPort: true,
    fs: {
      allow: [repoRoot, rootDir, frontendDir],
    },
  },
})
