import path from "node:path"
import { fileURLToPath } from "node:url"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
	plugins: [react(), tailwindcss()],
	root: "src/mainview",
	resolve: {
		alias: {
			"@": path.resolve(rootDir, "src/mainview"),
			"@shared": path.resolve(rootDir, "src/shared"),
		},
	},
	build: {
		outDir: "../../dist",
		emptyOutDir: true,
	},
	server: {
		port: 5173,
		strictPort: true,
		fs: {
			allow: [rootDir],
		},
	},
})
