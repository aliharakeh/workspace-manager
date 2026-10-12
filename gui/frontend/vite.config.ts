import path from "node:path"
import { fileURLToPath } from "node:url"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  root: rootDir,
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": rootDir,
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5174,
    strictPort: true,
  },
})
