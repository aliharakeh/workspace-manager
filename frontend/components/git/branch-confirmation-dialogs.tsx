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
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldLabel } from "@/components/ui/field"
import type { GitPanelModel } from "./use-git-panel"
type BranchConfirmationDialogsModel = Pick<
  GitPanelModel,
  | "pendingDelete"
  | "setPendingDelete"
  | "forceDelete"
  | "setForceDelete"
  | "deleting"
  | "pushing"
  | "rejected"
  | "setRejected"
  | "handlePush"
  | "handleConfirmDeleteBranch"
>

export function BranchConfirmationDialogs({
  model,
}: {
  model: BranchConfirmationDialogsModel
}) {
  const {
    pendingDelete,
    setPendingDelete,
    forceDelete,
    setForceDelete,
    deleting,
    pushing,
    rejected,
    setRejected,
    handlePush,
    handleConfirmDeleteBranch,
  } = model
  return (
    <>
      <AlertDialog
        open={!!rejected}
        onOpenChange={(open) => {
          if (!open) setRejected(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Push rejected</AlertDialogTitle>
            <AlertDialogDescription>
              The remote has commits{" "}
              <span className="font-mono">{rejected?.name}</span> does not. Pull
              first to merge them, or force push with lease: it overwrites the
              remote branch, but only if nobody pushed to it since your last
              fetch.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pushing !== null}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={pushing !== null}
              onClick={(event) => {
                // Keep the dialog open until the push finishes.
                event.preventDefault()
                if (rejected) void handlePush(rejected, true)
              }}
            >
              {pushing !== null ? "Pushing…" : "Force push"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog
        open={!!pendingDelete}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete branch?</AlertDialogTitle>
            <AlertDialogDescription>
              Deletes the local branch{" "}
              <span className="font-mono">{pendingDelete?.name}</span>. A branch
              that is not fully merged is refused unless you force it. Its
              remote branch is not touched.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-delete-branch-force"
              checked={forceDelete}
              onCheckedChange={(checked) => setForceDelete(checked === true)}
            />
            <FieldLabel
              htmlFor="git-delete-branch-force"
              className="text-xs font-normal text-muted-foreground"
            >
              Force: delete even if it has unmerged commits
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={() => void handleConfirmDeleteBranch()}
            >
              {deleting ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
