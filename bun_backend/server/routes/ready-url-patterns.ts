import { readyUrlPatternsRepo } from "@db/ready-url-patterns"
import { error, json, notFound, parseId, readJson } from "../lib/http"
import {
  invalidateReadyUrlPatternsCache,
  validateReadyUrlPattern,
} from "../services/ready-url"

type Body = {
  label?: string
  pattern?: string
  flags?: string
}

function normalizeFlags(flags: string | undefined): string {
  const next = (flags ?? "i").trim() || "i"
  if (!/^[gimsuy]*$/.test(next)) {
    throw new Error("Invalid regex flags")
  }
  return next
}

export async function handleReadyUrlPatterns(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/ready-url-patterns") {
    if (req.method === "GET") {
      readyUrlPatternsRepo.ensureSeeded()
      return json(readyUrlPatternsRepo.list())
    }

    if (req.method === "POST") {
      const body = await readJson<Body>(req)
      if (!body?.label?.trim()) return error("label is required")
      if (!body.pattern?.trim()) return error("pattern is required")

      let flags: string
      try {
        flags = normalizeFlags(body.flags)
      } catch (err) {
        return error(err instanceof Error ? err.message : "Invalid flags")
      }

      const validated = validateReadyUrlPattern(body.pattern.trim(), flags)
      if (!validated.ok) return error(validated.error)

      const row = readyUrlPatternsRepo.create({
        label: body.label.trim(),
        pattern: body.pattern.trim(),
        flags,
      })
      invalidateReadyUrlPatternsCache()
      return json(row, { status: 201 })
    }

    return null
  }

  const itemMatch = pathname.match(/^\/api\/ready-url-patterns\/(\d+)$/)
  if (!itemMatch) return null

  const id = parseId(itemMatch[1])
  if (!id) return error("Invalid pattern id")

  if (req.method === "PATCH") {
    const body = await readJson<Body>(req)
    if (!body) return error("Invalid JSON body")

    const existing = readyUrlPatternsRepo.get(id)
    if (!existing) return notFound("Pattern not found")

    const label =
      body.label !== undefined ? body.label.trim() : existing.label
    const pattern =
      body.pattern !== undefined ? body.pattern.trim() : existing.pattern

    let flags: string
    try {
      flags =
        body.flags !== undefined
          ? normalizeFlags(body.flags)
          : existing.flags
    } catch (err) {
      return error(err instanceof Error ? err.message : "Invalid flags")
    }

    if (!label) return error("label cannot be empty")
    if (!pattern) return error("pattern cannot be empty")

    const validated = validateReadyUrlPattern(pattern, flags)
    if (!validated.ok) return error(validated.error)

    const row = readyUrlPatternsRepo.update(id, { label, pattern, flags })
    invalidateReadyUrlPatternsCache()
    return json(row)
  }

  if (req.method === "DELETE") {
    if (!readyUrlPatternsRepo.delete(id)) return notFound("Pattern not found")
    invalidateReadyUrlPatternsCache()
    return new Response(null, { status: 204 })
  }

  return null
}
