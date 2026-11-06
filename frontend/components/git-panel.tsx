import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { DownloadCloudIcon, FolderGit2Icon, RefreshCwIcon } from "lucide-react"
import { BranchesSection } from "./git/branches-section"
import { GitDialogs } from "./git/git-dialogs"
import { useGitPanel, type GitPanelProps } from "./git/use-git-panel"
import { WorktreesSection } from "./git/worktrees-section"

export function GitPanel(props: GitPanelProps) {
  const model = useGitPanel(props)
  const { info, fetching, output, load, handleFetch } = model

  if (info && !info.is_repo) {
    return (
      <Empty className="border">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <FolderGit2Icon />
          </EmptyMedia>
          <EmptyTitle>Not a git repository</EmptyTitle>
          <EmptyDescription>
            The project path is not inside a git repository.
          </EmptyDescription>
        </EmptyHeader>
        <Button variant="outline" onClick={() => void load()}>
          <RefreshCwIcon data-icon="inline-start" />
          Check again
        </Button>
      </Empty>
    )
  }
  return (
    <>
      <div className="flex flex-col gap-6">
        <section className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-sm font-medium">Remotes</p>
              <p className="text-sm text-muted-foreground">
                Fetch every remote and prune branches deleted upstream.
              </p>
            </div>
            <Button disabled={fetching} onClick={() => void handleFetch()}>
              <DownloadCloudIcon data-icon="inline-start" />
              {fetching ? "Fetching…" : "Fetch all & prune"}
            </Button>
          </div>
        </section>
        <WorktreesSection model={model} />
        <BranchesSection model={model} />
        {output != null ? (
          <section className="flex flex-col gap-2">
            <p className="text-sm font-medium">Last git output</p>
            <pre className="max-h-40 overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">
              {output}
            </pre>
          </section>
        ) : null}
      </div>
      <GitDialogs model={model} />
    </>
  )
}
