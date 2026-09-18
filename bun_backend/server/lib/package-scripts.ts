import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

/** Lockfile → package manager, checked in order. */
const LOCKFILES: Array<[string, string]> = [
  ["bun.lock", "bun"],
  ["bun.lockb", "bun"],
  ["pnpm-lock.yaml", "pnpm"],
  ["yarn.lock", "yarn"],
  ["package-lock.json", "npm"],
]

export type PackageScript = {
  name: string
  script: string
  command: string
}

export type PackageScripts = {
  has_package_json: boolean
  package_manager: string
  scripts: PackageScript[]
}

export function detectPackageManager(projectPath: string): string {
  for (const [file, manager] of LOCKFILES) {
    if (existsSync(join(projectPath, file))) return manager
  }
  return "npm"
}

/** Read the `scripts` of a project's package.json, if it has one. */
export function readPackageScripts(projectPath: string): PackageScripts {
  const packageManager = detectPackageManager(projectPath)
  const missing: PackageScripts = {
    has_package_json: false,
    package_manager: packageManager,
    scripts: [],
  }

  let raw: string
  try {
    raw = readFileSync(join(projectPath, "package.json"), "utf8")
  } catch {
    return missing
  }

  const found = { ...missing, has_package_json: true }

  let parsed: { scripts?: unknown }
  try {
    parsed = JSON.parse(raw) as { scripts?: unknown }
  } catch {
    return found
  }

  const scripts = parsed.scripts
  if (!scripts || typeof scripts !== "object") return found

  return {
    ...found,
    scripts: Object.entries(scripts)
      .filter((entry): entry is [string, string] => typeof entry[1] === "string")
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, script]) => ({
        name,
        script,
        command: `${packageManager} run ${name}`,
      })),
  }
}
