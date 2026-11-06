import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import {
  ArrowDownToLineIcon,
  ArrowUpFromLineIcon,
  CheckIcon,
  GitBranchIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"
import { DateRangeFilter } from "./date-range-filter"
import { BulkBar, IconAction } from "./list-actions"
import type { GitPanelModel } from "./use-git-panel"

type BranchesSectionModel = Pick<
  GitPanelModel,
  | "info"
  | "setBranchCreateOpen"
  | "setRenaming"
  | "setPendingDelete"
  | "setForceDelete"
  | "checkingOut"
  | "selBranches"
  | "setSelBranches"
  | "setBulk"
  | "setBulkForce"
  | "dateFrom"
  | "setDateFrom"
  | "dateTo"
  | "setDateTo"
  | "pulling"
  | "pushing"
  | "running"
  | "handleCheckout"
  | "handlePush"
  | "handlePull"
  | "allBranches"
  | "filtering"
  | "localBranches"
  | "currentBranch"
  | "deletableBranches"
  | "selectedBranches"
>

export function BranchesSection({ model }: { model: BranchesSectionModel }) {
  const {
    info,
    setBranchCreateOpen,
    setRenaming,
    setPendingDelete,
    setForceDelete,
    checkingOut,
    selBranches,
    setSelBranches,
    setBulk,
    setBulkForce,
    dateFrom,
    setDateFrom,
    dateTo,
    setDateTo,
    pulling,
    pushing,
    running,
    handleCheckout,
    handlePush,
    handlePull,
    allBranches,
    filtering,
    localBranches,
    currentBranch,
    deletableBranches,
    selectedBranches,
  } = model
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">Local branches</p>
            <Badge variant="secondary">
              {filtering
                ? `${localBranches.length} of ${allBranches.length}`
                : localBranches.length}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Create, rename and delete local branches. A branch checked out in a
            worktree cannot be deleted.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <DateRangeFilter
            disabled={!info || allBranches.length === 0}
            from={dateFrom}
            to={dateTo}
            onChange={(f, t) => {
              setDateFrom(f)
              setDateTo(t)
            }}
          />
          <Button
            variant="outline"
            disabled={!info}
            onClick={() => setBranchCreateOpen(true)}
          >
            <PlusIcon data-icon="inline-start" />
            New branch
          </Button>
        </div>
      </div>

      {!info ? (
        <Skeleton className="h-14 w-full" />
      ) : localBranches.length === 0 ? (
        <p className="rounded-lg border px-3 py-2 text-sm text-muted-foreground">
          {filtering
            ? "No branches have a last commit in this range."
            : "No local branches yet."}
        </p>
      ) : (
        <>
          {deletableBranches.length > 0 ? (
            <BulkBar
              label="branches"
              total={deletableBranches.length}
              selected={selectedBranches.length}
              onToggleAll={(all) =>
                setSelBranches(all ? deletableBranches.map((b) => b.name) : [])
              }
              onDelete={() => {
                setBulkForce(false)
                setBulk("branches")
              }}
            />
          ) : null}
          <ul className="divide-y rounded-lg border">
            {localBranches.map((b) => (
              <li
                key={b.name}
                className="flex flex-wrap items-center gap-3 px-3 py-2"
              >
                {deletableBranches.length > 0 ? (
                  <Checkbox
                    aria-label={`Select branch ${b.name}`}
                    disabled={!!b.worktree}
                    checked={selBranches.includes(b.name)}
                    onCheckedChange={(checked) =>
                      setSelBranches((cur) =>
                        checked
                          ? [...cur, b.name]
                          : cur.filter((n) => n !== b.name)
                      )
                    }
                  />
                ) : null}
                <div className="min-w-48 flex-1">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <GitBranchIcon className="size-3.5 text-muted-foreground" />
                    <span className="text-sm font-medium">{b.name}</span>
                    {b.name === currentBranch ? (
                      <Badge title={b.worktree}>this app</Badge>
                    ) : b.worktree ? (
                      <Badge variant="secondary" title={b.worktree}>
                        checked out
                      </Badge>
                    ) : null}
                    {b.upstream ? (
                      <Badge
                        variant="outline"
                        title={b.gone ? "Upstream branch was deleted" : ""}
                      >
                        {b.upstream}
                        {b.gone ? " · gone" : ""}
                        {b.ahead ? ` · ↑${b.ahead}` : ""}
                        {b.behind ? ` · ↓${b.behind}` : ""}
                      </Badge>
                    ) : null}
                  </div>
                  <p className="truncate text-xs text-muted-foreground">
                    {b.date ? new Date(b.date * 1000).toLocaleString() : ""}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {!b.upstream || b.gone || b.ahead > 0 ? (
                    <IconAction
                      label={
                        b.upstream && !b.gone
                          ? `Push ${b.ahead} commit${b.ahead === 1 ? "" : "s"} to ${b.upstream}`
                          : "Publish branch to a remote"
                      }
                      disabled={pushing !== null}
                      onClick={() => void handlePush(b)}
                    >
                      <ArrowUpFromLineIcon />
                    </IconAction>
                  ) : null}
                  {b.worktree && b.upstream && !b.gone ? (
                    <IconAction
                      label={
                        running
                          ? "Stop the app to pull"
                          : `Pull ${b.upstream} into ${b.name}`
                      }
                      disabled={pulling !== null || running}
                      onClick={() => void handlePull(b)}
                    >
                      <ArrowDownToLineIcon />
                    </IconAction>
                  ) : null}
                  {!b.worktree ? (
                    <IconAction
                      label={
                        running
                          ? "Stop the app to switch branches"
                          : "Check out in this app's folder"
                      }
                      disabled={checkingOut || running}
                      onClick={() => void handleCheckout(b)}
                    >
                      <CheckIcon />
                    </IconAction>
                  ) : null}
                  <IconAction
                    label="Rename branch"
                    onClick={() => setRenaming(b)}
                  >
                    <PencilIcon />
                  </IconAction>
                  <IconAction
                    label={
                      b.worktree
                        ? "Checked out in a worktree: cannot delete"
                        : "Delete branch"
                    }
                    variant="ghost"
                    disabled={!!b.worktree}
                    onClick={() => {
                      setForceDelete(false)
                      setPendingDelete(b)
                    }}
                  >
                    <Trash2Icon />
                  </IconAction>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}
