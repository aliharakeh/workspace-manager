import { useState } from "react"
import {
  CopyIcon,
  PlusIcon,
  SquareIcon,
  TerminalIcon,
  Trash2Icon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { StatusEvent } from "@/lib/types"
import { useRunConfigs } from "@/lib/use-run-configs"
import { CommandConfigPanel } from "./command-config-panel"
import { Button } from "./ui/button"
import { Field, FieldLabel } from "./ui/field"
import { Input } from "./ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./ui/alert-dialog"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "./ui/empty"

export function RunConfigurationsPanel({
  appId,
  configSetId,
  status,
  onStatus,
}: {
  appId: number
  configSetId: number
  status: StatusEvent | null
  onStatus: (status: StatusEvent) => void
}) {
  const { configs, selected, select, loading, error } = useRunConfigs(
    appId,
    configSetId
  )
  const [creating, setCreating] = useState(false)
  const [duplicate, setDuplicate] = useState(false)
  const [name, setName] = useState("")
  const [busy, setBusy] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [pendingSelect, setPendingSelect] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const active =
    !!status?.running &&
    status.kind === selected?.kind &&
    status.configSetId === selected?.config_set_id

  async function create() {
    setBusy(true)
    try {
      const key = `custom-${crypto.randomUUID()}`
      await api.runConfig.save(appId, key, {
        name: name.trim(),
        mode: duplicate && selected ? selected.mode : "parallel",
        detect_ready: duplicate && selected ? selected.detect_ready : false,
        commands:
          duplicate && selected
            ? selected.commands.map((c) => ({
                label: c.label,
                command: c.command,
              }))
            : [],
      })
      setDirty(false)
      select(key)
      setCreating(false)
      toast.success("Run configuration created")
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : "Failed to create configuration"
      )
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!selected) return
    setBusy(true)
    try {
      await api.runConfig.delete(appId, selected.kind)
      setDirty(false)
      setDeleting(false)
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : "Failed to delete configuration"
      )
    } finally {
      setBusy(false)
    }
  }

  if (loading)
    return (
      <p className="text-sm text-muted-foreground">Loading configurations…</p>
    )
  if (error)
    return (
      <p role="alert" className="text-sm text-destructive">
        {error}
      </p>
    )
  return (
    <div className="flex flex-col gap-5">
      {status?.running && !active ? (
        <div className="flex items-center gap-3">
          <p role="status" className="text-sm">
            Running {status.configName || status.kind}
          </p>
          <Button
            variant="stop"
            disabled={busy}
            onClick={async () => {
              setBusy(true)
              try {
                onStatus(await api.runner.stop(appId))
              } catch (e) {
                toast.error(e instanceof Error ? e.message : "Failed to stop")
              } finally {
                setBusy(false)
              }
            }}
          >
            <SquareIcon data-icon="inline-start" className="fill-current" />
            Stop
          </Button>
        </div>
      ) : null}
      <div className="flex flex-wrap items-end gap-2">
        {selected ? (
          <Field className="max-w-sm flex-1">
            <FieldLabel htmlFor="run-config-selector">
              Run config
            </FieldLabel>
            <Select
              value={selected.kind}
              items={configs.map((c) => ({ value: c.kind, label: c.name }))}
              onValueChange={(key) => {
                if (!key || key === selected.kind) return
                if (dirty) setPendingSelect(key)
                else select(key)
              }}
            >
              <SelectTrigger id="run-config-selector" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {configs.map((c) => (
                    <SelectItem key={c.kind} value={c.kind}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
        ) : null}
        <Button
          variant="outline"
          disabled={dirty}
          onClick={() => {
            setName("")
            setDuplicate(false)
            setCreating(true)
          }}
        >
          <PlusIcon data-icon="inline-start" />
          Add configuration
        </Button>
        {selected ? (
          <>
            <Button
              variant="outline"
              disabled={dirty}
              onClick={() => {
                setName(`${selected.name} copy`)
                setDuplicate(true)
                setCreating(true)
              }}
            >
              <CopyIcon data-icon="inline-start" />
              Duplicate
            </Button>
            <Button
              variant="ghost"
              size="icon"
              disabled={active || busy}
              aria-label="Delete configuration"
              onClick={() => setDeleting(true)}
            >
              <Trash2Icon />
            </Button>
          </>
        ) : null}
      </div>
      {selected ? (
        <CommandConfigPanel
          key={`${appId}:${configSetId}:${selected.kind}`}
          appId={appId}
          kind={selected.kind}
          running={active}
          anyRunning={!!status?.running}
          onDirtyChange={setDirty}
          onStatus={onStatus}
        />
      ) : (
        <Empty className="border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <TerminalIcon />
            </EmptyMedia>
            <EmptyTitle>No run configurations yet</EmptyTitle>
            <EmptyDescription>
              Add a configuration with your own name and commands. Run its
              commands together or one after another.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      <Dialog open={creating} onOpenChange={setCreating}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {duplicate ? "Duplicate configuration" : "Add run configuration"}
            </DialogTitle>
            <DialogDescription>
              Choose a name such as Development, Tests, or Install dependencies.
            </DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              void create()
            }}
            className="flex flex-col gap-4"
          >
            <Field>
              <FieldLabel htmlFor="new-run-name">Name</FieldLabel>
              <Input
                id="new-run-name"
                name="name"
                autoFocus
                required
                maxLength={128}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </Field>
            <Button type="submit" disabled={busy}>
              {busy ? "Creating…" : "Create configuration"}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
      <AlertDialog
        open={pendingSelect !== null}
        onOpenChange={(open) => {
          if (!open) setPendingSelect(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>
              Save your changes before switching to keep them.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (pendingSelect) select(pendingSelect)
                setDirty(false)
                setPendingSelect(null)
              }}
            >
              Discard changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={deleting} onOpenChange={setDeleting}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete “{selected?.name}”?</AlertDialogTitle>
            <AlertDialogDescription>
              This permanently deletes this configuration and its commands.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={busy}
              onClick={() => void remove()}
            >
              Delete configuration
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
