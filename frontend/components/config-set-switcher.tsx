import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import {
  CheckIcon,
  ChevronsUpDownIcon,
  CopyIcon,
  PencilIcon,
  PlusIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import type { App, ConfigSet, ConfigSetDetail, CopyParts } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

type CategorySel = { all: boolean; items: string[] }

type CopySel = {
  env: CategorySel
  templates: CategorySel
  run: CategorySel
}

function blankCategory(): CategorySel {
  return { all: true, items: [] }
}

function blankSel(): CopySel {
  return {
    env: blankCategory(),
    templates: blankCategory(),
    run: blankCategory(),
  }
}

function categoryToPart(sel: CategorySel): boolean | string[] {
  if (sel.all) return true
  return sel.items.length > 0 ? sel.items : false
}

function runToPart(sel: CategorySel): boolean | number[] {
  if (sel.all) return true
  return sel.items.length > 0 ? sel.items.map(Number) : false
}

function selToParts(sel: CopySel): CopyParts {
  return {
    env: categoryToPart(sel.env),
    templates: categoryToPart(sel.templates),
    run: runToPart(sel.run),
  }
}

function isPartEnabled(part: boolean | string[] | number[] | undefined) {
  if (part === undefined || part === true) return true
  return Array.isArray(part) && part.length > 0
}

function partsEnabled(parts: CopyParts): boolean {
  return (
    isPartEnabled(parts.env) ||
    isPartEnabled(parts.templates) ||
    isPartEnabled(parts.run)
  )
}

type ConfigSetSwitcherProps = {
  app: App
  onAppChange: (app: App) => void
}

type CategoryItem = { id: string; title: string; sub?: string }

function CategoryPicker({
  items,
  sel,
  onChange,
  query,
  onQueryChange,
}: {
  items: CategoryItem[]
  sel: CategorySel
  onChange: (sel: CategorySel) => void
  query: string
  onQueryChange: (query: string) => void
}) {
  const count = items.length
  const allChecked = count > 0 && (sel.all || sel.items.length === count)
  const q = query.trim().toLowerCase()
  const visible = q
    ? items.filter(
        (item) =>
          item.title.toLowerCase().includes(q) ||
          (item.sub ?? "").toLowerCase().includes(q)
      )
    : items
  return (
    <div className="flex flex-col gap-2">
      {count > 0 ? (
        <InputGroup className="w-full">
          <InputGroupAddon align="inline-start">
            <SearchIcon />
          </InputGroupAddon>
          <InputGroupInput
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Search…"
            aria-label="Search items to copy"
          />
          {query ? (
            <InputGroupAddon align="inline-end">
              <InputGroupButton
                size="icon-xs"
                aria-label="Clear search"
                onClick={() => onQueryChange("")}
              >
                <XIcon />
              </InputGroupButton>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
      ) : null}
      <Field orientation="horizontal">
        <Checkbox
          id="category-select-all"
          checked={allChecked}
          disabled={count === 0}
          onCheckedChange={() =>
            onChange(
              allChecked ? { all: false, items: [] } : { all: true, items: [] }
            )
          }
        />
        <FieldLabel htmlFor="category-select-all">Select all</FieldLabel>
        <span className="text-xs text-muted-foreground">
          {count === 0
            ? "(none)"
            : sel.all
              ? `All ${count}`
              : `${sel.items.length} of ${count}`}
        </span>
      </Field>
      {count > 0 && !sel.all ? (
        <div className="flex max-h-52 flex-col gap-1 overflow-y-auto rounded-lg border p-2 text-sm">
          {visible.length === 0 ? (
            <p className="px-1 py-2 text-xs text-muted-foreground">
              No items match “{query.trim()}”.
            </p>
          ) : (
            visible.map((item, index) => {
              const checkboxId = `category-item-${index}`
              return (
                <Field
                  key={item.id}
                  orientation="horizontal"
                  className="min-w-0"
                >
                  <Checkbox
                    id={checkboxId}
                    checked={sel.items.includes(item.id)}
                    onCheckedChange={(checked) => {
                      const next = new Set(sel.items)
                      if (checked) next.add(item.id)
                      else next.delete(item.id)
                      onChange(
                        next.size === count
                          ? { all: true, items: [] }
                          : { all: false, items: [...next] }
                      )
                    }}
                  />
                  <FieldLabel
                    htmlFor={checkboxId}
                    className="min-w-0 truncate font-normal"
                    title={item.sub ?? item.title}
                  >
                    {item.title}
                  </FieldLabel>
                </Field>
              )
            })
          )}
        </div>
      ) : null}
    </div>
  )
}

function PartsPicker({
  detail,
  loading,
  error,
  sel,
  onChange,
}: {
  detail: ConfigSetDetail | null
  loading: boolean
  error: string | null
  sel: CopySel
  onChange: (sel: CopySel) => void
}) {
  const [tab, setTab] = useState("env")
  const [query, setQuery] = useState("")

  if (loading) {
    return <p className="text-xs text-muted-foreground">Loading source set…</p>
  }
  if (error) {
    return <p className="text-xs text-destructive">{error}</p>
  }
  if (!detail) return null
  return (
    <Tabs
      value={tab}
      onValueChange={(value) => {
        if (value) {
          setTab(value)
          setQuery("")
        }
      }}
    >
      <TabsList className="w-full">
        <TabsTrigger value="env" className="flex-1">
          Env vars
        </TabsTrigger>
        <TabsTrigger value="templates" className="flex-1">
          Templates
        </TabsTrigger>
        <TabsTrigger value="run" className="flex-1">
          Run config
        </TabsTrigger>
      </TabsList>
      <TabsContent value="env" className="mt-3">
        <CategoryPicker
          items={detail.env_vars.map((v) => ({
            id: v.key,
            title: v.key,
            sub: v.value,
          }))}
          sel={sel.env}
          onChange={(env) => onChange({ ...sel, env })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="templates" className="mt-3">
        <CategoryPicker
          items={detail.templates.map((t) => ({
            id: t.file_path,
            title: t.file_path,
          }))}
          sel={sel.templates}
          onChange={(templates) => onChange({ ...sel, templates })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
      <TabsContent value="run" className="mt-3">
        <CategoryPicker
          items={(detail.run_config?.commands ?? []).map((c) => ({
            id: String(c.id),
            title: c.label ?? c.command,
            sub: c.label ? c.command : undefined,
          }))}
          sel={sel.run}
          onChange={(run) => onChange({ ...sel, run })}
          query={query}
          onQueryChange={setQuery}
        />
      </TabsContent>
    </Tabs>
  )
}

export function ConfigSetSwitcher({
  app,
  onAppChange,
}: ConfigSetSwitcherProps) {
  const [sets, setSets] = useState<ConfigSet[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [copyOpen, setCopyOpen] = useState(false)
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
        <p className="text-xs text-muted-foreground">
          Each set bundles its own env vars, templates, and run commands — use
          one per environment (e.g. dev, staging, prod). Switch the active set,
          or copy selected items between sets.
        </p>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New config set</DialogTitle>
            <DialogDescription>
              Each set has its own env vars, templates, and run config.
              Optionally copy selected parts from an existing set.
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
                <SelectTrigger id="config-set-copy" className="w-full">
                  <SelectValue placeholder="Empty (start fresh)" />
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
                <SelectTrigger id="copy-source" className="w-full">
                  <SelectValue />
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
