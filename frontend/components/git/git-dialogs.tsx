import { AddWorktreeDialog } from "./add-worktree-dialog"
import { BranchConfirmationDialogs } from "./branch-confirmation-dialogs"
import { BranchNameDialog } from "./branch-name-dialog"
import { BulkDeleteDialog } from "./bulk-delete-dialog"
import { MergeConflictsDialog } from "./merge-conflicts-dialog"
import { RemoveWorktreeDialog } from "./remove-worktree-dialog"
import { SharedLinksDialog } from "./shared-links-dialog"
import type { GitPanelModel } from "./use-git-panel"

export function GitDialogs({ model }: { model: GitPanelModel }) {
  const {
    appId,
    info,
    links,
    setLinks,
    linksOpen,
    setLinksOpen,
    setOutput,
    addOpen,
    setAddOpen,
    branchCreateOpen,
    setBranchCreateOpen,
    renaming,
    setRenaming,
    load,
  } = model
  return (
    <>
      {info ? (
        <AddWorktreeDialog
          appId={appId}
          info={info}
          links={links}
          open={addOpen}
          onOpenChange={setAddOpen}
          onAdded={() => void load()}
          onOutput={setOutput}
        />
      ) : null}
      <SharedLinksDialog
        appId={appId}
        links={links}
        open={linksOpen}
        onOpenChange={setLinksOpen}
        onSaved={setLinks}
      />
      {info ? (
        <BranchNameDialog
          appId={appId}
          branches={info.branches}
          renaming={renaming}
          open={branchCreateOpen || !!renaming}
          onOpenChange={(open) => {
            if (!open) {
              setBranchCreateOpen(false)
              setRenaming(null)
            }
          }}
          onDone={() => void load()}
          onOutput={setOutput}
        />
      ) : null}
      <MergeConflictsDialog model={model} />
      <BranchConfirmationDialogs model={model} />
      <BulkDeleteDialog model={model} />
      <RemoveWorktreeDialog model={model} />
    </>
  )
}
