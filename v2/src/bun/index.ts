import { BrowserWindow, Updater } from "electrobun/bun"
import "./db"
import { readyUrlPatternsRepo } from "./db/ready-url-patterns"
import { createAppRPC } from "./rpc"
import { setRunnerBroadcast } from "./services/runner"

readyUrlPatternsRepo.ensureSeeded()

const DEV_SERVER_PORT = 5173
const DEV_SERVER_URL = `http://localhost:${DEV_SERVER_PORT}`

async function getMainViewUrl() {
  const channel = await Updater.localInfo.channel()
  if (channel === "dev") {
    try {
      await fetch(DEV_SERVER_URL, { method: "HEAD" })
      console.log(`HMR enabled: Using Vite dev server at ${DEV_SERVER_URL}`)
      return DEV_SERVER_URL
    } catch {
      console.log(
        "Vite dev server not running. Run 'bun run dev:hmr' for HMR support."
      )
    }
  }
  return "views://mainview/index.html"
}

const rpc = createAppRPC()
const url = await getMainViewUrl()

const mainWindow = new BrowserWindow<typeof rpc>({
  title: "App Runner",
  url,
  rpc,
  frame: {
    width: 1280,
    height: 860,
    x: 120,
    y: 80,
  },
})

setRunnerBroadcast((appId, event) => {
  mainWindow.webview.rpc?.send.runnerEvent({ appId, event })
})

console.log("App Runner started")
