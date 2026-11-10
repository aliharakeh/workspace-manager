import { ConfigSetCompareDialog } from "@/components/config-set-compare-dialog"
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
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api } from "@/lib/api"
import type { App, ConfigSet, ConfigSetDetail } from "@/lib/types"
import {
  CheckIcon,
  ChevronsUpDownIcon,
  CopyIcon,
  GitCompareIcon,
  Link2Icon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import {
  blankSel,
  partsEnabled,
  selToParts,
  type CopySel,
} from "./config-sets/copy-selection"
import { PartsPicker } from "./config-sets/parts-picker"
type ConfigSetSwitcherProps = {
  app: App
  onAppChange: (app: App) => void
  runningConfigSetId?: number
}

export function ConfigSetSwitcher({
  app,
  onAppChange,
  runningConfigSetId,
}: ConfigSetSwitcherProps) {
  const [sets, setSets] = useState<ConfigSet[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [copyOpen, setCopyOpen] = useState(false)
  const [compareOpen, setCompareOpen] = useState(false)
  const [renameOpen, setRenameOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [name, setName] = useState("")
  const [copyFromId, setCopyFromId] = useState<number | "">("")
  const [copySourceId, setCopySourceId] = useState<number | "">("")
  const [sel, setSel] = useState<CopySel>(blankSel)
  const [sourceDetail, setSourceDetail] = useState<ConfigSetDetail | null>(null)
  const [sourceLoading, setSourceLoading] = useState(false)
  const [sourceError, setSourceError] = useState<string | null>(null)
  const sourceReqIdRef = useRef<number | null>(null)
  const [saving, setSaving] = useState(false)

  function resetSource() {
    sourceReqIdRef.current = null
    setSourceDetail(null)
    setSourceLoading(false)
    setSourceError(null)
  }

  function loadSourceDetail(id: number) {
    sourceReqIdRef.current = id
    setSourceLoading(true)
    setSourceError(null)
    api.configSets
      .getDetail(id)
      .then((detail) => {
        if (sourceReqIdRef.current === id) setSourceDetail(detail)
      })
      .catch((err) => {
        if (sourceReqIdRef.current === id) {
          setSourceDetail(null)
          setSourceError(
            err instanceof Error ? err.message : "Failed to load source set"
          )
        }
      })
      .finally(() => {
        if (sourceReqIdRef.current === id) setSourceLoading(false)
      })
  }

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
    if (copyFromId !== "" && !partsEnabled(selToParts(sel))) {
      toast.error("Select at least one item to copy")
      return
    }
    setSaving(true)
    try {
      const created = await api.configSets.create(app.id, {
        name: name.trim(),
        copy_from_id: copyFromId === "" ? undefined : copyFromId,
        activate: true,
        parts: copyFromId === "" ? undefined : selToParts(sel),
      })
      const refreshed = await api.apps.get(app.id)
      setSets(await api.configSets.list(app.id))
      onAppChange(refreshed)
      setCreateOpen(false)
      setName("")
      setCopyFromId("")
      setSel(blankSel())
      resetSource()
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
    const parts = selToParts(sel)
    if (!partsEnabled(parts)) {
      toast.error("Select at least one item to copy")
      return
    }
    setSaving(true)
    try {
      await api.configSets.copyFrom(active.id, copySourceId, parts)
      setCopyOpen(false)
      setCopySourceId("")
      setSel(blankSel())
      resetSource()
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
    return <p className="text-xs text-muted-foreground">Loading config sets…</p>
  }

  return (
    <>
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">Config set</span>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="outline" size="sm" className="min-w-40" />
              }
            >
              <span className="truncate">{active?.name ?? "—"}</span>
              {active?.shared_set_id != null ? (
                <Link2Icon
                  className="size-3.5 shrink-0 text-muted-foreground"
                  aria-label={`Shares variables from `}
                />
              ) : null}
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
                    {set.shared_set_id != null ? (
                      <Link2Icon
                        className="size-3.5 shrink-0 text-muted-foreground"
                        aria-label={`Shares variables from `}
                      />
                    ) : null}
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
                    setSel(blankSel())
                    resetSource()
                    if (active) loadSourceDetail(active.id)
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
                    const firstId =
                      sets.find((s) => s.id !== active?.id)?.id ?? ""
                    setCopySourceId(firstId)
                    setSel(blankSel())
                    resetSource()
                    if (firstId !== "") loadSourceDetail(firstId)
                    setCopyOpen(true)
                  }}
                >
                  <CopyIcon />
                  Copy into current…
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!active || sets.length < 2}
                  onClick={() => setCompareOpen(true)}
                >
                  <GitCompareIcon />
                  Compare with…
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
        {runningConfigSetId != null ? (
          <p className="text-xs text-muted-foreground" role="status">
            Running with {sets.find((set) => set.id === runningConfigSetId)?.name ??
              `config set ${runningConfigSetId}`}.
            {runningConfigSetId !== app.active_config_set_id
              ? " The active set applies to the next start or reload."
              : ""}
          </p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          The active set supplies env vars, templates, and Setup, Dev, Build and
          Prod commands. Switching sets applies to the next start or reload.
        </p>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-h-[85dvh] min-w-0 overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>New config set</DialogTitle>
            <DialogDescription>
              Each set has its own env vars, templates, and setup, dev, build
              and prod configs. Optionally copy selected parts from an existing
              set.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup className="min-w-0">
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
              <Select
                items={[
                  { value: null, label: "Empty (start fresh)" },
                  ...sets.map((set) => ({
                    value: String(set.id),
                    label: set.name,
                  })),
                ]}
                value={copyFromId === "" ? null : String(copyFromId)}
                onValueChange={(value) => {
                  const next = value === null ? "" : Number(value)
                  setCopyFromId(next)
                  setSel(blankSel())
                  resetSource()
                  if (next !== "") loadSourceDetail(next)
                }}
              >
                <SelectTrigger id="config-set-copy" className="w-full min-w-0">
                  <SelectValue
                    className="min-w-0 truncate"
                    placeholder="Empty (start fresh)"
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={null}>Empty (start fresh)</SelectItem>
                  </SelectGroup>
                  <SelectSeparator />
                  <SelectGroup>
                    <SelectLabel>Config sets</SelectLabel>
                    {sets.map((set) => (
                      <SelectItem key={set.id} value={String(set.id)}>
                        {set.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            {copyFromId !== "" ? (
              <PartsPicker
                detail={sourceDetail}
                loading={sourceLoading}
                error={sourceError}
                sel={sel}
                onChange={setSel}
              />
            ) : null}
          </FieldGroup>
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
        <DialogContent className="max-h-[85dvh] min-w-0 overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle className="min-w-0 wrap-anywhere">
              Copy into “{active?.name}”
            </DialogTitle>
            <DialogDescription>
              Replaces the selected parts in the current set with a copy from
              another set.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup className="min-w-0">
            <Field>
              <FieldLabel htmlFor="copy-source">Source set</FieldLabel>
              <Select
                items={sets.map((set) => ({
                  value: String(set.id),
                  label: set.name,
                }))}
                value={copySourceId === "" ? null : String(copySourceId)}
                onValueChange={(value) => {
                  const next = value === null ? "" : Number(value)
                  setCopySourceId(next)
                  setSel(blankSel())
                  resetSource()
                  if (next !== "") loadSourceDetail(next)
                }}
              >
                <SelectTrigger id="copy-source" className="w-full min-w-0">
                  <SelectValue className="min-w-0 truncate" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectLabel>Config sets</SelectLabel>
                    {sets
                      .filter((s) => s.id !== active?.id)
                      .map((set) => (
                        <SelectItem key={set.id} value={String(set.id)}>
                          {set.name}
                        </SelectItem>
                      ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <PartsPicker
              detail={sourceDetail}
              loading={sourceLoading}
              error={sourceError}
              sel={sel}
              onChange={setSel}
            />
          </FieldGroup>
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

      <ConfigSetCompareDialog
        open={compareOpen}
        onOpenChange={setCompareOpen}
        sets={sets}
        current={active}
        onCopied={() => onAppChange({ ...app })}
      />

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
              templates, and setup, dev, build and prod configs.
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
