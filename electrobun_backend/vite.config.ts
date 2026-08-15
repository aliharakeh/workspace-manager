import path from "node:path"
import { fileURLToPath } from "node:url"
import type { Plugin } from "vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(rootDir, "..")
const frontendDir = path.resolve(repoRoot, "frontend")
const hostFile = path.resolve(rootDir, "src/host.ts")

function resolveDepsFrom(hostImporter: string): Plugin {
	return {
		name: "resolve-deps-from-host",
		enforce: "pre",
		async resolveId(id, importer, options) {
			if (!id || id.startsWith("\0")) return null
			if (id.startsWith(".") || path.isAbsolute(id)) return null
			if (id === "@host" || id.startsWith("@/") || id.startsWith("@shared")) {
				return null
			}
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
			"@host": path.resolve(rootDir, "src/host.ts"),
			"@shared": path.resolve(rootDir, "src/shared"),
		},
	},
	build: {
		outDir: "dist",
		emptyOutDir: true,
	},
	server: {
		port: 5173,
		strictPort: true,
		fs: {
			allow: [repoRoot, rootDir, frontendDir],
		},
	},
})
