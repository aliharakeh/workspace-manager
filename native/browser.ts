/**
 * Open the OS default browser and detect Bun standalone binaries.
 */

import { basename } from "node:path"

/** True when running the Bun `--compile` binary (not `bun run …`). */
export function isStandaloneBinary() {
  const name = basename(process.execPath).toLowerCase()
  return name !== "bun" && name !== "bun.exe"
}

/** Open `url` in the OS default browser (best-effort). */
export function openBrowser(url: string) {
  const args =
    process.platform === "darwin"
      ? ["open", url]
      : process.platform === "win32"
        ? ["cmd", "/c", "start", "", url]
        : ["xdg-open", url]

  Bun.spawn(args, {
    stdout: "ignore",
    stderr: "ignore",
    stdin: "ignore",
  }).unref()
}
