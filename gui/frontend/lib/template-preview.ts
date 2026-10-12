import Handlebars from "handlebars"

export function renderTemplatePreview(
  source: string,
  env: Record<string, string>
): { ok: true; text: string } | { ok: false; error: string } {
  try {
    const compiled = Handlebars.compile(source, { noEscape: true })
    return { ok: true, text: compiled(env) }
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : "Failed to render template",
    }
  }
}
