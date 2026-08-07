import { useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { App } from "@/lib/types"
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

type AppDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspaceId: number
  app?: App | null
  onSaved: (app: App) => void
}

export function AppDialog({
  open,
  onOpenChange,
  workspaceId,
  app,
  onSaved,
}: AppDialogProps) {
  const [name, setName] = useState(app?.name ?? "")
  const [projectPath, setProjectPath] = useState(app?.project_path ?? "")
  const [saving, setSaving] = useState(false)
  const isEdit = !!app

  function handleOpenChange(next: boolean) {
    if (next) {
      setName(app?.name ?? "")
      setProjectPath(app?.project_path ?? "")
    }
    onOpenChange(next)
  }

  async function handleSave() {
    if (!name.trim() || !projectPath.trim()) {
      toast.error("Name and project path are required")
      return
    }
    setSaving(true)
    try {
      const saved = isEdit
        ? await api.apps.update(app.id, {
            name: name.trim(),
            project_path: projectPath.trim(),
          })
        : await api.apps.create(workspaceId, {
            name: name.trim(),
            project_path: projectPath.trim(),
          })
      onSaved(saved)
      onOpenChange(false)
      toast.success(isEdit ? "App updated" : "App created")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit app" : "New app"}</DialogTitle>
          <DialogDescription>
            Point at a local project directory this runner can use as cwd.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="app-name">Name</FieldLabel>
            <Input
              id="app-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="API server"
              autoFocus
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="app-path">Project path</FieldLabel>
            <Input
              id="app-path"
              value={projectPath}
              onChange={(e) => setProjectPath(e.target.value)}
              placeholder="C:\Projects\my-app"
            />
          </Field>
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleSave()}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type DeleteAppDialogProps = {
  app: App | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeleted: (id: number) => void
}

export function DeleteAppDialog({
  app,
  open,
  onOpenChange,
  onDeleted,
}: DeleteAppDialogProps) {
  const [deleting, setDeleting] = useState(false)

  async function handleDelete() {
    if (!app) return
    setDeleting(true)
    try {
      await api.apps.delete(app.id)
      onDeleted(app.id)
      onOpenChange(false)
      toast.success("App deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setDeleting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete app?</AlertDialogTitle>
          <AlertDialogDescription>
            This will permanently delete “{app?.name}” and all of its config
            sets (env vars, templates, and run configs).
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={() => void handleDelete()}
          >
            {deleting ? "Deleting…" : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
