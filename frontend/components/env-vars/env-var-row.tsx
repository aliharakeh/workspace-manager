import { EnvVarOptionsDialog } from "@/components/env-var-options-dialog"
import { SecretInput } from "@/components/secret-input"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api } from "@/lib/api"
import type { EnvVar } from "@/lib/types"
import { ListChecksIcon, SaveIcon, Trash2Icon } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"
type EnvVarRowProps = {
  item: EnvVar
  deleting: boolean
  onSaved: (item: EnvVar) => void
  onRequestDelete: () => void
}
export function EnvVarRow({
  item,
  deleting,
  onSaved,
  onRequestDelete,
}: EnvVarRowProps) {
  const [key, setKey] = useState(item.key)
  const [value, setValue] = useState(item.value)
  const [saving, setSaving] = useState(false)
  const [togglingPrivate, setTogglingPrivate] = useState(false)
  const [optionsOpen, setOptionsOpen] = useState(false)
  const [confirmPlain, setConfirmPlain] = useState(false)
  const [switching, setSwitching] = useState(false)
  // A variable with options is a choice: its value is picked, not typed.
  const isChoice = item.options.length > 0

  useEffect(() => {
    setKey(item.key)
    setValue(item.value)
  }, [item.id, item.key, item.value])

  const dirty = key.trim() !== item.key || (!isChoice && value !== item.value)

  async function handleSave() {
    const nextKey = key.trim()
    if (!nextKey) {
      toast.error("Key is required")
      return
    }
    if (!dirty) return

    setSaving(true)
    try {
      const updated = await api.envVars.update(
        item.id,
        isChoice ? { key: nextKey } : { key: nextKey, value }
      )
      onSaved(updated)
      toast.success("Env var updated")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setSaving(false)
    }
  }

  async function handlePrivateToggle(next: boolean) {
    if (next === item.private) return
    setTogglingPrivate(true)
    try {
      const updated = await api.envVars.update(item.id, {
        private: next,
      })
      onSaved(updated)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setTogglingPrivate(false)
    }
  }

  async function handleSelect(label: string) {
    if (label === item.selected) return
    setSwitching(true)
    try {
      onSaved(await api.envVars.update(item.id, { selected: label }))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to switch")
    } finally {
      setSwitching(false)
    }
  }

  async function handleMakePlain() {
    setSwitching(true)
    try {
      onSaved(await api.envVars.update(item.id, { options: [] }))
      setConfirmPlain(false)
      toast.success("Now a plain value")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setSwitching(false)
    }
  }

  const busy = saving || deleting || togglingPrivate || switching

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center">
      <Input
        className="font-mono sm:max-w-48"
        value={key}
        disabled={saving || deleting || togglingPrivate}
        onChange={(e) => setKey(e.target.value)}
        aria-label="Variable key"
      />
      {isChoice ? (
        <>
          <Select
            items={item.options.map((o) => ({
              value: o.label,
              label: o.label,
            }))}
            value={item.selected}
            onValueChange={(next) => {
              if (next !== null) void handleSelect(next)
            }}
            disabled={busy}
          >
            <SelectTrigger
              className="w-full font-mono sm:w-40"
              aria-label="Selected option"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {item.options.map((o) => (
                  <SelectItem key={o.label} value={o.label}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <SecretInput
            secret={item.private}
            className="flex-1 font-mono"
            value={item.value}
            readOnly
            disabled={busy}
            aria-label="Value of the selected option"
            title="Edit the values in Options"
          />
        </>
      ) : (
        <SecretInput
          secret={item.private}
          className="flex-1 font-mono"
          value={value}
          disabled={busy}
          onChange={(e) => setValue(e.target.value)}
          aria-label="Variable value"
        />
      )}
      <div className="flex items-center gap-1">
        <label className="flex items-center gap-1.5 pr-1 text-xs text-muted-foreground">
          <Checkbox
            checked={item.private}
            disabled={togglingPrivate || saving || deleting}
            onCheckedChange={(checked) =>
              void handlePrivateToggle(checked === true)
            }
            aria-label="Private"
            title="Private: masked here, and never shown to AI or agents"
          />
          Private
        </label>
        <label className="flex items-center gap-1.5 pr-1 text-xs text-muted-foreground">
          <Checkbox
            checked={isChoice}
            disabled={busy}
            onCheckedChange={(checked) => {
              if (checked === true) setOptionsOpen(true)
              else setConfirmPlain(true)
            }}
            aria-label="Choose from options"
            title="Pick the value from a list of options instead of typing it"
          />
          Options
        </label>
        <Button
          variant="ghost"
          size="icon"
          disabled={busy}
          onClick={() => setOptionsOpen(true)}
          aria-label="Edit options"
          title="Edit options"
        >
          <ListChecksIcon />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          disabled={!dirty || saving || deleting || togglingPrivate}
          onClick={() => void handleSave()}
          aria-label={saving ? "Saving" : "Save"}
          title={saving ? "Saving…" : "Save"}
        >
          <SaveIcon />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          disabled={deleting || saving || togglingPrivate}
          onClick={onRequestDelete}
          aria-label="Delete"
          title="Delete"
        >
          <Trash2Icon />
        </Button>
      </div>
      <EnvVarOptionsDialog
        item={item}
        open={optionsOpen}
        onOpenChange={setOptionsOpen}
        onSaved={onSaved}
      />
      <AlertDialog
        open={confirmPlain}
        onOpenChange={(next) => {
          if (!next && !switching) setConfirmPlain(false)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Make “{item.key}” a plain value?
            </AlertDialogTitle>
            <AlertDialogDescription>
              The other options are removed. The current value stays as it is.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={switching}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={switching}
              onClick={(e) => {
                e.preventDefault()
                void handleMakePlain()
              }}
            >
              Remove options
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
