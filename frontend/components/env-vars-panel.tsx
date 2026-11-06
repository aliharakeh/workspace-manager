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
import { Badge } from "@/components/ui/badge"
import { LinkSharedDialog } from "@/components/link-shared-dialog"
import { api } from "@/lib/api"
import type { ConfigSet, EnvVar } from "@/lib/types"
import {
  FileUpIcon,
  KeyRoundIcon,
  Link2Icon,
  Link2OffIcon,
  ListChecksIcon,
  LockIcon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react"
import { SecretInput } from "@/components/secret-input"
import { toast } from "sonner"

type EnvVarsPanelProps = {
  appId: number
  workspaceId: number
  /** The config set whose variables are shown (null: none is active). */
  activeConfigSetId: number | null
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

/** A variable taken from the workspace's shared config: shown, never edited here. */
function SharedEnvVarRow({ item }: { item: EnvVar }) {
  const isChoice = item.options.length > 0
  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-muted/30 p-3 sm:flex-row sm:items-center">
      <div className="flex items-center gap-2 sm:w-48">
        <LockIcon
          className="size-3.5 shrink-0 text-muted-foreground"
          aria-label="Shared from the workspace"
        />
        <Input
          className="font-mono"
          value={item.key}
          readOnly
          aria-label="Variable key"
        />
      </div>
      <SecretInput
        secret={item.private}
        className="flex-1 font-mono"
        value={item.value}
        readOnly
        aria-label="Variable value"
        title={
          isChoice ? `Option “${item.selected}” of the shared variable` : undefined
        }
      />
      <div className="flex items-center gap-2">
        {isChoice ? <Badge variant="outline">{item.selected}</Badge> : null}
        {item.private ? (
          <Badge variant="outline" title="Never shown to the AI or to agents">
            private
          </Badge>
        ) : null}
      </div>
    </div>
  )
}

export function EnvVarsPanel({
  appId,
  workspaceId,
  activeConfigSetId,
  dataEpoch,
}: EnvVarsPanelProps) {
  const [vars, setVars] = useState<EnvVar[]>([])
  const [configSet, setConfigSet] = useState<ConfigSet | null>(null)
  const [linkOpen, setLinkOpen] = useState(false)
  const [unlinkOpen, setUnlinkOpen] = useState(false)
  const [unlinking, setUnlinking] = useState(false)
  const [key, setKey] = useState("")
  const [value, setValue] = useState("")
  const [query, setQuery] = useState("")
  const deferredQuery = useDeferredValue(query)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [importing, setImporting] = useState(false)
  const [isPrivate, setIsPrivate] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<EnvVar | null>(null)
  const [deleting, setDeleting] = useState(false)

  // Rows mirroring a workspace shared variable are read-only and shown apart
  // from the app's own variables.
  const sharedVars = useMemo(
    () => vars.filter((v) => v.shared_var_id != null),
    [vars]
  )
  const ownVars = useMemo(
    () => vars.filter((v) => v.shared_var_id == null),
    [vars]
  )
  const linked = configSet?.shared_set_id != null

  const matchesQuery = useMemo(() => {
    const q = deferredQuery.trim().toLowerCase()
    return (v: EnvVar) =>
      !q ||
      v.key.toLowerCase().includes(q) ||
      v.value.toLowerCase().includes(q) ||
      v.options.some(
        (o) =>
          o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q)
      )
  }, [deferredQuery])
  const filtered = useMemo(
    () => ownVars.filter(matchesQuery),
    [ownVars, matchesQuery]
  )
  const filteredShared = useMemo(
    () => sharedVars.filter(matchesQuery),
    [sharedVars, matchesQuery]
  )

  // A silent load swaps the rows without the loading state, which would
  // replace the list and reset the scroll position.
  const loadSeq = useRef(0)
  async function load(silent = false) {
    const seq = ++loadSeq.current
    if (!silent) setLoading(true)
    try {
      const [list, sets] = await Promise.all([
        api.envVars.list(appId),
        api.configSets.list(appId),
      ])
      if (seq === loadSeq.current) {
        setVars(list)
        setConfigSet(
          (sets as unknown as ConfigSet[]).find((s) => s.id === activeConfigSetId) ??
            null
        )
      }
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
        private: isPrivate,
      })
      setVars((prev) =>
        [...prev, created].sort((a, b) => a.key.localeCompare(b.key))
      )
      setKey("")
      setValue("")
      setIsPrivate(false)
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

  async function handleUnlink(keep: boolean) {
    if (!configSet) return
    setUnlinking(true)
    try {
      await api.sharedSets.unlink(configSet.id, keep)
      setUnlinkOpen(false)
      await load(true)
      toast.success(
        keep ? "Unlinked; the variables are now this app's own" : "Unlinked"
      )
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to unlink")
    } finally {
      setUnlinking(false)
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
            <SecretInput
              id="env-value"
              secret={isPrivate}
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
            <FieldLabel htmlFor="env-private">Private</FieldLabel>
            <div className="flex h-8 items-center">
              <Checkbox
                id="env-private"
                checked={isPrivate}
                onCheckedChange={(checked) => setIsPrivate(checked === true)}
                aria-label="Private"
                title="Private: masked here, and never shown to AI or agents"
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
            {!linked && configSet ? (
              <Button
                variant="outline"
                onClick={() => setLinkOpen(true)}
                title="Take variables from the workspace's shared config"
              >
                <Link2Icon data-icon="inline-start" />
                Link shared variables
              </Button>
            ) : null}
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
        <div className="flex flex-col gap-4">
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
              {filtered.length + filteredShared.length} of {vars.length}
            </span>
          </div>

          {linked && configSet ? (
            <section className="flex flex-col gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-sm font-medium">
                  Shared from workspace
                  <span className="text-muted-foreground"> · </span>
                  <span className="font-mono">{configSet.shared_set_name}</span>
                </h3>
                <div className="ml-auto flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setLinkOpen(true)}
                    title="Choose which variables to take, or link another set"
                  >
                    <Link2Icon data-icon="inline-start" />
                    Edit selection
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setUnlinkOpen(true)}
                  >
                    <Link2OffIcon data-icon="inline-start" />
                    Unlink
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Read-only here. Change them in the workspace&apos;s shared
                config and every config set using them follows.
              </p>
              {filteredShared.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {sharedVars.length === 0
                    ? "No shared variables taken yet."
                    : `No shared variables match “${query.trim()}”.`}
                </p>
              ) : (
                <div className="flex flex-col gap-2">
                  {filteredShared.map((item) => (
                    <SharedEnvVarRow key={item.id} item={item} />
                  ))}
                </div>
              )}
            </section>
          ) : null}

          <section className="flex flex-col gap-2">
            {linked ? (
              <h3 className="text-sm font-medium">App variables</h3>
            ) : null}
            {filtered.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {ownVars.length === 0
                  ? "No app variables."
                  : `No env vars match “${query.trim()}”.`}
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
          </section>
        </div>
      )}

      {configSet ? (
        <LinkSharedDialog
          open={linkOpen}
          onOpenChange={setLinkOpen}
          workspaceId={workspaceId}
          configSetId={configSet.id}
          configSetName={configSet.name}
          currentSetId={configSet.shared_set_id}
          linkedVarIds={sharedVars.map((v) => v.shared_var_id as number)}
          ownKeys={ownVars.map((v) => v.key)}
          onLinked={() => void load(true)}
        />
      ) : null}

      <AlertDialog
        open={unlinkOpen}
        onOpenChange={(next) => {
          if (!unlinking) setUnlinkOpen(next)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unlink shared variables?</AlertDialogTitle>
            <AlertDialogDescription>
              {sharedVars.length} variable{sharedVars.length === 1 ? "" : "s"}{" "}
              from “{configSet?.shared_set_name}” stop following the workspace.
              Keep them as this config set&apos;s own variables, or delete them.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={unlinking}>Cancel</AlertDialogCancel>
            <Button
              variant="outline"
              disabled={unlinking}
              onClick={() => void handleUnlink(false)}
            >
              Delete them
            </Button>
            <AlertDialogAction
              disabled={unlinking}
              onClick={(e) => {
                e.preventDefault()
                void handleUnlink(true)
              }}
            >
              Keep as app variables
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
