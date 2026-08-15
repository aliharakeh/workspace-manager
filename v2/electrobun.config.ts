import type { ElectrobunConfig } from "electrobun"

export default {
	app: {
		name: "App Runner",
		identifier: "app-runner.electrobun.dev",
		version: "0.0.1",
	},
	build: {
		copy: {
			"dist/index.html": "views/mainview/index.html",
			"dist/assets": "views/mainview/assets",
			drizzle: "drizzle",
		},
		watchIgnore: ["dist/**", "data/**"],
		mac: {
			bundleCEF: false,
		},
		linux: {
			bundleCEF: false,
		},
		win: {
			bundleCEF: false,
		},
	},
} satisfies ElectrobunConfig
