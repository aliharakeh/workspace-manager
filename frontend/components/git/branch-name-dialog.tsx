import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"
import type { GitLocalBranch } from "@/lib/types"
import { useState } from "react"
import { toast } from "sonner"

/** Creates a local branch, or renames the supplied branch. */
export function BranchNameDialog({
  appId,
  branches,
  renaming,
  open,
  onOpenChange,
  onDone,
  onOutput,
}: {
  appId: number
  branches: string[]
  renaming: GitLocalBranch | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDone: () => void
  onOutput: (output: string) => void
}) {
  const [name, setName] = useState("")
  const [base, setBase] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the form is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setName(renaming?.name ?? "")
      setBase("")
    }
  }

  const isRename = !!renaming

  async function handleSave() {
    const next = name.trim()
    if (!next) {
      toast.error("Branch name is required")
      return
    }
    setSaving(true)
    try {
      if (renaming) {
        await api.git.branchRename(appId, renaming.name, next)
        toast.success(`Renamed ${renaming.name} to ${next}`)
      } else {
        await api.git.branchCreate(appId, next, base.trim())
        toast.success(`Branch ${next} created`)
      }
      onOpenChange(false)
      onDone()
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed"
      onOutput(msg)
      toast.error(`${renaming ? "Rename" : "Create"} failed: ${msg}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isRename ? "Rename branch" : "New branch"}</DialogTitle>
          <DialogDescription>
            {isRename ? (
              <>
                Renames <span className="font-mono">{renaming?.name}</span>. A
                worktree that has it checked out follows the new name; its
                upstream and remote branch are not renamed.
              </>
            ) : (
              "Creates a local branch without checking it out."
            )}
          </DialogDescription>
        </DialogHeader>
        <datalist id="git-base-branches">
          {branches.map((b) => (
            <option key={b} value={b} />
          ))}
        </datalist>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="git-branch-name">
              {isRename ? "New name" : "Branch name"}
            </FieldLabel>
            <Input
              id="git-branch-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !saving) void handleSave()
              }}
              placeholder="feature/my-change"
              autoFocus
            />
          </Field>
          {!isRename ? (
            <Field>
              <FieldLabel htmlFor="git-branch-base">Start from</FieldLabel>
              <Input
                id="git-branch-base"
                value={base}
                onChange={(e) => setBase(e.target.value)}
                placeholder="HEAD"
                list="git-base-branches"
              />
              <FieldDescription>
                A branch, tag or commit. Empty: the current HEAD of the app's
                folder.
              </FieldDescription>
            </Field>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleSave()}>
            {saving
              ? isRename
                ? "Renaming…"
                : "Creating…"
              : isRename
                ? "Rename"
                : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
