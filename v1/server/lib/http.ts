export function json(data: unknown, init: ResponseInit = {}) {
  return Response.json(data, init)
}

export function error(message: string, status = 400) {
  return json({ error: message }, { status })
}

export function notFound(message = "Not found") {
  return error(message, 404)
}

export async function readJson<T>(req: Request): Promise<T | null> {
  try {
    return (await req.json()) as T
  } catch {
    return null
  }
}

export function parseId(value: string | undefined): number | null {
  if (!value) return null
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : null
}

export type RouteMatch = {
  params: Record<string, string>
}

export function matchRoute(
  pattern: string,
  pathname: string
): RouteMatch | null {
  const patternParts = pattern.split("/").filter(Boolean)
  const pathParts = pathname.split("/").filter(Boolean)
  if (patternParts.length !== pathParts.length) return null

  const params: Record<string, string> = {}
  for (let i = 0; i < patternParts.length; i++) {
    const pp = patternParts[i]!
    const vp = pathParts[i]!
    if (pp.startsWith(":")) {
      params[pp.slice(1)] = decodeURIComponent(vp)
    } else if (pp !== vp) {
      return null
    }
  }
  return { params }
}
