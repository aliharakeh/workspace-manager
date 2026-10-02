import { api } from "@/host"

export { api, ApiError, onBlueprintLog, onRunnerEvent } from "@/host"

export function handleReadyUrlClick(
  event: { preventDefault(): void; stopPropagation(): void },
  url: string
) {
  event.preventDefault()
  event.stopPropagation()
  void api.openExternal(url)
}
