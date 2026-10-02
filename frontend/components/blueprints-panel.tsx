import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import {
  ArrowDownIcon,
  ArrowUpIcon,
  LayoutTemplateIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import type { Blueprint, BlueprintInput } from "@/lib/types"
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
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type BlueprintsPanelProps = {
  active: boolean
}

type CommandRow = { key: number; label: string; command: string }

type Draft = {
  id: number | null
  name: string
  description: string
  createFolder: boolean
  commands: CommandRow[]
}

export function BlueprintsPanel({ active }: BlueprintsPanelProps) {
  const [blueprints, setBlueprints] = useState<Blueprint[]>([])
  const [loaded, setLoaded] = useState(false)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState<Blueprint | null>(null)
  const nextKey = useRef(0)

  function newRow(label = "", command = ""): CommandRow {
    return { key: nextKey.current++, label, command }
  }

  // Reload each time the panel is shown.
  useEffect(() => {
    if (!active) return
    let cancelled = false
    api.blueprints
      .list()
      .then((list) => {
        if (cancelled) return
        setBlueprints(list)
        setLoaded(true)
      })
      .catch((err) => {
        if (cancelled) return
        setLoaded(true)
        toast.error(
          err instanceof Error ? err.message : "Failed to load blueprints"
        )
      })
    return () => {
      cancelled = true
    }
  }, [active])

  function startEdit(blueprint: Blueprint | null) {
    setDraft({
      id: blueprint?.id ?? null,
      name: blueprint?.name ?? "",
      description: blueprint?.description ?? "",
      createFolder: blueprint?.create_folder ?? true,
      commands: blueprint?.commands.length
        ? blueprint.commands.map((c) => newRow(c.label ?? "", c.command))
        : [newRow()],
    })
  }

  function patch(next: Partial<Draft>) {
    setDraft((prev) => (prev ? { ...prev, ...next } : prev))
  }

  function patchRow(key: number, next: Partial<CommandRow>) {
    setDraft((prev) =>
      prev
        ? {
            ...prev,
            commands: prev.commands.map((c) =>
              c.key === key ? { ...c, ...next } : c
            ),
          }
        : prev
    )
  }

  function moveRow(index: number, delta: number) {
    setDraft((prev) => {
      if (!prev) return prev
      const target = index + delta
      if (target < 0 || target >= prev.commands.length) return prev
      const commands = [...prev.commands]
      ;[commands[index], commands[target]] = [
        commands[target]!,
        commands[index]!,
      ]
      return { ...prev, commands }
    })
  }

  async function handleSave() {
    if (!draft) return
    if (!draft.name.trim()) {
      toast.error("Name is required")
      return
    }
    const body: BlueprintInput = {
      name: draft.name,
      description: draft.description,
      create_folder: draft.createFolder,
      commands: draft.commands.map((c) => ({
        label: c.label.trim() || null,
        command: c.command,
      })),
    }
    setSaving(true)
    try {
      const saved =
        draft.id == null
          ? await api.blueprints.create(body)
          : await api.blueprints.update(draft.id, body)
      setBlueprints((prev) =>
        (draft.id == null
          ? [...prev, saved]
          : prev.map((b) => (b.id === saved.id ? saved : b))
        ).sort((a, b) => a.name.localeCompare(b.name))
      )
      setDraft(null)
      toast.success("Blueprint saved")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!deleting) return
    try {
      await api.blueprints.delete(deleting.id)
      setBlueprints((prev) => prev.filter((b) => b.id !== deleting.id))
      toast.success("Blueprint deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-muted-foreground">
          Reusable recipes for creating a new app in any workspace.
        </p>
        {draft ? null : (
          <Button onClick={() => startEdit(null)}>
            <PlusIcon data-icon="inline-start" />
            New blueprint
          </Button>
        )}
      </div>

      {draft ? (
        <>
          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="blueprint-name">Name</FieldLabel>
                <Input
                  id="blueprint-name"
                  value={draft.name}
                  onChange={(e) => patch({ name: e.target.value })}
                  placeholder="Vite + React"
                  autoFocus
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="blueprint-description">
                  Description
                </FieldLabel>
                <Input
                  id="blueprint-description"
                  value={draft.description}
                  onChange={(e) => patch({ description: e.target.value })}
                  placeholder="Optional"
                />
              </Field>
              <Field orientation="horizontal" className="items-start gap-2">
                <Checkbox
                  id="blueprint-create-folder"
                  checked={draft.createFolder}
                  onCheckedChange={(checked) =>
                    patch({ createFolder: checked === true })
                  }
                />
                <div className="flex flex-col gap-1">
                  <FieldLabel htmlFor="blueprint-create-folder">
                    Create a folder for the app
                  </FieldLabel>
                  <FieldDescription>
                    On: the folder is created inside the parent folder and the
                    commands run inside it. Off: the commands run in the parent
                    folder and must create the app folder themselves, e.g.{" "}
                    <code>bun create vite {"{{folder_name}}"}</code>. You can
                    change this each time you use the blueprint.
                  </FieldDescription>
                </div>
              </Field>
              <Field>
                <FieldLabel>Commands</FieldLabel>
                <FieldDescription>
                  Run in order after the folder is ready; the first failure
                  stops the run. Then <code>git init</code> runs if the folder
                  isn’t a repository. Available: <code>{"{{app_name}}"}</code>,{" "}
                  <code>{"{{folder_name}}"}</code>, <code>{"{{app_dir}}"}</code>
                  . Values are inserted as-is, so quote them if they may contain
                  spaces. Commands must not wait for input.
                </FieldDescription>
                <ul className="flex flex-col gap-2">
                  {draft.commands.map((row, index) => (
                    <li key={row.key} className="flex items-center gap-1.5">
                      <Input
                        value={row.label}
                        onChange={(e) =>
                          patchRow(row.key, { label: e.target.value })
                        }
                        placeholder="Label"
                        aria-label="Command label"
                        className="w-32 shrink-0"
                      />
                      <Input
                        value={row.command}
                        onChange={(e) =>
                          patchRow(row.key, { command: e.target.value })
                        }
                        placeholder="bun install"
                        aria-label="Command"
                        className="min-w-0 flex-1 font-mono text-xs"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label="Move up"
                        disabled={index === 0}
                        onClick={() => moveRow(index, -1)}
                      >
                        <ArrowUpIcon />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label="Move down"
                        disabled={index === draft.commands.length - 1}
                        onClick={() => moveRow(index, 1)}
                      >
                        <ArrowDownIcon />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label="Remove command"
                        onClick={() =>
                          patch({
                            commands: draft.commands.filter(
                              (c) => c.key !== row.key
                            ),
                          })
                        }
                      >
                        <XIcon />
                      </Button>
                    </li>
                  ))}
                </ul>
                <div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      patch({ commands: [...draft.commands, newRow()] })
                    }
                  >
                    <PlusIcon data-icon="inline-start" />
                    Add command
                  </Button>
                </div>
              </Field>
            </FieldGroup>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDraft(null)}>
              Back
            </Button>
            <Button disabled={saving} onClick={() => void handleSave()}>
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          {!loaded ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : blueprints.length === 0 ? (
            <Empty className="border">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <LayoutTemplateIcon />
                </EmptyMedia>
                <EmptyTitle>No blueprints</EmptyTitle>
                <EmptyDescription>
                  A blueprint creates a new app: it runs your shell commands,
                  then runs git init.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <ul className="flex flex-col gap-2">
              {blueprints.map((blueprint) => (
                <li
                  key={blueprint.id}
                  className="flex items-center gap-2 rounded-lg border px-3 py-2"
                >
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-medium">
                        {blueprint.name}
                      </span>
                      <Badge variant="outline">
                        {blueprint.commands.length} command
                        {blueprint.commands.length === 1 ? "" : "s"}
                      </Badge>
                    </div>
                    {blueprint.description ? (
                      <span className="truncate text-xs text-muted-foreground">
                        {blueprint.description}
                      </span>
                    ) : null}
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Edit ${blueprint.name}`}
                    onClick={() => startEdit(blueprint)}
                  >
                    <PencilIcon />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Delete ${blueprint.name}`}
                    onClick={() => setDeleting(blueprint)}
                  >
                    <Trash2Icon />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <AlertDialog
        open={!!deleting}
        onOpenChange={(next) => {
          if (!next) setDeleting(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete blueprint?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes “{deleting?.name}”. Apps already created from it are
              not affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => void handleDelete()}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
