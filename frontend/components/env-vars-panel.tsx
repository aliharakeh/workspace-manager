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
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { EnvVarOptionsDialog } from "@/components/env-var-options-dialog"
import { Input } from "@/components/ui/input"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
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
import {
  FileUpIcon,
  KeyRoundIcon,
  ListChecksIcon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react"
import { toast } from "sonner"

type EnvVarsPanelProps = {
  appId: number
  /**
   * Bumped when a write to this app's data is reported (including this
   * window's own). The panel refreshes in place instead of remounting, so the
   * scroll position and unsaved edits stay.
   */
  dataEpoch: number
}

type EnvVarRowProps = {
  item: EnvVar
  deleting: boolean
  onSaved: (item: EnvVar) => void
  onRequestDelete: () => void
}

function EnvVarRow({
  item,
  deleting,
  onSaved,
  onRequestDelete,
}: EnvVarRowProps) {
  const [key, setKey] = useState(item.key)
  const [value, setValue] = useState(item.value)
  const [saving, setSaving] = useState(false)
  const [togglingAi, setTogglingAi] = useState(false)
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

  async function handleAiToggle(next: boolean) {
    if (next === item.include_in_ai) return
    setTogglingAi(true)
    try {
      const updated = await api.envVars.update(item.id, {
        include_in_ai: next,
      })
      onSaved(updated)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setTogglingAi(false)
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

  const busy = saving || deleting || togglingAi || switching

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center">
      <Input
        className="font-mono sm:max-w-48"
        value={key}
        disabled={saving || deleting || togglingAi}
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
            <SelectTrigger className="w-full font-mono sm:w-40" aria-label="Selected option">
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
          <Input
            className="flex-1 font-mono"
            value={item.value}
            readOnly
            disabled={busy}
            aria-label="Value of the selected option"
            title="Edit the values in Options"
          />
        </>
      ) : (
        <Input
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
            checked={item.include_in_ai}
            disabled={togglingAi || saving || deleting}
            onCheckedChange={(checked) =>
              void handleAiToggle(checked === true)
            }
            aria-label="Include in AI"
            title="Include this variable in AI chat"
          />
          AI
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
          disabled={!dirty || saving || deleting || togglingAi}
          onClick={() => void handleSave()}
          aria-label={saving ? "Saving" : "Save"}
          title={saving ? "Saving…" : "Save"}
        >
          <SaveIcon />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          disabled={deleting || saving || togglingAi}
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
            <AlertDialogTitle>Make “{item.key}” a plain value?</AlertDialogTitle>
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

export function EnvVarsPanel({ appId, dataEpoch }: EnvVarsPanelProps) {
  const [vars, setVars] = useState<EnvVar[]>([])
  const [key, setKey] = useState("")
  const [value, setValue] = useState("")
  const [query, setQuery] = useState("")
  const deferredQuery = useDeferredValue(query)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [importing, setImporting] = useState(false)
  const [includeInAi, setIncludeInAi] = useState(true)
  const [pendingDelete, setPendingDelete] = useState<EnvVar | null>(null)
  const [deleting, setDeleting] = useState(false)

  const filtered = useMemo(() => {
    const q = deferredQuery.trim().toLowerCase()
    if (!q) return vars
    return vars.filter(
      (v) =>
        v.key.toLowerCase().includes(q) ||
        v.value.toLowerCase().includes(q) ||
        v.options.some(
          (o) =>
            o.label.toLowerCase().includes(q) ||
            o.value.toLowerCase().includes(q)
        )
    )
  }, [vars, deferredQuery])

  // A silent load swaps the rows without the loading state, which would
  // replace the list and reset the scroll position.
  const loadSeq = useRef(0)
  async function load(silent = false) {
    const seq = ++loadSeq.current
    if (!silent) setLoading(true)
    try {
      const list = await api.envVars.list(appId)
      if (seq === loadSeq.current) setVars(list)
    } catch (err) {
      if (!silent)
        toast.error(
          err instanceof Error ? err.message : "Failed to load env vars"
        )
    } finally {
      if (!silent) setLoading(false)
    }
  }

  useEffect(() => {
    setPendingDelete(null)
    void load()
  }, [appId])

  const firstEpoch = useRef(true)
  useEffect(() => {
    if (firstEpoch.current) {
      firstEpoch.current = false
      return
    }
    void load(true)
  }, [dataEpoch])

  async function handleAdd() {
    if (!key.trim()) {
      toast.error("Key is required")
      return
    }
    setAdding(true)
    try {
      const created = await api.envVars.create(appId, {
        key: key.trim(),
        value,
        include_in_ai: includeInAi,
      })
      setVars((prev) =>
        [...prev, created].sort((a, b) => a.key.localeCompare(b.key))
      )
      setKey("")
      setValue("")
      setIncludeInAi(true)
      toast.success("Env var added")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add")
    } finally {
      setAdding(false)
    }
  }

  async function handleConfirmDelete() {
    if (!pendingDelete) return
    setDeleting(true)
    try {
      await api.envVars.delete(pendingDelete.id)
      setVars((prev) => prev.filter((v) => v.id !== pendingDelete.id))
      setPendingDelete(null)
      toast.success("Env var deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setDeleting(false)
    }
  }

  async function handleImport() {
    setImporting(true)
    try {
      const res = await api.envVars.importEnv(appId)
      if (res.cancelled) return
      setVars(res.vars ?? [])
      toast.success(
        `Imported ${res.imported ?? 0} variable(s) from ${res.path ?? "file"}`
      )
      if (res.template) {
        toast.success(
          res.template.created
            ? `Template added for ${res.template.file_path} with {{KEY}} placeholders`
            : `Template updated for ${res.template.file_path} with {{KEY}} placeholders`
        )
      } else {
        toast.warning("Template was not created for the imported file")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to import")
    } finally {
      setImporting(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading…</p>
  }

  return (
    <div className="flex flex-col gap-4">
      <FieldGroup className="rounded-lg border p-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <Field className="flex-1">
            <FieldLabel htmlFor="env-key">Key</FieldLabel>
            <Input
              id="env-key"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="API_URL"
              className="font-mono"
            />
          </Field>
          <Field className="flex-[2]">
            <FieldLabel htmlFor="env-value">Value</FieldLabel>
            <Input
              id="env-value"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="https://…"
              className="font-mono"
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleAdd()
              }}
            />
          </Field>
          <Field className="w-auto">
            <FieldLabel htmlFor="env-include-ai">AI</FieldLabel>
            <div className="flex h-8 items-center">
              <Checkbox
                id="env-include-ai"
                checked={includeInAi}
                onCheckedChange={(checked) => setIncludeInAi(checked === true)}
                aria-label="Include in AI"
                title="Include this variable in AI chat"
              />
            </div>
          </Field>
          <div className="flex gap-2">
            <Button disabled={adding} onClick={() => void handleAdd()}>
              <PlusIcon data-icon="inline-start" />
              {adding ? "Adding…" : "Add"}
            </Button>
            <Button
              variant="outline"
              disabled={importing}
              onClick={() => void handleImport()}
              title="Import variables and create a template from a .env, .yaml or .yml file"
            >
              <FileUpIcon data-icon="inline-start" />
              {importing ? "Importing…" : "Import file"}
            </Button>
          </div>
        </div>
      </FieldGroup>

      {vars.length === 0 ? (
        <Empty className="border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <KeyRoundIcon />
            </EmptyMedia>
            <EmptyTitle>No environment variables</EmptyTitle>
            <EmptyDescription>
              Add keys available to templates and run commands. Use Save on a
              row after editing.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <InputGroup className="max-w-sm flex-1">
              <InputGroupAddon align="inline-start">
                <SearchIcon />
              </InputGroupAddon>
              <InputGroupInput
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search key or value…"
                aria-label="Search env vars"
              />
              {query ? (
                <InputGroupAddon align="inline-end">
                  <InputGroupButton
                    size="icon-xs"
                    aria-label="Clear search"
                    onClick={() => setQuery("")}
                  >
                    <XIcon />
                  </InputGroupButton>
                </InputGroupAddon>
              ) : null}
            </InputGroup>
            <span className="text-xs text-muted-foreground">
              {filtered.length} of {vars.length}
            </span>
          </div>
          {filtered.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No env vars match “{query.trim()}”.
            </p>
          ) : (
            <div className="flex flex-col gap-2">
              {filtered.map((item) => (
                <EnvVarRow
                  key={item.id}
                  item={item}
                  deleting={deleting && pendingDelete?.id === item.id}
                  onSaved={(updated) =>
                    setVars((prev) =>
                      prev
                        .map((v) => (v.id === updated.id ? updated : v))
                        .sort((a, b) => a.key.localeCompare(b.key))
                    )
                  }
                  onRequestDelete={() => setPendingDelete(item)}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <AlertDialog
        open={!!pendingDelete}
        onOpenChange={(next) => {
          if (!next && !deleting) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete env var?</AlertDialogTitle>
            <AlertDialogDescription>
              This permanently deletes “{pendingDelete?.key}” from the active
              config set.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={(e) => {
                e.preventDefault()
                void handleConfirmDelete()
              }}
            >
              {deleting ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
