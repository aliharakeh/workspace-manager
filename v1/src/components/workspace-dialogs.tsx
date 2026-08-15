import { useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { Workspace } from "@/lib/types"
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

type WorkspaceDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspace?: Workspace | null
  onSaved: (workspace: Workspace) => void
}

export function WorkspaceDialog({
  open,
  onOpenChange,
  workspace,
  onSaved,
}: WorkspaceDialogProps) {
  const [name, setName] = useState(workspace?.name ?? "")
  const [saving, setSaving] = useState(false)
  const isEdit = !!workspace

  function handleOpenChange(next: boolean) {
    if (next) setName(workspace?.name ?? "")
    onOpenChange(next)
  }

  async function handleSave() {
    if (!name.trim()) {
      toast.error("Name is required")
      return
    }
    setSaving(true)
    try {
      const saved = isEdit
        ? await api.workspaces.update(workspace.id, { name: name.trim() })
        : await api.workspaces.create({ name: name.trim() })
      onSaved(saved)
      onOpenChange(false)
      toast.success(isEdit ? "Workspace updated" : "Workspace created")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit workspace" : "New workspace"}</DialogTitle>
          <DialogDescription>
            Workspaces group related apps you want to run together.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="workspace-name">Name</FieldLabel>
            <Input
              id="workspace-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My projects"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleSave()
              }}
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

type DeleteWorkspaceDialogProps = {
  workspace: Workspace | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeleted: (id: number) => void
}

export function DeleteWorkspaceDialog({
  workspace,
  open,
  onOpenChange,
  onDeleted,
}: DeleteWorkspaceDialogProps) {
  const [deleting, setDeleting] = useState(false)

  async function handleDelete() {
    if (!workspace) return
    setDeleting(true)
    try {
      await api.workspaces.delete(workspace.id)
      onDeleted(workspace.id)
      onOpenChange(false)
      toast.success("Workspace deleted")
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
          <AlertDialogTitle>Delete workspace?</AlertDialogTitle>
          <AlertDialogDescription>
            This will permanently delete “{workspace?.name}” and all of its apps,
            env vars, templates, and run config.
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
