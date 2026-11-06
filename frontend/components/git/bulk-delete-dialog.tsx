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
type BulkDeleteDialogModel = Pick<
  GitPanelModel,
  | "bulk"
  | "setBulk"
  | "bulkForce"
  | "setBulkForce"
  | "bulkWithBranches"
  | "setBulkWithBranches"
  | "bulkBusy"
  | "handleBulkConfirm"
  | "selectedWorktrees"
  | "selectedBranches"
>

export function BulkDeleteDialog({ model }: { model: BulkDeleteDialogModel }) {
  const {
    bulk,
    setBulk,
    bulkForce,
    setBulkForce,
    bulkWithBranches,
    setBulkWithBranches,
    bulkBusy,
    handleBulkConfirm,
    selectedWorktrees,
    selectedBranches,
  } = model
  return (
    <>
      <AlertDialog
        open={bulk !== null}
        onOpenChange={(open) => {
          if (!open && !bulkBusy) setBulk(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete{" "}
              {bulk === "worktrees"
                ? selectedWorktrees.length
                : selectedBranches.length}{" "}
              {bulk === "worktrees" ? "worktrees" : "branches"}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {bulk === "worktrees"
                ? "Deletes these worktree folders:"
                : "Deletes these local branches (their remote branches are not touched):"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <ul className="max-h-40 overflow-y-auto rounded-md border p-2 font-mono text-xs">
            {bulk === "worktrees"
              ? selectedWorktrees.map((w) => (
                  <li key={w.path} className="truncate" title={w.path}>
                    {w.branch || w.path}
                  </li>
                ))
              : selectedBranches.map((b) => (
                  <li key={b.name} className="truncate">
                    {b.name}
                  </li>
                ))}
          </ul>
          {bulk === "worktrees" ? (
            <Field orientation="horizontal" className="gap-2">
              <Checkbox
                id="git-bulk-with-branches"
                checked={bulkWithBranches}
                onCheckedChange={(checked) =>
                  setBulkWithBranches(checked === true)
                }
              />
              <FieldLabel
                htmlFor="git-bulk-with-branches"
                className="text-xs font-normal text-muted-foreground"
              >
                Also delete their local branches
              </FieldLabel>
            </Field>
          ) : null}
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-bulk-force"
              checked={bulkForce}
              onCheckedChange={(checked) => setBulkForce(checked === true)}
            />
            <FieldLabel
              htmlFor="git-bulk-force"
              className="text-xs font-normal text-muted-foreground"
            >
              {bulk === "worktrees"
                ? "Force: also discard uncommitted changes and delete unmerged branches"
                : "Force: delete even if they have unmerged commits"}
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={bulkBusy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={bulkBusy}
              onClick={(event) => {
                event.preventDefault()
                void handleBulkConfirm()
              }}
            >
              {bulkBusy ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
