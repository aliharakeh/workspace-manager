import { useEffect, useState } from "react"
import { CodeIcon } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { Editor } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type OpenEditorButtonProps = {
  appId: number
  iconOnly?: boolean
}

// Installed editors don't change while the app runs: look them up once.
let editorsPromise: Promise<Editor[]> | null = null
function loadEditors() {
  editorsPromise ??= api.apps.editors().catch(() => [])
  return editorsPromise
}

export function OpenEditorButton({ appId, iconOnly }: OpenEditorButtonProps) {
  const [editors, setEditors] = useState<Editor[]>([])

  useEffect(() => {
    void loadEditors().then(setEditors)
  }, [])

  async function open(editor: Editor) {
    try {
      await api.apps.openInEditor(appId, editor.id)
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : `Could not open ${editor.name}`
      )
    }
  }

  if (editors.length === 0) return null

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            size={iconOnly ? "icon-sm" : "sm"}
            title="Open in editor"
            aria-label="Open in editor"
            onClick={(event) => event.stopPropagation()}
          />
        }
      >
        <CodeIcon data-icon={iconOnly ? undefined : "inline-start"} />
        {iconOnly ? null : "Open in editor"}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {editors.map((editor) => (
          <DropdownMenuItem
            key={editor.id}
            onClick={(event) => {
              event.stopPropagation()
              void open(editor)
            }}
          >
            {editor.name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
