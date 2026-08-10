import { parse as parseYaml, stringify as stringifyYaml } from "yaml"
import { envFileToTemplate, parseEnvFile } from "./parse-env"

/**
 * A supported "import file" format. Each format knows how to:
 * - detect whether it handles a given file name
 * - parse the file content into flat key/value entries (imported as vars)
 * - build a Handlebars template referencing those keys (e.g. `{{KEY}}`)
 *
 * Register new formats (JSON, TOML, INI, …) by adding to `importFormats`.
 */
export type ImportEntry = { key: string; value: string }

export type ImportFormat = {
  /** Human-readable label used in errors/toasts, e.g. ".env", "YAML". */
  label: string
  /** Whether this format handles the given file name (case-insensitive). */
  matches(fileName: string): boolean
  /** Parse content into flat key/value entries. Throws with a clear message on invalid content. */
  parse(content: string): ImportEntry[]
  /** Build a Handlebars template from the file content referencing the imported keys. */
  toTemplate(content: string): string
}

/* ------------------------------------------------------------------ */
/* .env — KEY=VALUE lines                                              */
/* ------------------------------------------------------------------ */

const envFormat: ImportFormat = {
  label: ".env",
  matches: (fileName) => {
    const name = fileName.toLowerCase().split(/[\\/]/).pop() ?? ""
    // .env, .env.example, .env.local, .env.production, env, .envrc, …
    return name === "env" || name.startsWith(".env")
  },
  parse: (content) => parseEnvFile(content),
  toTemplate: (content) => envFileToTemplate(content),
}

/* ------------------------------------------------------------------ */
/* YAML — nested maps/arrays flattened into dot-notation keys          */
/* ------------------------------------------------------------------ */

function flattenYaml(value: unknown, prefix: string, out: ImportEntry[]): void {
  if (value === null || value === undefined) return
  if (Array.isArray(value)) {
    value.forEach((v, i) =>
      flattenYaml(v, prefix ? `${prefix}.${i}` : String(i), out)
    )
    return
  }
  if (typeof value === "object") {
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      flattenYaml(v, prefix ? `${prefix}.${k}` : k, out)
    }
    return
  }
  out.push({ key: prefix, value: String(value) })
}

/** Replace every scalar leaf with `{{[flat.key.path]}}` so it resolves from the flat env record. */
function templateifyYaml(value: unknown, prefix: string): unknown {
  if (Array.isArray(value)) {
    return value.map((v, i) =>
      templateifyYaml(v, prefix ? `${prefix}.${i}` : String(i))
    )
  }
  if (value !== null && typeof value === "object") {
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      out[k] = templateifyYaml(v, prefix ? `${prefix}.${k}` : k)
    }
    return out
  }
  return `{{[${prefix}]}}`
}

function parseYamlSafe(content: string): unknown {
  try {
    return parseYaml(content)
  } catch (err) {
    throw new Error(
      `Invalid YAML: ${err instanceof Error ? err.message : "parse error"}`,
      { cause: err }
    )
  }
}

const yamlFormat: ImportFormat = {
  label: "YAML",
  matches: (fileName) => {
    const name = fileName.toLowerCase()
    return name.endsWith(".yaml") || name.endsWith(".yml")
  },
  parse: (content) => {
    const data = parseYamlSafe(content)
    const entries: ImportEntry[] = []
    flattenYaml(data, "", entries)
    return entries
  },
  toTemplate: (content) => {
    const data = parseYamlSafe(content)
    const tpl = stringifyYaml(templateifyYaml(data, ""), { lineWidth: 0 })
    // The YAML serializer quotes {{[...]}} placeholders because they start
    // with "{" (flow-mapping syntax). Strip those quotes — Handlebars
    // substitutes the real values before the file is used, so the template
    // itself doesn't need to stay valid YAML (and matches the .env style).
    return tpl.replace(/"({{[^}]*}})"/g, "$1") + "\n"
  },
}

/* ------------------------------------------------------------------ */
/* Registry                                                            */
/* ------------------------------------------------------------------ */

export const importFormats: ImportFormat[] = [envFormat, yamlFormat]

/** Pick the first registered format that handles the given file name. */
export function detectImportFormat(fileName: string): ImportFormat | null {
  return importFormats.find((f) => f.matches(fileName)) ?? null
}
