import { Electroview } from "electrobun/view"
import type { AppRPC } from "../../shared/rpc"
import type { RunnerEvent } from "./types"

type RunnerEventHandler = (appId: number, event: RunnerEvent) => void

const listeners = new Set<RunnerEventHandler>()

export function onRunnerEvent(handler: RunnerEventHandler) {
  listeners.add(handler)
  return () => {
    listeners.delete(handler)
  }
}

export const rpc = Electroview.defineRPC<AppRPC>({
  maxRequestTime: Infinity,
  handlers: {
    messages: {
      runnerEvent: ({ appId, event }) => {
        for (const handler of listeners) handler(appId, event)
      },
    },
  },
})

export const electroview = new Electroview({ rpc })
