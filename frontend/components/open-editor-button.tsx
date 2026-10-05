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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"

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

  const button = (
    <Button
      variant="outline"
      size={iconOnly ? "icon-sm" : "sm"}
      aria-label="Open in editor"
      onClick={(event) => event.stopPropagation()}
    />
  )

  return (
    <DropdownMenu>
      {iconOnly ? (
        <Tooltip>
          <TooltipTrigger
            render={<DropdownMenuTrigger render={button} />}
          >
            <CodeIcon />
          </TooltipTrigger>
          <TooltipContent>Open the project in an editor</TooltipContent>
        </Tooltip>
      ) : (
        <DropdownMenuTrigger render={button}>
          <CodeIcon data-icon="inline-start" />
          Open in editor
        </DropdownMenuTrigger>
      )}
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
