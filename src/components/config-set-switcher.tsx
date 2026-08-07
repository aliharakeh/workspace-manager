import { useEffect, useState } from "react"
import { toast } from "sonner"
import {
  CheckIcon,
  ChevronsUpDownIcon,
  CopyIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"
import { api } from "@/lib/api"
import type { App, ConfigSet } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type CopyParts = { env: boolean; templates: boolean; run: boolean }

const ALL_PARTS: CopyParts = { env: true, templates: true, run: true }

type ConfigSetSwitcherProps = {
  app: App
  onAppChange: (app: App) => void
}

function PartsPicker({
  parts,
  onChange,
  disabled,
}: {
  parts: CopyParts
  onChange: (parts: CopyParts) => void
  disabled?: boolean
}) {
  return (
    <div className="flex flex-col gap-2">
      <FieldLabel>Copy</FieldLabel>
      <div className="flex flex-col gap-1.5 text-sm">
        {(
          [
            ["env", "Env vars"],
            ["templates", "Templates"],
            ["run", "Run config"],
          ] as const
        ).map(([key, label]) => (
          <label key={key} className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={parts[key]}
              disabled={disabled}
              onChange={(e) =>
                onChange({ ...parts, [key]: e.target.checked })
              }
            />
            {label}
          </label>
        ))}
      </div>
    </div>
  )
}

export function ConfigSetSwitcher({ app, onAppChange }: ConfigSetSwitcherProps) {
  const [sets, setSets] = useState<ConfigSet[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [copyOpen, setCopyOpen] = useState(false)
  const [renameOpen, setRenameOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [name, setName] = useState("")
  const [copyFromId, setCopyFromId] = useState<number | "">("")
  const [copySourceId, setCopySourceId] = useState<number | "">("")
  const [createParts, setCreateParts] = useState<CopyParts>(ALL_PARTS)
  const [copyParts, setCopyParts] = useState<CopyParts>(ALL_PARTS)
  const [saving, setSaving] = useState(false)

  const active =
    sets.find((s) => s.id === app.active_config_set_id) ?? sets[0] ?? null

  async function load() {
    setLoading(true)
    try {
      setSets(await api.configSets.list(app.id))
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load config sets"
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [app.id])

  async function activate(id: number) {
    try {
      const result = await api.configSets.activate(id)
      onAppChange(result.app)
      toast.success("Config set activated")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to activate")
    }
  }

  async function handleCreate() {
    if (!name.trim()) {
      toast.error("Name is required")
      return
    }
    if (
      copyFromId !== "" &&
      !createParts.env &&
      !createParts.templates &&
      !createParts.run
    ) {
      toast.error("Select at least one part to copy")
      return
    }
    setSaving(true)
    try {
      const created = await api.configSets.create(app.id, {
        name: name.trim(),
        copy_from_id: copyFromId === "" ? undefined : copyFromId,
        activate: true,
        parts: copyFromId === "" ? undefined : createParts,
      })
      const refreshed = await api.apps.get(app.id)
      setSets(await api.configSets.list(app.id))
      onAppChange(refreshed)
      setCreateOpen(false)
      setName("")
      setCopyFromId("")
      setCreateParts(ALL_PARTS)
      toast.success(
        copyFromId === ""
          ? `Created “${created.name}”`
          : `Created “${created.name}” (copied)`
      )
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create")
    } finally {
      setSaving(false)
    }
  }

  async function handleRename() {
    if (!active || !name.trim()) {
      toast.error("Name is required")
      return
    }
    setSaving(true)
    try {
      const updated = await api.configSets.update(active.id, {
        name: name.trim(),
      })
      setSets(await api.configSets.list(app.id))
      setRenameOpen(false)
      toast.success(`Renamed to “${updated.name}”`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to rename")
    } finally {
      setSaving(false)
    }
  }

  async function handleCopyInto() {
    if (!active || copySourceId === "") {
      toast.error("Pick a source config set")
      return
    }
    if (!copyParts.env && !copyParts.templates && !copyParts.run) {
      toast.error("Select at least one part to copy")
      return
    }
    setSaving(true)
    try {
      await api.configSets.copyFrom(active.id, copySourceId, copyParts)
      setCopyOpen(false)
      setCopySourceId("")
      setCopyParts(ALL_PARTS)
      toast.success(`Copied into “${active.name}”`)
      onAppChange({ ...app })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to copy")
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (deleteId == null) return
    setSaving(true)
    try {
      await api.configSets.delete(deleteId)
      const refreshed = await api.apps.get(app.id)
      setSets(await api.configSets.list(app.id))
      onAppChange(refreshed)
      setDeleteId(null)
      toast.success("Config set deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setSaving(false)
    }
  }

  if (loading && !active) {
    return (
      <p className="text-xs text-muted-foreground">Loading config sets…</p>
    )
  }

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-muted-foreground">Config set</span>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="outline" size="sm" className="min-w-40" />
            }
          >
            <span className="truncate">{active?.name ?? "—"}</span>
            <ChevronsUpDownIcon data-icon="inline-end" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="min-w-48">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Switch set</DropdownMenuLabel>
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
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem
                onClick={() => {
                  setName("")
                  setCopyFromId(active?.id ?? "")
                  setCreateParts(ALL_PARTS)
                  setCreateOpen(true)
                }}
              >
                <PlusIcon />
                New set…
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!active}
                onClick={() => {
                  setName(active?.name ?? "")
                  setRenameOpen(true)
                }}
              >
                <PencilIcon />
                Rename…
              </DropdownMenuItem>
              <DropdownMenuItem
                disabled={!active || sets.length < 2}
                onClick={() => {
                  setCopySourceId(
                    sets.find((s) => s.id !== active?.id)?.id ?? ""
                  )
                  setCopyParts(ALL_PARTS)
                  setCopyOpen(true)
                }}
              >
                <CopyIcon />
                Copy into current…
              </DropdownMenuItem>
              <DropdownMenuItem
                variant="destructive"
                disabled={!active || sets.length <= 1}
                onClick={() => active && setDeleteId(active.id)}
              >
                <Trash2Icon />
                Delete current
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New config set</DialogTitle>
            <DialogDescription>
              Each set has its own env vars, templates, and run config. Optionally
              copy selected parts from an existing set.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <Field>
              <FieldLabel htmlFor="config-set-name">Name</FieldLabel>
              <Input
                id="config-set-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Staging"
                autoFocus
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="config-set-copy">Copy from</FieldLabel>
              <select
                id="config-set-copy"
                className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 text-sm outline-none"
                value={copyFromId === "" ? "" : String(copyFromId)}
                onChange={(e) =>
                  setCopyFromId(
                    e.target.value === "" ? "" : Number(e.target.value)
                  )
                }
              >
                <option value="">Empty (start fresh)</option>
                {sets.map((set) => (
                  <option key={set.id} value={set.id}>
                    {set.name}
                  </option>
                ))}
              </select>
            </Field>
            {copyFromId !== "" ? (
              <PartsPicker parts={createParts} onChange={setCreateParts} />
            ) : null}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button disabled={saving} onClick={() => void handleCreate()}>
              {saving ? "Creating…" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename config set</DialogTitle>
            <DialogDescription>
              Change the display name of “{active?.name}”.
            </DialogDescription>
          </DialogHeader>
          <Field>
            <FieldLabel htmlFor="config-set-rename">Name</FieldLabel>
            <Input
              id="config-set-rename"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleRename()
              }}
            />
          </Field>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameOpen(false)}>
              Cancel
            </Button>
            <Button disabled={saving} onClick={() => void handleRename()}>
              {saving ? "Saving…" : "Rename"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={copyOpen} onOpenChange={setCopyOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Copy into “{active?.name}”</DialogTitle>
            <DialogDescription>
              Replaces the selected parts in the current set with a copy from
              another set.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <Field>
              <FieldLabel htmlFor="copy-source">Source set</FieldLabel>
              <select
                id="copy-source"
                className="flex h-8 w-full rounded-lg border border-input bg-transparent px-2.5 text-sm outline-none"
                value={copySourceId === "" ? "" : String(copySourceId)}
                onChange={(e) =>
                  setCopySourceId(
                    e.target.value === "" ? "" : Number(e.target.value)
                  )
                }
              >
                {sets
                  .filter((s) => s.id !== active?.id)
                  .map((set) => (
                    <option key={set.id} value={set.id}>
                      {set.name}
                    </option>
                  ))}
              </select>
            </Field>
            <PartsPicker parts={copyParts} onChange={setCopyParts} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCopyOpen(false)}>
              Cancel
            </Button>
            <Button disabled={saving} onClick={() => void handleCopyInto()}>
              {saving ? "Copying…" : "Replace with copy"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={deleteId != null}
        onOpenChange={(open) => {
          if (!open) setDeleteId(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete config set?</AlertDialogTitle>
            <AlertDialogDescription>
              This permanently deletes “{active?.name}” and all of its env vars,
              templates, and run config.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={saving}
              onClick={() => void handleDelete()}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
