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
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api"
import { PlusIcon } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

/** Common candidates, offered as one-click additions. */
const LINK_SUGGESTIONS = [
  "node_modules",
  ".env",
  ".env.local",
  ".venv",
  "vendor",
]

export function SharedLinksDialog({
  appId,
  links,
  open,
  onOpenChange,
  onSaved,
}: {
  appId: number
  links: string[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onSaved: (links: string[]) => void
}) {
  const [text, setText] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the draft is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setText(links.join("\n"))
  }

  const draft = text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)

  function addSuggestion(path: string) {
    setText([...draft, path].join("\n"))
  }

  async function handleSave() {
    setSaving(true)
    try {
      onSaved(await api.git.linksSave(appId, draft))
      toast.success("Shared files saved")
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Shared files</DialogTitle>
          <DialogDescription>
            Files and folders each worktree links to in this app's folder
            instead of having its own copy. They are linked when a worktree is
            added (unless turned off there), before Setup runs in a worktree, or
            with the link button on its row.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="git-shared-links">
              Paths, one per line
            </FieldLabel>
            <Textarea
              id="git-shared-links"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={"node_modules\n.env"}
              rows={6}
              className="font-mono text-xs"
            />
            <FieldDescription>
              Relative to the app folder. A worktree that already has a real
              file or folder at a path keeps it: delete it there to share the
              app's one. Changes made through a link change the app's own copy.
            </FieldDescription>
          </Field>
          <div className="flex flex-wrap gap-1.5">
            {LINK_SUGGESTIONS.filter((s) => !draft.includes(s)).map((s) => (
              <Button
                key={s}
                variant="outline"
                size="sm"
                className="font-mono text-xs"
                onClick={() => addSuggestion(s)}
              >
                <PlusIcon data-icon="inline-start" />
                {s}
              </Button>
            ))}
          </div>
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
