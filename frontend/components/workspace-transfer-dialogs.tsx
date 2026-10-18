import { useEffect, useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type {
  SecretsMode,
  Workspace,
  WorkspaceBundle,
  WorkspaceImportResult,
} from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

const SECRET_CHOICES: { value: SecretsMode; label: string; hint: string }[] = [
  {
    value: "placeholder",
    label: "Keep keys, hide secrets",
    hint: "Env vars and templates excluded from AI keep their key or path but lose their value. Safe to share.",
  },
  {
    value: "exclude",
    label: "Leave secrets out",
    hint: "Env vars and templates excluded from AI are not written at all.",
  },
  {
    value: "include",
    label: "Include everything",
    hint: "Writes real secret values into the file. Use it for moving to another machine, and keep the file private.",
  },
]

type ExportWorkspaceDialogProps = {
  workspace: Workspace | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ExportWorkspaceDialog({
  workspace,
  open,
  onOpenChange,
}: ExportWorkspaceDialogProps) {
  const [secrets, setSecrets] = useState<SecretsMode>("placeholder")
  const [busy, setBusy] = useState(false)

  async function handleExport() {
    if (!workspace) return
    setBusy(true)
    try {
      const result = await api.workspaces.exportToFile(workspace.id, { secrets })
      if (result.cancelled) return
      onOpenChange(false)
      toast.success(`Exported ${result.apps} app(s) to ${result.path}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Export failed")
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Export workspace</DialogTitle>
          <DialogDescription>
            Writes “{workspace?.name}” with its apps, config sets, env vars,
            templates and command lists to a JSON file. App folders are stored
            relative to the folder they share.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel>Secrets</FieldLabel>
            <div className="flex flex-col gap-2">
              {SECRET_CHOICES.map((choice) => (
                <label
                  key={choice.value}
                  className="flex cursor-pointer items-start gap-2 text-sm"
                >
                  <input
                    type="radio"
                    name="export-secrets"
                    className="mt-1"
                    checked={secrets === choice.value}
                    onChange={() => setSecrets(choice.value)}
                  />
                  <span>
                    <span className="font-medium">{choice.label}</span>
                    <span className="block text-xs text-muted-foreground">
                      {choice.hint}
                    </span>
                  </span>
                </label>
              ))}
            </div>
          </Field>
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={busy || !workspace} onClick={() => void handleExport()}>
            {busy ? "Exporting…" : "Export…"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type ImportWorkspaceDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImported: (result: WorkspaceImportResult) => void
}

export function ImportWorkspaceDialog({
  open,
  onOpenChange,
  onImported,
}: ImportWorkspaceDialogProps) {
  const [fileName, setFileName] = useState("")
  const [bundle, setBundle] = useState<WorkspaceBundle | null>(null)
  const [name, setName] = useState("")
  const [baseFolder, setBaseFolder] = useState("")
  const [folders, setFolders] = useState<Record<string, string>>({})
  const [preview, setPreview] = useState<WorkspaceImportResult | null>(null)
  const [previewError, setPreviewError] = useState("")
  const [busy, setBusy] = useState(false)

  // The dialog stays mounted: start empty each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setFileName("")
      setBundle(null)
      setName("")
      setBaseFolder("")
      setFolders({})
      setPreview(null)
      setPreviewError("")
    }
  }

  // Preview what would be created whenever the choices change.
  useEffect(() => {
    if (!bundle) return
    let cancelled = false
    api.workspaces
      .import(bundle, {
        name,
        base_folder: baseFolder,
        folders,
        dry_run: true,
      })
      .then((result) => {
        if (cancelled) return
        setPreview(result)
        setPreviewError("")
      })
      .catch((err) => {
        if (cancelled) return
        setPreview(null)
        setPreviewError(err instanceof Error ? err.message : "Invalid bundle")
      })
    return () => {
      cancelled = true
    }
  }, [bundle, name, baseFolder, folders])

  async function chooseFile() {
    try {
      const result = await api.workspaces.openBundleFile()
      if (result.cancelled || !result.bundle) return
      setFileName(result.path)
      setBundle(result.bundle)
      setName(result.bundle.workspace.name)
      setBaseFolder("")
      setFolders({})
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not read the file")
    }
  }

  async function browse(onPick: (path: string) => void, startDir?: string) {
    try {
      const picked = await api.fs.pickFolder({ startDir })
      if (!picked.cancelled && picked.path) onPick(picked.path)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not open the picker")
    }
  }

  async function handleImport() {
    if (!bundle) return
    setBusy(true)
    try {
      const result = await api.workspaces.import(bundle, {
        name,
        base_folder: baseFolder,
        folders,
        dry_run: false,
      })
      onImported(result)
      onOpenChange(false)
      toast.success(`Imported “${result.name}” with ${result.apps.length} app(s)`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Import failed")
    } finally {
      setBusy(false)
    }
  }

  const emptyValues = (preview?.apps ?? []).reduce(
    (n, a) => n + a.empty_env_vars + a.empty_templates,
    0
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import workspace</DialogTitle>
          <DialogDescription>
            Creates a new workspace from an exported bundle. Nothing existing is
            changed.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel>Bundle file</FieldLabel>
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={() => void chooseFile()}>
                {bundle ? "Choose another…" : "Choose file…"}
              </Button>
              <span className="truncate text-xs text-muted-foreground">
                {fileName}
              </span>
            </div>
          </Field>
          {bundle ? (
            <>
              <Field>
                <FieldLabel htmlFor="import-name">Workspace name</FieldLabel>
                <Input
                  id="import-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="import-base">Base folder</FieldLabel>
                <div className="flex gap-2">
                  <Input
                    id="import-base"
                    value={baseFolder}
                    onChange={(e) => setBaseFolder(e.target.value)}
                    placeholder={bundle.source_root || "Folder the apps live under"}
                  />
                  <Button
                    variant="outline"
                    onClick={() =>
                      void browse(setBaseFolder, baseFolder || bundle.source_root)
                    }
                  >
                    Browse…
                  </Button>
                </div>
                <span className="text-xs text-muted-foreground">
                  App folders are placed under it. Leave empty to keep the
                  original location.
                </span>
              </Field>
              {previewError ? (
                <p className="text-sm text-destructive">{previewError}</p>
              ) : null}
              {preview ? (
                <div className="flex flex-col gap-2">
                  <FieldLabel>Apps to create</FieldLabel>
                  <div className="flex max-h-64 flex-col gap-2 overflow-y-auto">
                    {preview.apps.map((app) => (
                      <div key={app.key} className="rounded-md border p-2 text-sm">
                        <div className="flex items-baseline justify-between gap-2">
                          <span className="font-medium">{app.name}</span>
                          <span className="text-xs text-muted-foreground">
                            {app.config_sets} config set(s) · {app.env_vars} env
                            var(s) · {app.templates} template(s) · {app.commands}{" "}
                            command(s)
                          </span>
                        </div>
                        <div className="mt-1 flex gap-2">
                          <Input
                            value={folders[app.key] ?? app.folder}
                            onChange={(e) =>
                              setFolders((prev) => ({
                                ...prev,
                                [app.key]: e.target.value,
                              }))
                            }
                            aria-label={`Folder for ${app.name}`}
                          />
                          <Button
                            variant="outline"
                            onClick={() =>
                              void browse(
                                (path) =>
                                  setFolders((prev) => ({
                                    ...prev,
                                    [app.key]: path,
                                  })),
                                folders[app.key] ?? app.folder
                              )
                            }
                          >
                            Browse…
                          </Button>
                        </div>
                        {app.folder_missing ? (
                          <p className="mt-1 text-xs text-amber-600">
                            This folder does not exist yet. The app is still
                            imported; point it at the right folder later.
                          </p>
                        ) : null}
                      </div>
                    ))}
                  </div>
                  {emptyValues > 0 ? (
                    <p className="text-xs text-muted-foreground">
                      {emptyValues} secret value(s) were not exported. They are
                      created empty and hidden from AI; fill them in after
                      importing.
                    </p>
                  ) : null}
                </div>
              ) : null}
            </>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={busy || !bundle || !preview || !name.trim()}
            onClick={() => void handleImport()}
          >
            {busy ? "Importing…" : "Import"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
