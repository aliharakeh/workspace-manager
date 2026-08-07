import { useEffect, useState } from "react"
import { toast } from "sonner"
import { PlusIcon, Trash2Icon } from "lucide-react"
import { api } from "@/lib/api"
import type { RunMode } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"

type DraftCommand = {
  key: string
  label: string
  command: string
}

type RunConfigPanelProps = {
  appId: number
}

export function RunConfigPanel({ appId }: RunConfigPanelProps) {
  const [mode, setMode] = useState<RunMode>("parallel")
  const [commands, setCommands] = useState<DraftCommand[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const config = await api.runConfig.get(appId)
        if (cancelled) return
        setMode(config.mode)
        setCommands(
          config.commands.map((c) => ({
            key: String(c.id),
            label: c.label ?? "",
            command: c.command,
          }))
        )
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : "Failed to load run config"
        )
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [appId])

  async function handleSave() {
    if (commands.some((c) => !c.command.trim())) {
      toast.error("Each process needs a command")
      return
    }
    setSaving(true)
    try {
      const saved = await api.runConfig.save(appId, {
        mode,
        commands: commands.map((c) => ({
          label: c.label.trim() || null,
          command: c.command.trim(),
        })),
      })
      setMode(saved.mode)
      setCommands(
        saved.commands.map((c) => ({
          key: String(c.id),
          label: c.label ?? "",
          command: c.command,
        }))
      )
      toast.success("Run config saved")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading…</p>
  }

  return (
    <div className="flex flex-col gap-4">
      <Field>
        <FieldLabel>Execution mode</FieldLabel>
        <ToggleGroup
          value={[mode]}
          onValueChange={(v) => {
            if (v[0] === "sequential" || v[0] === "parallel") setMode(v[0])
          }}
          spacing={2}
        >
          <ToggleGroupItem value="parallel">Parallel</ToggleGroupItem>
          <ToggleGroupItem value="sequential">Sequential</ToggleGroupItem>
        </ToggleGroup>
        <p className="text-xs text-muted-foreground">
          {mode === "parallel"
            ? "All commands start together."
            : "Commands run one after another; a non-zero exit stops the chain."}
        </p>
      </Field>

      <div className="flex flex-col gap-2">
        {commands.map((cmd, index) => (
          <div
            key={cmd.key}
            className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center"
          >
            <span className="w-6 text-xs text-muted-foreground">{index + 1}</span>
            <Input
              className="sm:max-w-40"
              placeholder="Label"
              value={cmd.label}
              onChange={(e) =>
                setCommands((prev) =>
                  prev.map((c) =>
                    c.key === cmd.key ? { ...c, label: e.target.value } : c
                  )
                )
              }
            />
            <Input
              className="flex-1 font-mono"
              placeholder="npm run dev"
              value={cmd.command}
              onChange={(e) =>
                setCommands((prev) =>
                  prev.map((c) =>
                    c.key === cmd.key ? { ...c, command: e.target.value } : c
                  )
                )
              }
            />
            <Button
              variant="ghost"
              size="icon"
              onClick={() =>
                setCommands((prev) => prev.filter((c) => c.key !== cmd.key))
              }
            >
              <Trash2Icon />
            </Button>
          </div>
        ))}
      </div>

      <div className="flex gap-2">
        <Button
          variant="outline"
          onClick={() =>
            setCommands((prev) => [
              ...prev,
              {
                key: `new-${Date.now()}`,
                label: "",
                command: "",
              },
            ])
          }
        >
          <PlusIcon data-icon="inline-start" />
          Add command
        </Button>
        <Button disabled={saving} onClick={() => void handleSave()}>
          {saving ? "Saving…" : "Save run config"}
        </Button>
      </div>
    </div>
  )
}
