import { useCallback, useEffect, useState } from "react"
import { CloudDownloadIcon, ZapIcon } from "lucide-react"
import { toast } from "sonner"
import { api, onDataChanged } from "@/lib/api"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type ScriptRunButtonProps = {
  /** True for a workspace shared set's script, false for an app config set's. */
  shared: boolean
  setId: number
  /** Shared sets only: its name, shown in the tooltip. */
  setName?: string | null
  /** Icon only (for cards) or icon with a label (for the variables panel). */
  iconOnly?: boolean
  className?: string
}

/**
 * Runs the dynamic environment script of a config set and saves its output as
 * variables. Renders nothing while the set has no script. The own script of an
 * app config set and the script of the shared set it links to use different
 * icons.
 */
export function ScriptRunButton({
  shared,
  setId,
  setName,
  iconOnly = true,
  className,
}: ScriptRunButtonProps) {
  const [configured, setConfigured] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    api.envProvider
      .get(shared, setId)
      .then((p) => setConfigured(p.command.trim() !== ""))
      .catch(() => setConfigured(false))
  }, [shared, setId])

  useEffect(() => {
    load()
    return onDataChanged((event) => {
      if (["envVar", "configSet", "sharedSet"].includes(event.entity)) load()
    })
  }, [load])

  if (!configured) return null

  const Icon = shared ? CloudDownloadIcon : ZapIcon
  const label = shared
    ? `Run linked script${setName ? ` (${setName})` : ""}`
    : "Run script"
  const hint = shared
    ? "Run the script of the linked shared config set and refresh its variables"
    : "Run this config set's script and refresh its variables"

  async function run() {
    setBusy(true)
    try {
      await api.envProvider.run(shared, setId)
      toast.success(
        shared ? "Linked script ran; variables refreshed" : "Script ran; variables refreshed"
      )
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const click = (event: { stopPropagation: () => void }) => {
    event.stopPropagation()
    void run()
  }

  if (!iconOnly) {
    return (
      <Button
        variant="outline"
        size="sm"
        disabled={busy}
        onClick={click}
        title={hint}
        className={className}
      >
        <Icon data-icon="inline-start" />
        {busy ? "Running…" : label}
      </Button>
    )
  }
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label={label}
            disabled={busy}
            onClick={click}
            className={className}
          />
        }
      >
        <Icon className={cn(busy && "animate-pulse")} />
      </TooltipTrigger>
      <TooltipContent>{busy ? "Running…" : label}</TooltipContent>
    </Tooltip>
  )
}

type AppScriptButtonsProps = {
  appId: number
  activeConfigSetId: number | null
  className?: string
}

/**
 * The script actions of an app's active config set: its own script and the
 * script of the shared set it links to, each shown only when configured.
 */
export function AppScriptButtons({
  appId,
  activeConfigSetId,
  className,
}: AppScriptButtonsProps) {
  const [linked, setLinked] = useState<{ id: number; name: string | null } | null>(
    null
  )

  const load = useCallback(() => {
    if (activeConfigSetId == null) return
    api.configSets
      .list(appId)
      .then((sets) => {
        const set = sets.find((s) => s.id === activeConfigSetId)
        setLinked(
          set?.shared_set_id != null
            ? { id: set.shared_set_id, name: set.shared_set_name }
            : null
        )
      })
      .catch(() => setLinked(null))
  }, [appId, activeConfigSetId])

  useEffect(() => {
    load()
    return onDataChanged((event) => {
      if (["configSet", "sharedSet"].includes(event.entity)) load()
    })
  }, [load])

  if (activeConfigSetId == null) return null
  return (
    <div className={cn("flex items-center gap-1", className)}>
      <ScriptRunButton shared={false} setId={activeConfigSetId} />
      {linked ? (
        <ScriptRunButton shared setId={linked.id} setName={linked.name} />
      ) : null}
    </div>
  )
}
