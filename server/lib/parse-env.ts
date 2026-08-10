export type ParsedEnvEntry = { key: string; value: string }

export class EnvParseError extends Error {
  line: number

  constructor(message: string, line: number) {
    super(`${message} (line ${line})`)
    this.line = line
  }
}

/**
 * Parse a `.env`-style file into key/value pairs.
 *
 * Supported syntax:
 * - `KEY=VALUE` and `export KEY=VALUE`
 * - Full-line comments (`# ...`)
 * - Trailing comments after an unquoted value (`KEY=value # note`)
 * - Single- or double-quoted values (`KEY="value"`)
 * - Blank lines (ignored)
 *
 * Throws `EnvParseError` when a content line is not a valid assignment.
 */
export function parseEnvFile(content: string): ParsedEnvEntry[] {
  const entries: ParsedEnvEntry[] = []
  const lines = content.split(/\r?\n/)

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i]!
    let trimmed = line.trim()

    if (!trimmed || trimmed.startsWith("#")) continue
    if (trimmed.startsWith("export ")) {
      line = trimmed.slice("export ".length)
      trimmed = line.trim()
    }

    const eq = trimmed.indexOf("=")
    if (eq <= 0) {
      throw new EnvParseError(
        "Not a valid env line (expected KEY=VALUE)",
        i + 1
      )
    }

    const key = trimmed.slice(0, eq).trim()
    if (!key) throw new EnvParseError("Empty variable key", i + 1)

    let value = trimmed.slice(eq + 1).trim()

    const quote = value[0]
    if (quote === '"' || quote === "'") {
      const end = value.lastIndexOf(quote)
      if (end <= 0) {
        throw new EnvParseError("Unclosed quote in value", i + 1)
      }
      value = value.slice(1, end)
    } else {
      const hash = value.indexOf(" #")
      if (hash !== -1) value = value.slice(0, hash).trim()
    }

    entries.push({ key, value })
  }

  return entries
}

/**
 * Build a Handlebars template from `.env`-style content, replacing each
 * variable's value with `{{KEY}}` so it resolves from the app's env vars at
 * render time. Comments and blank lines are preserved.
 */
export function envFileToTemplate(content: string): string {
  const out: string[] = []
  for (const raw of content.split(/\r?\n/)) {
    const trimmed = raw.trim()
    if (!trimmed || trimmed.startsWith("#")) {
      out.push(raw)
      continue
    }
    let head = trimmed
    let prefix = ""
    if (head.startsWith("export ")) {
      prefix = "export "
      head = head.slice("export ".length)
    }
    const eq = head.indexOf("=")
    if (eq <= 0) {
      out.push(raw)
      continue
    }
    const key = head.slice(0, eq).trim()
    out.push(`${prefix}${key}={{${key}}}`)
  }
  return out.join("\n") + "\n"
}
