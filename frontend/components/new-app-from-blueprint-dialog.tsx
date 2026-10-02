import { useEffect, useMemo, useRef, useState } from "react"
import { flushSync } from "react-dom"
import { toast } from "sonner"
import { FolderOpenIcon, LayoutTemplateIcon } from "lucide-react"
import { api, onBlueprintLog } from "@/lib/api"
import { slugify } from "@/lib/routes"
import { decodeTerminalData } from "@/lib/terminal"
import type { App, Blueprint, Workspace } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Terminal, type TerminalHandle } from "@/components/terminal"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Empty,
  EmptyContent,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

type NewAppFromBlueprintDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspace: Workspace | null
  onCreated: (app: App) => void
  onManage: () => void
}

type Phase = "form" | "running" | "failed"

const PARENT_KEY = "blueprint-parent-path"

function joinPath(parent: string, folder: string) {
  const sep = parent.includes("\\") ? "\\" : "/"
  return `${parent.replace(/[\\/]+$/, "")}${sep}${folder}`
}

function renderCommand(command: string, vars: Record<string, string>) {
  return command.replace(
    /\{\{\s*(app_name|folder_name|app_dir)\s*\}\}/g,
    (_, key: string) => vars[key] ?? ""
  )
}

function readStoredParent() {
  try {
    return localStorage.getItem(PARENT_KEY) ?? ""
  } catch {
    return ""
  }
}

export function NewAppFromBlueprintDialog({
  open,
  onOpenChange,
  workspace,
  onCreated,
  onManage,
}: NewAppFromBlueprintDialogProps) {
  const [blueprints, setBlueprints] = useState<Blueprint[] | null>(null)
  const [blueprintId, setBlueprintId] = useState<number | null>(null)
  const [name, setName] = useState("")
  const [folderName, setFolderName] = useState("")
  const [folderTouched, setFolderTouched] = useState(false)
  const [parentPath, setParentPath] = useState("")
  const [createFolder, setCreateFolder] = useState(true)
  const [browsing, setBrowsing] = useState(false)
  const [phase, setPhase] = useState<Phase>("form")
  const [error, setError] = useState("")
  // Set while a command has failed and the run waits for skip / stop.
  const [failure, setFailure] = useState("")
  const [adding, setAdding] = useState(false)
  const runId = useRef("")
  // Output that arrives before the terminal has mounted waits here.
  const terminal = useRef<TerminalHandle | null>(null)
  const pendingOutput = useRef<Uint8Array[]>([])

  const blueprint = blueprints?.find((b) => b.id === blueprintId) ?? null
  const trimmedParent = parentPath.trim()
  const folder = folderName.trim()
  const appDir = trimmedParent && folder ? joinPath(trimmedParent, folder) : ""
  const cwd = createFolder ? appDir : trimmedParent
  const commands = useMemo(() => {
    const vars = { app_name: name.trim(), folder_name: folder, app_dir: appDir }
    return (blueprint?.commands ?? []).map((c) =>
      renderCommand(c.command, vars)
    )
  }, [blueprint, name, folder, appDir])

  // Reset when the dialog opens (state adjusted during render, not in an effect).
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setPhase("form")
      setError("")
      setName("")
      setFolderName("")
      setFolderTouched(false)
      setParentPath(readStoredParent())
      setBrowsing(false)
      setBlueprints(null)
    }
  }

  useEffect(() => {
    if (!open || !workspace) return
    let cancelled = false
    api.blueprints
      .list()
      .then((list) => {
        if (cancelled) return
        setBlueprints(list)
        setBlueprintId(list[0]?.id ?? null)
        setCreateFolder(list[0]?.create_folder ?? true)
      })
      .catch((err) => {
        if (cancelled) return
        setBlueprints([])
        toast.error(
          err instanceof Error ? err.message : "Failed to load blueprints"
        )
      })
    return () => {
      cancelled = true
    }
  }, [open, workspace])

  function handleSelectBlueprint(value: string | null) {
    const next = blueprints?.find((b) => String(b.id) === value)
    if (!next) return
    setBlueprintId(next.id)
    setCreateFolder(next.create_folder)
  }

  function handleNameChange(value: string) {
    setName(value)
    if (!folderTouched) setFolderName(value.trim() ? slugify(value) : "")
  }

  async function handleBrowse() {
    flushSync(() => setBrowsing(true))
    await new Promise((r) => window.setTimeout(r, 0))
    try {
      const picked = await api.fs.pickFolder({
        startDir: trimmedParent || undefined,
      })
      if (picked.cancelled || !picked.path) return
      setParentPath(picked.path)
      try {
        localStorage.setItem(PARENT_KEY, picked.path)
      } catch {
        // remembering the folder is optional
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to browse")
    } finally {
      setBrowsing(false)
    }
  }

  async function handleRun() {
    if (!blueprint || !workspace) return
    const id = crypto.randomUUID()
    runId.current = id
    pendingOutput.current = []
    setError("")
    setFailure("")
    setPhase("running")
    const off = onBlueprintLog((event) => {
      if (event.runId !== id) return
      if (event.stream === "failed") {
        setFailure(event.text)
      } else if (event.stream === "data") {
        const bytes = decodeTerminalData(event.data)
        if (terminal.current) terminal.current.write(bytes)
        else pendingOutput.current.push(bytes)
      }
    })
    try {
      const result = await api.blueprints.createApp({
        run_id: id,
        blueprint_id: blueprint.id,
        workspace_id: workspace.id,
        name: name.trim(),
        parent_path: trimmedParent,
        folder_name: folder,
        create_folder: createFolder,
        ask_on_error: true,
      })
      try {
        localStorage.setItem(PARENT_KEY, trimmedParent)
      } catch {
        // remembering the folder is optional
      }
      toast.success("App created")
      if (result.warning) toast.warning(result.warning)
      onCreated(result.app as App)
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create app")
      setPhase("failed")
    } finally {
      off()
    }
  }

  function handleTerminal(handle: TerminalHandle | null) {
    terminal.current = handle
    if (!handle) return
    for (const bytes of pendingOutput.current.splice(0)) handle.write(bytes)
  }

  function handleResolve(action: "skip" | "abort") {
    setFailure("")
    void api.blueprints.resolve(runId.current, action)
  }

  async function handleAddAnyway() {
    if (!workspace) return
    setAdding(true)
    try {
      const app = await api.apps.create(workspace.id, {
        name: name.trim(),
        project_path: appDir,
      })
      toast.success("App added")
      onCreated(app as App)
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add app")
    } finally {
      setAdding(false)
    }
  }

  const running = phase === "running"
  const canRun =
    !!blueprint && !!name.trim() && !!folder && !!trimmedParent && !browsing

  return (
    <Dialog
      open={open}
      modal={!browsing}
      disablePointerDismissal={browsing || running}
      onOpenChange={(next) => {
        if ((browsing || running) && !next) return
        onOpenChange(next)
      }}
    >
      <DialogContent
        className={`flex max-h-[90vh] flex-col ${phase === "form" ? "sm:max-w-xl" : "sm:max-w-3xl"}`}
      >
        <DialogHeader>
          <DialogTitle>New app from blueprint</DialogTitle>
          <DialogDescription>
            {workspace
              ? `Creates an app in ${workspace.name} by running the blueprint’s commands, then git init.`
              : null}
          </DialogDescription>
        </DialogHeader>

        {blueprints === null ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : blueprints.length === 0 ? (
          <Empty className="border">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <LayoutTemplateIcon />
              </EmptyMedia>
              <EmptyTitle>No blueprints yet</EmptyTitle>
              <EmptyDescription>
                Create a blueprint with the shell commands that scaffold your
                app.
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button onClick={onManage}>Create a blueprint</Button>
            </EmptyContent>
          </Empty>
        ) : phase === "form" ? (
          <>
            <div className="min-h-0 flex-1 overflow-y-auto pr-1">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="bp-select">Blueprint</FieldLabel>
                  <Select
                    items={blueprints.map((b) => ({
                      value: String(b.id),
                      label: b.name,
                    }))}
                    value={blueprintId == null ? null : String(blueprintId)}
                    onValueChange={handleSelectBlueprint}
                  >
                    <SelectTrigger id="bp-select" className="w-full">
                      <SelectValue placeholder="Choose a blueprint…" />
                    </SelectTrigger>
                    <SelectContent>
                      {blueprints.map((b) => (
                        <SelectItem key={b.id} value={String(b.id)}>
                          {b.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {blueprint?.description ? (
                    <FieldDescription>{blueprint.description}</FieldDescription>
                  ) : null}
                </Field>
                <Field>
                  <FieldLabel htmlFor="bp-name">App name</FieldLabel>
                  <Input
                    id="bp-name"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    placeholder="API server"
                    autoFocus
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="bp-parent">Parent folder</FieldLabel>
                  <div className="flex gap-2">
                    <Input
                      id="bp-parent"
                      value={parentPath}
                      onChange={(e) => setParentPath(e.target.value)}
                      placeholder="C:\Projects"
                      className="min-w-0 flex-1"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      disabled={browsing}
                      onClick={() => void handleBrowse()}
                    >
                      <FolderOpenIcon data-icon="inline-start" />
                      {browsing ? "Browsing…" : "Browse"}
                    </Button>
                  </div>
                </Field>
                <Field orientation="horizontal" className="items-start gap-2">
                  <Checkbox
                    id="bp-create-folder"
                    checked={createFolder}
                    onCheckedChange={(checked) =>
                      setCreateFolder(checked === true)
                    }
                  />
                  <div className="flex flex-col gap-1">
                    <FieldLabel htmlFor="bp-create-folder">
                      Create a folder for the app
                    </FieldLabel>
                    <FieldDescription>
                      Off: the commands run in the parent folder and must create
                      the app folder themselves.
                    </FieldDescription>
                  </div>
                </Field>
                <Field>
                  <FieldLabel htmlFor="bp-folder">Folder name</FieldLabel>
                  <Input
                    id="bp-folder"
                    value={folderName}
                    onChange={(e) => {
                      setFolderTouched(true)
                      setFolderName(e.target.value)
                    }}
                    placeholder="api-server"
                    className="font-mono text-xs"
                  />
                </Field>
                <div className="flex flex-col gap-1.5 rounded-lg border bg-muted/30 p-3 text-xs">
                  <div>
                    <span className="text-muted-foreground">App folder: </span>
                    <span className="font-mono break-all">{appDir || "—"}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">
                      Commands run in:{" "}
                    </span>
                    <span className="font-mono break-all">{cwd || "—"}</span>
                  </div>
                  {commands.length > 0 ? (
                    <ol className="mt-1 flex flex-col gap-0.5 font-mono">
                      {commands.map((command, i) => (
                        <li key={i} className="break-all">
                          <span className="text-muted-foreground">$ </span>
                          {command}
                        </li>
                      ))}
                    </ol>
                  ) : (
                    <span className="text-muted-foreground">
                      This blueprint has no commands.
                    </span>
                  )}
                  <div className="text-muted-foreground">
                    Then <span className="font-mono">git init</span> runs in the
                    app folder if it isn’t a repository yet.
                  </div>
                </div>
              </FieldGroup>
            </div>
            <DialogFooter className="sm:justify-between">
              <Button variant="ghost" onClick={onManage}>
                Manage blueprints
              </Button>
              <div className="flex flex-col-reverse gap-2 sm:flex-row">
                <Button variant="outline" onClick={() => onOpenChange(false)}>
                  Cancel
                </Button>
                <Button disabled={!canRun} onClick={() => void handleRun()}>
                  Create app
                </Button>
              </div>
            </DialogFooter>
          </>
        ) : (
          <>
            {phase === "failed" ? (
              <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
                {error}
              </div>
            ) : failure ? (
              <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
                {failure}
                <p className="mt-1 text-xs text-muted-foreground">
                  Skip it to continue with the next command, or stop here.
                </p>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Running commands in{" "}
                <span className="font-mono break-all">{cwd}</span>…
              </p>
            )}
            <Terminal
              className="h-80"
              onHandle={handleTerminal}
              onData={(data) => {
                // Keystrokes only matter while a command runs; ignore the rest.
                if (running) void api.blueprints.sendInput(runId.current, data).catch(() => {})
              }}
              onResize={(cols, rows) => {
                void api.blueprints.resize(runId.current, cols, rows).catch(() => {})
              }}
            />
            <DialogFooter>
              {running && failure ? (
                <>
                  <Button variant="outline" onClick={() => handleResolve("abort")}>
                    Stop
                  </Button>
                  <Button onClick={() => handleResolve("skip")}>
                    Skip and continue
                  </Button>
                </>
              ) : running ? (
                <Button
                  variant="outline"
                  onClick={() => void api.blueprints.cancel(runId.current)}
                >
                  Cancel
                </Button>
              ) : (
                <>
                  <Button variant="outline" onClick={() => onOpenChange(false)}>
                    Close
                  </Button>
                  <Button variant="outline" onClick={() => setPhase("form")}>
                    Back
                  </Button>
                  <Button
                    variant="secondary"
                    disabled={adding}
                    onClick={() => void handleAddAnyway()}
                    title="Register the folder as an app without running anything else"
                  >
                    {adding ? "Adding…" : "Add app anyway"}
                  </Button>
                </>
              )}
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
