import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"
import type { EnvVar, EnvVarOption } from "@/lib/types"
import { CircleCheckIcon, CircleIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

type EnvVarOptionsDialogProps = {
  item: EnvVar
  open: boolean
  onOpenChange: (open: boolean) => void
  onSaved: (item: EnvVar) => void
  /** Saves the options; defaults to editing an app variable. A workspace shared variable passes its own. */
  saveOptions?: (
    id: number,
    body: { options: EnvVarOption[]; selected: string }
  ) => Promise<EnvVar>
}

type Draft = EnvVarOption & { key: number }

function seed(item: EnvVar): { rows: Draft[]; selected: number } {
  if (item.options.length === 0) {
    // A plain variable becomes a choice whose first option is its current value.
    return { rows: [{ key: 0, label: "default", value: item.value }], selected: 0 }
  }
  const rows = item.options.map((o, i) => ({ key: i, ...o }))
  const selected = Math.max(
    0,
    item.options.findIndex((o) => o.label === item.selected)
  )
  return { rows, selected }
}

/**
 * Edits the options of one variable. The dropdown in the list only switches
 * between them; adding, renaming and changing values happens here.
 */
export function EnvVarOptionsDialog({
  item,
  open,
  onOpenChange,
  onSaved,
  saveOptions = (id, body) => api.envVars.update(id, body),
}: EnvVarOptionsDialogProps) {
  const [rows, setRows] = useState<Draft[]>([])
  const [selected, setSelected] = useState(0)
  const [nextKey, setNextKey] = useState(1)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open) return
    const draft = seed(item)
    setRows(draft.rows)
    setSelected(draft.selected)
    setNextKey(draft.rows.length)
    // Only reseed when the dialog opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  function patch(key: number, change: Partial<EnvVarOption>) {
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, ...change } : r)))
  }

  function add() {
    setRows((prev) => [...prev, { key: nextKey, label: "", value: "" }])
    setNextKey((n) => n + 1)
  }

  function remove(key: number) {
    const index = rows.findIndex((r) => r.key === key)
    setRows((prev) => prev.filter((r) => r.key !== key))
    setSelected((cur) => (index < cur ? cur - 1 : Math.max(0, cur === index ? 0 : cur)))
  }

  async function save() {
    const options = rows.map((r) => ({ label: r.label.trim(), value: r.value }))
    if (options.length === 0) {
      toast.error("Add at least one option")
      return
    }
    if (options.some((o) => !o.label)) {
      toast.error("Every option needs a label")
      return
    }
    if (new Set(options.map((o) => o.label)).size !== options.length) {
      toast.error("Option labels must be different")
      return
    }
    setSaving(true)
    try {
      const updated = await saveOptions(item.id, {
        options,
        selected: options[Math.min(selected, options.length - 1)].label,
      })
      onSaved(updated)
      onOpenChange(false)
      toast.success("Options saved")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save options")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            Options for <span className="font-mono">{item.key}</span>
          </DialogTitle>
          <DialogDescription>
            The variable takes the value of the selected option. Use the circle
            to choose which option is in use.
          </DialogDescription>
        </DialogHeader>
        <div className="flex max-h-[60vh] flex-col gap-2 overflow-y-auto">
          {rows.map((row, index) => (
            <div key={row.key} className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="icon"
                disabled={saving}
                onClick={() => setSelected(index)}
                aria-label={
                  index === selected ? "Selected option" : "Select this option"
                }
                aria-pressed={index === selected}
                title={index === selected ? "In use" : "Use this option"}
              >
                {index === selected ? <CircleCheckIcon /> : <CircleIcon />}
              </Button>
              <Input
                className="w-36 font-mono"
                value={row.label}
                disabled={saving}
                placeholder="label"
                aria-label="Option label"
                onChange={(e) => patch(row.key, { label: e.target.value })}
              />
              <Input
                className="flex-1 font-mono"
                value={row.value}
                disabled={saving}
                placeholder="value"
                aria-label="Option value"
                onChange={(e) => patch(row.key, { value: e.target.value })}
              />
              <Button
                variant="ghost"
                size="icon"
                disabled={saving || rows.length === 1}
                onClick={() => remove(row.key)}
                aria-label="Remove option"
                title={
                  rows.length === 1
                    ? "Keep one option, or untick Options to make this a plain value"
                    : "Remove option"
                }
              >
                <Trash2Icon />
              </Button>
            </div>
          ))}
          <div>
            <Button variant="outline" disabled={saving} onClick={add}>
              <PlusIcon data-icon="inline-start" />
              Add option
            </Button>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            disabled={saving}
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void save()}>
            {saving ? "Saving…" : "Save options"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
