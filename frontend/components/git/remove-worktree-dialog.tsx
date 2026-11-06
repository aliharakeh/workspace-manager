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
type RemoveWorktreeDialogModel = Pick<
  GitPanelModel,
  | "pendingRemove"
  | "setPendingRemove"
  | "force"
  | "setForce"
  | "removing"
  | "handleConfirmRemove"
>

export function RemoveWorktreeDialog({
  model,
}: {
  model: RemoveWorktreeDialogModel
}) {
  const {
    pendingRemove,
    setPendingRemove,
    force,
    setForce,
    removing,
    handleConfirmRemove,
  } = model
  return (
    <>
      <AlertDialog
        open={!!pendingRemove}
        onOpenChange={(open) => {
          if (!open) setPendingRemove(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove worktree?</AlertDialogTitle>
            <AlertDialogDescription>
              Deletes the folder{" "}
              <span className="font-mono">{pendingRemove?.path}</span>. The
              branch{pendingRemove?.branch ? ` ${pendingRemove.branch}` : ""} is
              kept.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-remove-force"
              checked={force}
              onCheckedChange={(checked) => setForce(checked === true)}
            />
            <FieldLabel
              htmlFor="git-remove-force"
              className="text-xs font-normal text-muted-foreground"
            >
              Force: also discard uncommitted changes
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removing}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={removing}
              onClick={() => void handleConfirmRemove()}
            >
              {removing ? "Removing…" : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
