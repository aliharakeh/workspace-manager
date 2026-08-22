export type AppTab = "env" | "templates" | "run" | "ai" | "logs"

export const APP_TABS: AppTab[] = ["env", "templates", "run", "ai", "logs"]

export type Route = {
  workspaceId: number | null
  appId: number | null
  tab: AppTab | null
  /** Activated config set id for the selected app. Mirrors the app's active set. */
  configSetId: number | null
}

export const HOME_ROUTE: Route = {
  workspaceId: null,
  appId: null,
  tab: null,
  configSetId: null,
}

/** Human-readable path segment for a name, e.g. "My App" -> "my-app". */
export function slugify(name: string): string {
  return (
    name
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "item"
  )
}

/**
 * Parse a segment of the form "my-app-3". Ids are numeric so the slug part is
 * whatever precedes the trailing "-<digits>".
 */
function parseSegment(segment: string): { slug: string; id: number | null } {
  const match = /^(.*)-(\d+)$/.exec(segment)
  if (match) return { slug: match[1]!, id: Number(match[2]) }
  return { slug: segment, id: null }
}

function formatSegment(name: string, id: number): string {
  return `${slugify(name)}-${id}`
}

/**
 * Parse a location (pathname + search) into a Route. Works for deep links and
 * normal in-app navigation. Unknown/malformed segments fall back to HOME_ROUTE.
 *
 *   /w/<ws>-<wsId>                             -> workspace only
 *   /w/<ws>-<wsId>/a/<app>-<appId>             -> app, env tab
 *   /w/<ws>-<wsId>/a/<app>-<appId>/logs        -> app, logs tab
 *   ...?set=<configSetId>                      -> activated config set
 */
export function parseRoute(pathname: string, search?: string): Route {
  const parts = pathname.split("/").filter(Boolean)
  if (parts[0] !== "w" || parts.length < 2) return HOME_ROUTE

  const ws = parseSegment(parts[1]!)
  if (ws.id == null) return HOME_ROUTE

  let appId: number | null = null
  let tab: AppTab | null = null

  if (parts[2] === "a" && parts.length >= 4) {
    const app = parseSegment(parts[3]!)
    if (app.id != null) {
      appId = app.id
      const raw = parts[4]
      if (raw && (APP_TABS as string[]).includes(raw)) {
        tab = raw as AppTab
      }
    }
  }

  let configSetId: number | null = null
  if (search) {
    const setParam = new URLSearchParams(search).get("set")
    if (setParam && /^\d+$/.test(setParam)) {
      configSetId = Number(setParam)
    }
  }

  return { workspaceId: ws.id, appId, tab, configSetId }
}

/**
 * Build a location (path + search) for a Route. Names are optional and only
 * affect the slug; resolution always uses ids.
 */
export function formatRoute(
  route: Route,
  workspaceName?: string,
  appName?: string
): string {
  if (route.workspaceId == null) return "/"
  const wsName = workspaceName ?? `w${route.workspaceId}`
  let path = `/w/${formatSegment(wsName, route.workspaceId)}`
  if (route.appId != null) {
    const name = appName ?? `a${route.appId}`
    path += `/a/${formatSegment(name, route.appId)}`
    if (route.tab) path += `/${route.tab}`
    if (route.configSetId != null) path += `?set=${route.configSetId}`
  }
  return path
}
