import { useEffect, useState } from "react"
import { toast } from "sonner"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"
import { api } from "@/lib/api"
import type { App, ConfigSet } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type ConfigSetPickerProps = {
  app: App
  onAppChange: (app: App) => void
  /** Stop click from selecting the app row. */
  stopPropagation?: boolean
}

/** Compact activate-only picker for workspace overview rows. */
export function ConfigSetPicker({
  app,
  onAppChange,
  stopPropagation,
}: ConfigSetPickerProps) {
  const [sets, setSets] = useState<ConfigSet[]>([])
  const [loading, setLoading] = useState(true)

  const activeName =
    app.active_config_set_name ??
    sets.find((s) => s.id === app.active_config_set_id)?.name ??
    "—"

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const list = await api.configSets.list(app.id)
        if (!cancelled) setSets(list)
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof Error ? err.message : "Failed to load config sets"
          )
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [app.id, app.active_config_set_id])

  async function activate(id: number) {
    try {
      const result = await api.configSets.activate(id)
      onAppChange(result.app)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to activate")
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            className="max-w-44"
            disabled={loading}
            onClick={(e) => {
              if (stopPropagation) e.stopPropagation()
            }}
          />
        }
      >
        <span className="truncate">{activeName}</span>
        <ChevronsUpDownIcon data-icon="inline-end" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="min-w-40"
        onClick={(e) => {
          if (stopPropagation) e.stopPropagation()
        }}
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>Config set</DropdownMenuLabel>
          {sets.map((set) => (
            <DropdownMenuItem
              key={set.id}
              onClick={() => void activate(set.id)}
            >
              <span className="truncate">{set.name}</span>
              {set.id === app.active_config_set_id ? (
                <CheckIcon className="ml-auto" />
              ) : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
