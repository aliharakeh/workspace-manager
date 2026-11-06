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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { EnvVarOptionsDialog } from "@/components/env-var-options-dialog"
import { SecretInput } from "@/components/secret-input"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { api } from "@/lib/api"
import type { EnvVar, SharedEnvVar, SharedSet, SharedSetDetail } from "@/lib/types"
import {
  ListChecksIcon,
  PencilIcon,
  PlusIcon,
  SaveIcon,
  Trash2Icon,
} from "lucide-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"

type SharedConfigDialogProps = {
  workspaceId: number
  workspaceName: string
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Bumped when a shared config write is reported, so the dialog refreshes in place. */
  epoch: number
}

/** A shared variable viewed as an app variable, for the options dialog. */
function asEnvVar(v: SharedEnvVar): EnvVar {
  return {
    id: v.id,
    config_set_id: v.set_id,
    key: v.key,
    value: v.value,
    private: v.private,
    options: v.options,
    selected: v.selected,
    shared_var_id: null,
    created_at: v.created_at,
    updated_at: v.updated_at,
  }
}

/** "app / config set" lines of everywhere a shared variable is used. */
function usageLines(v: SharedEnvVar) {
  return v.used_by.map((u) => `${u.app_name} / ${u.config_set_name}`)
}

function UsedBy({ item }: { item: SharedEnvVar }) {
  if (item.used_by.length === 0) {
    return (
      <Badge variant="outline" className="shrink-0 text-muted-foreground">
        unused
      </Badge>
    )
  }
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Badge variant="secondary" className="shrink-0">
            used by {item.used_by.length}
          </Badge>
        }
      />
      <TooltipContent>
        <div className="flex flex-col">
          {usageLines(item).map((line) => (
            <span key={line}>{line}</span>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

type RowProps = {
  item: SharedEnvVar
  busy: boolean
  onChanged: () => void
  onRequestDelete: () => void
}

function SharedVarRow({ item, busy, onChanged, onRequestDelete }: RowProps) {
  const [key, setKey] = useState(item.key)
  const [value, setValue] = useState(item.value)
  const [saving, setSaving] = useState(false)
  const [optionsOpen, setOptionsOpen] = useState(false)
  const isChoice = item.options.length > 0

  useEffect(() => {
    setKey(item.key)
    setValue(item.value)
  }, [item.id, item.key, item.value])

  const dirty = key.trim() !== item.key || (!isChoice && value !== item.value)

  async function update(
    body: Parameters<typeof api.sharedSets.updateVar>[1],
    success?: string
  ) {
    setSaving(true)
    try {
      await api.sharedSets.updateVar(item.id, body)
      onChanged()
      if (success) toast.success(success)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setSaving(false)
    }
  }

  async function handleSave() {
    const nextKey = key.trim()
    if (!nextKey) {
      toast.error("Key is required")
      return
    }
    if (!dirty) return
    await update(
      isChoice ? { key: nextKey } : { key: nextKey, value },
      item.used_by.length > 0
        ? `Updated in ${item.used_by.length} config set${item.used_by.length === 1 ? "" : "s"}`
        : "Variable updated"
    )
  }

  const disabled = busy || saving

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center">
      <Input
        className="font-mono sm:max-w-48"
        value={key}
        disabled={disabled}
        onChange={(e) => setKey(e.target.value)}
        aria-label="Variable key"
      />
      <SecretInput
        secret={item.private}
        className="flex-1 font-mono"
        value={isChoice ? item.value : value}
        readOnly={isChoice}
        disabled={disabled}
        onChange={(e) => setValue(e.target.value)}
        aria-label="Variable value"
        title={isChoice ? `Option “${item.selected}” — edit it in Options` : undefined}
      />
      <div className="flex items-center gap-1">
        <UsedBy item={item} />
        <label className="flex items-center gap-1.5 pr-1 text-xs text-muted-foreground">
          <Checkbox
            checked={item.private}
            disabled={disabled}
            onCheckedChange={(checked) =>
              void update({ private: checked === true })
            }
            aria-label="Private"
            title="Private: masked here, and never shown to AI or agents"
          />
          Private
        </label>
        <Button
          variant="ghost"
          size="icon"
          disabled={disabled}
          onClick={() => setOptionsOpen(true)}
          aria-label="Edit options"
          title="Pick the value from a list of options"
        >
          <ListChecksIcon />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          disabled={!dirty || disabled}
          onClick={() => void handleSave()}
          aria-label="Save"
          title="Save"
        >
          <SaveIcon />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          disabled={disabled}
          onClick={onRequestDelete}
          aria-label="Delete"
          title="Delete"
        >
          <Trash2Icon />
        </Button>
      </div>
      <EnvVarOptionsDialog
        item={asEnvVar(item)}
        open={optionsOpen}
        onOpenChange={setOptionsOpen}
        onSaved={() => onChanged()}
        saveOptions={async (id, body) => {
          await api.sharedSets.updateVar(id, body)
          return asEnvVar(item)
        }}
      />
    </div>
  )
}

/**
 * The workspace's shared config: sets of variables that app config sets link to.
 * A change here reaches every linked config set at once; the apps only read them.
 */
export function SharedConfigDialog({
  workspaceId,
  workspaceName,
  open,
  onOpenChange,
  epoch,
}: SharedConfigDialogProps) {
  const [sets, setSets] = useState<SharedSet[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detail, setDetail] = useState<SharedSetDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [newSetName, setNewSetName] = useState("")
  const [creatingSet, setCreatingSet] = useState(false)
  const [renaming, setRenaming] = useState<string | null>(null)
  const [key, setKey] = useState("")
  const [value, setValue] = useState("")
  const [pendingDeleteVar, setPendingDeleteVar] = useState<SharedEnvVar | null>(null)
  const [pendingDeleteSet, setPendingDeleteSet] = useState(false)

  const selectedIdRef = useRef<number | null>(null)
  const seq = useRef(0)
  const load = useCallback(
    async (preferId?: number | null, silent = false) => {
      const mine = ++seq.current
      if (!silent) setLoading(true)
      try {
        const list = await api.sharedSets.list(workspaceId)
        if (mine !== seq.current) return
        setSets(list)
        const id =
          preferId !== undefined && list.some((s) => s.id === preferId)
            ? preferId
            : list.some((s) => s.id === selectedIdRef.current)
              ? selectedIdRef.current
              : (list[0]?.id ?? null)
        setSelectedId(id)
        selectedIdRef.current = id
        setDetail(id == null ? null : await api.sharedSets.get(id))
      } catch (err) {
        if (!silent)
          toast.error(err instanceof Error ? err.message : "Failed to load shared config")
      } finally {
        if (!silent && mine === seq.current) setLoading(false)
      }
    },
    [workspaceId]
  )
  useEffect(() => {
    if (!open) return
    selectedIdRef.current = null
    void load(null)
  }, [open, load])

  const firstEpoch = useRef(true)
  useEffect(() => {
    if (firstEpoch.current) {
      firstEpoch.current = false
      return
    }
    if (open) void load(undefined, true)
  }, [epoch])

  const selected = sets.find((s) => s.id === selectedId) ?? null
  const usedCount = new Set(
    (detail?.vars ?? []).flatMap((v) => v.used_by.map((u) => u.config_set_id))
  ).size

  async function run(task: () => Promise<void>, failure: string) {
    setBusy(true)
    try {
      await task()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : failure)
    } finally {
      setBusy(false)
    }
  }

  const handleCreateSet = () =>
    run(async () => {
      const name = newSetName.trim()
      if (!name) {
        toast.error("Name is required")
        return
      }
      const created = await api.sharedSets.create(workspaceId, name)
      setNewSetName("")
      setCreatingSet(false)
      await load(created.id, true)
      toast.success(`Created “${created.name}”`)
    }, "Failed to create")

  const handleRename = () =>
    run(async () => {
      if (!selected || renaming == null) return
      const name = renaming.trim()
      if (!name) {
        toast.error("Name is required")
        return
      }
      if (name !== selected.name) await api.sharedSets.rename(selected.id, name)
      setRenaming(null)
      await load(selected.id, true)
    }, "Failed to rename")

  const handleDeleteSet = () =>
    run(async () => {
      if (!selected) return
      await api.sharedSets.delete(selected.id)
      setPendingDeleteSet(false)
      await load(null, true)
      toast.success("Shared config set deleted")
    }, "Failed to delete")

  const handleAddVar = () =>
    run(async () => {
      if (!selected) return
      if (!key.trim()) {
        toast.error("Key is required")
        return
      }
      await api.sharedSets.createVar(selected.id, { key: key.trim(), value })
      setKey("")
      setValue("")
      await load(selected.id, true)
    }, "Failed to add")

  const handleDeleteVar = () =>
    run(async () => {
      if (!pendingDeleteVar || !selected) return
      await api.sharedSets.deleteVar(pendingDeleteVar.id)
      setPendingDeleteVar(null)
      await load(selected.id, true)
      toast.success("Variable deleted")
    }, "Failed to delete")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Shared config · {workspaceName}</DialogTitle>
          <DialogDescription>
            Variables every app in this workspace can use. An app config set links
            to one set here and picks the variables it needs; they are read-only
            in the app, and a change here reaches all of them straight away.
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto">
            <div className="flex flex-wrap items-center gap-2">
              {sets.length > 0 && selected && renaming == null ? (
                <Select
                  items={sets.map((s) => ({ value: String(s.id), label: s.name }))}
                  value={String(selected.id)}
                  onValueChange={(next) => {
                    if (next === null) return
                    selectedIdRef.current = Number(next)
                    void load(Number(next), true)
                  }}
                  disabled={busy}
                >
                  <SelectTrigger className="w-56" aria-label="Shared config set">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {sets.map((s) => (
                        <SelectItem key={s.id} value={String(s.id)}>
                          {s.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              ) : null}
              {renaming != null ? (
                <>
                  <Input
                    className="w-56"
                    value={renaming}
                    autoFocus
                    onChange={(e) => setRenaming(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") void handleRename()
                      if (e.key === "Escape") setRenaming(null)
                    }}
                    aria-label="Shared config set name"
                  />
                  <Button disabled={busy} onClick={() => void handleRename()}>
                    Save
                  </Button>
                  <Button variant="ghost" onClick={() => setRenaming(null)}>
                    Cancel
                  </Button>
                </>
              ) : selected ? (
                <>
                  <Button
                    variant="ghost"
                    size="icon"
                    disabled={busy}
                    onClick={() => setRenaming(selected.name)}
                    aria-label="Rename set"
                    title="Rename set"
                  >
                    <PencilIcon />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    disabled={busy}
                    onClick={() => setPendingDeleteSet(true)}
                    aria-label="Delete set"
                    title="Delete set"
                  >
                    <Trash2Icon />
                  </Button>
                </>
              ) : null}
              <div className="ml-auto flex items-center gap-2">
                {creatingSet ? (
                  <>
                    <Input
                      className="w-44"
                      value={newSetName}
                      autoFocus
                      placeholder="dev2"
                      onChange={(e) => setNewSetName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") void handleCreateSet()
                        if (e.key === "Escape") setCreatingSet(false)
                      }}
                      aria-label="New set name"
                    />
                    <Button disabled={busy} onClick={() => void handleCreateSet()}>
                      Create
                    </Button>
                    <Button variant="ghost" onClick={() => setCreatingSet(false)}>
                      Cancel
                    </Button>
                  </>
                ) : (
                  <Button variant="outline" disabled={busy} onClick={() => setCreatingSet(true)}>
                    <PlusIcon data-icon="inline-start" />
                    New set
                  </Button>
                )}
              </div>
            </div>

            {!selected ? (
              <p className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
                No shared config yet. Create a set (for example “dev”) and add the
                variables several apps need, like the database URL.
              </p>
            ) : (
              <>
                <div className="flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center">
                  <Input
                    className="font-mono sm:max-w-48"
                    value={key}
                    placeholder="DATABASE_URL"
                    onChange={(e) => setKey(e.target.value)}
                    aria-label="New variable key"
                  />
                  <Input
                    className="flex-1 font-mono"
                    value={value}
                    placeholder="value"
                    onChange={(e) => setValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") void handleAddVar()
                    }}
                    aria-label="New variable value"
                  />
                  <Button disabled={busy} onClick={() => void handleAddVar()}>
                    <PlusIcon data-icon="inline-start" />
                    Add
                  </Button>
                </div>
                {detail && detail.vars.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    This set has no variables yet.
                  </p>
                ) : (
                  <div className="flex flex-col gap-2">
                    {(detail?.vars ?? []).map((v) => (
                      <SharedVarRow
                        key={v.id}
                        item={v}
                        busy={busy}
                        onChanged={() => void load(selected.id, true)}
                        onRequestDelete={() => setPendingDeleteVar(v)}
                      />
                    ))}
                  </div>
                )}
              </>
            )}
          </div>
        )}

        <AlertDialog
          open={!!pendingDeleteVar}
          onOpenChange={(next) => {
            if (!next && !busy) setPendingDeleteVar(null)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete “{pendingDeleteVar?.key}”?</AlertDialogTitle>
              <AlertDialogDescription>
                {pendingDeleteVar && pendingDeleteVar.used_by.length > 0
                  ? `It is used by ${pendingDeleteVar.used_by.length} app config set${pendingDeleteVar.used_by.length === 1 ? "" : "s"} (${usageLines(pendingDeleteVar).join(", ")}) and is removed from all of them.`
                  : "No app uses it yet."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={busy}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                disabled={busy}
                onClick={(e) => {
                  e.preventDefault()
                  void handleDeleteVar()
                }}
              >
                Delete
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        <AlertDialog
          open={pendingDeleteSet}
          onOpenChange={(next) => {
            if (!next && !busy) setPendingDeleteSet(false)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete “{selected?.name}”?</AlertDialogTitle>
              <AlertDialogDescription>
                {usedCount > 0
                  ? `${usedCount} app config set${usedCount === 1 ? " uses" : "s use"} it. Their shared variables are removed and they stop sharing.`
                  : "Its variables are deleted. No app config set uses it."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={busy}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                disabled={busy}
                onClick={(e) => {
                  e.preventDefault()
                  void handleDeleteSet()
                }}
              >
                Delete set
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </DialogContent>
    </Dialog>
  )
}
