import { OpenEditorButton } from "@/components/open-editor-button"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { GitPanelModel } from "./use-git-panel"
type MergeConflictsDialogModel = Pick<
  GitPanelModel,
  "appId" | "conflicts" | "setConflicts"
>

export function MergeConflictsDialog({
  model,
}: {
  model: MergeConflictsDialogModel
}) {
  const { appId, conflicts, setConflicts } = model
  return (
    <>
      <Dialog
        open={!!conflicts}
        onOpenChange={(open) => {
          if (!open) setConflicts(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Merge conflicts</DialogTitle>
            <DialogDescription>
              The pull stopped because these files conflict. Open them in an
              editor, resolve the conflict markers, then commit in{" "}
              <span className="font-mono">{conflicts?.path}</span> to finish the
              merge.
            </DialogDescription>
          </DialogHeader>
          <ul className="max-h-64 divide-y overflow-auto rounded-lg border">
            {conflicts?.conflicts.map((file) => (
              <li key={file} className="flex items-center gap-2 px-3 py-2">
                <span
                  className="min-w-0 flex-1 truncate font-mono text-xs"
                  title={file}
                >
                  {conflicts.path && file.startsWith(conflicts.path)
                    ? file.slice(conflicts.path.length).replace(/^[\\/]/, "")
                    : file}
                </span>
                <OpenEditorButton appId={appId} filePath={file} iconOnly />
              </li>
            ))}
          </ul>
          <DialogFooter>
            <Button onClick={() => setConflicts(null)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
