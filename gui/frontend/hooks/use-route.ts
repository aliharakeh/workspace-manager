import { useCallback, useEffect, useState } from "react"
import {
  parseRoute,
  type Route,
} from "@/lib/routes"

/** Split a full location string into pathname + search and parse it. */
function parseLocation(location: string): Route {
  const qIndex = location.indexOf("?")
  const pathname = qIndex === -1 ? location : location.slice(0, qIndex)
  const search = qIndex === -1 ? "" : location.slice(qIndex)
  return parseRoute(pathname, search)
}

/**
 * Keeps the current workspace/app/tab/config-set selection in sync with the URL
 * so any navigation is reflected in the address bar and deep links land on the
 * same section. Handles the browser back/forward buttons via `popstate`.
 */
export function useRoute() {
  const [route, setRoute] = useState<Route>(() =>
    parseLocation(window.location.pathname + window.location.search)
  )

  useEffect(() => {
    const onPop = () =>
      setRoute(parseLocation(window.location.pathname + window.location.search))
    window.addEventListener("popstate", onPop)
    return () => window.removeEventListener("popstate", onPop)
  }, [])

  const navigate = useCallback((path: string) => {
    if (path === window.location.pathname + window.location.search) {
      setRoute(parseLocation(path))
      return
    }
    setRoute(parseLocation(path))
    window.history.pushState(null, "", path)
  }, [])

  return { route, navigate }
}
