import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import type { GitInfo } from "@/lib/types"
import { useState } from "react"
import { toast } from "sonner"
import { reportLinkStates } from "./worktree-links"

/** Mirrors DefaultWorktreePath in gui/git: sibling "<repo>-<branch>". */
function defaultWorktreePath(mainPath: string, branch: string): string {
  const slug = branch.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "")
  return `${mainPath}-${slug}`
}

export function AddWorktreeDialog({
  appId,
  info,
  links,
  open,
  onOpenChange,
  onAdded,
  onOutput,
}: {
  appId: number
  info: GitInfo
  links: string[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdded: () => void
  onOutput: (output: string) => void
}) {
  const [branch, setBranch] = useState("")
  const [newBranch, setNewBranch] = useState(true)
  const [base, setBase] = useState("")
  const [path, setPath] = useState("")
  const [link, setLink] = useState(true)
  const [linkText, setLinkText] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the form is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setBranch("")
      setNewBranch(true)
      setBase("")
      setPath("")
      setLink(links.length > 0)
      setLinkText(links.join("\n"))
    }
  }

  const linkPaths = linkText
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
  // Paths that differ from the app's become this worktree's own list.
  const ownLinks = linkPaths.join("\n") !== links.join("\n")
  const mainPath = info.worktrees[0]?.path ?? ""
  const defaultPath =
    branch.trim() && mainPath
      ? defaultWorktreePath(mainPath, branch.trim())
      : "Sibling folder <repo>-<branch>"

  async function handleAdd() {
    if (!branch.trim()) {
      toast.error("Branch is required")
      return
    }
    setSaving(true)
    try {
      const res = await api.git.worktreeAdd(appId, {
        branch: branch.trim(),
        new_branch: newBranch,
        base: newBranch ? base.trim() : "",
        path: path.trim(),
        link_shared: link && linkPaths.length > 0,
        links: link && ownLinks && linkPaths.length ? linkPaths : undefined,
      })
      toast.success(`Worktree created at ${res.path}`)
      onOpenChange(false)
      onAdded()
      if (res.links.length) reportLinkStates(res.links, onOutput)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add worktree")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add worktree</DialogTitle>
          <DialogDescription>
            Check out a branch into its own folder.
          </DialogDescription>
        </DialogHeader>
        <datalist id="git-branches">
          {info.branches.map((b) => (
            <option key={b} value={b} />
          ))}
        </datalist>
        <FieldGroup>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-new-branch"
              checked={newBranch}
              onCheckedChange={(checked) => setNewBranch(checked === true)}
            />
            <FieldLabel htmlFor="git-new-branch" className="font-normal">
              Create a new branch
            </FieldLabel>
          </Field>
          <Field>
            <FieldLabel htmlFor="git-branch">
              {newBranch ? "New branch name" : "Branch"}
            </FieldLabel>
            <Input
              id="git-branch"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder={newBranch ? "feature/my-change" : "main"}
              list={newBranch ? undefined : "git-branches"}
              autoFocus
            />
          </Field>
          {newBranch ? (
            <Field>
              <FieldLabel htmlFor="git-base">Start from</FieldLabel>
              <Input
                id="git-base"
                value={base}
                onChange={(e) => setBase(e.target.value)}
                placeholder="HEAD"
                list="git-branches"
              />
            </Field>
          ) : null}
          <Field>
            <FieldLabel htmlFor="git-path">Folder</FieldLabel>
            <Input
              id="git-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder={defaultPath}
              className="font-mono text-xs"
            />
          </Field>
          <Field orientation="horizontal">
            <FieldContent>
              <FieldLabel htmlFor="git-link-shared">
                Link shared files
              </FieldLabel>
              <FieldDescription>
                Link these files and folders to the app's folder as soon as the
                worktree is created.
              </FieldDescription>
            </FieldContent>
            <Switch
              id="git-link-shared"
              checked={link}
              onCheckedChange={setLink}
            />
          </Field>
          {link ? (
            <Field>
              <Textarea
                id="git-link-paths"
                aria-label="Shared files for this worktree"
                value={linkText}
                onChange={(e) => setLinkText(e.target.value)}
                placeholder={"node_modules\n.env"}
                rows={4}
                className="font-mono text-xs"
              />
              <FieldDescription>
                {ownLinks ? (
                  <>
                    This worktree keeps its own list for later setups; the app's
                    Shared files stay unchanged.{" "}
                    <button
                      type="button"
                      className="underline underline-offset-4"
                      onClick={() => setLinkText(links.join("\n"))}
                    >
                      Use the app's list
                    </button>
                  </>
                ) : (
                  "One path per line, relative to the app folder. Starts as the app's Shared files; edit it to change this worktree only."
                )}
              </FieldDescription>
            </Field>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleAdd()}>
            {saving ? "Adding…" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
