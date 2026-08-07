import { health } from "./routes/health"

const port = Number(process.env.PORT) || 3000

const server = Bun.serve({
  port,
  fetch(req) {
    const { pathname } = new URL(req.url)

    if (req.method === "GET" && pathname === "/api/health") {
      return health()
    }

    if (pathname.startsWith("/api/")) {
      return Response.json({ error: "Not found" }, { status: 404 })
    }

    return Response.json({ error: "Not found" }, { status: 404 })
  },
})

console.log(`API listening on http://localhost:${server.port}`)
