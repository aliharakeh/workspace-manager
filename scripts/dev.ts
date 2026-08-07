const children = [
  Bun.spawn(["bun", "run", "--hot", "server/index.ts"], {
    stdout: "inherit",
    stderr: "inherit",
    stdin: "inherit",
  }),
  Bun.spawn(["bunx", "vite"], {
    stdout: "inherit",
    stderr: "inherit",
    stdin: "inherit",
  }),
]

function shutdown() {
  for (const child of children) {
    child.kill()
  }
  process.exit(0)
}

process.on("SIGINT", shutdown)
process.on("SIGTERM", shutdown)

await Promise.all(children.map((child) => child.exited))
