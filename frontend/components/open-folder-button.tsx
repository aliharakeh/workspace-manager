import { FolderOpenIcon } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"

type OpenFolderButtonProps = {
  appId: number
  iconOnly?: boolean
  className?: string
}

export function OpenFolderButton({
  appId,
  iconOnly,
  className,
}: OpenFolderButtonProps) {
  async function open() {
    try {
      await api.apps.openFolder(appId)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not open folder")
    }
  }

  function onClick(event: React.MouseEvent) {
    event.stopPropagation()
    void open()
  }

  if (!iconOnly) {
    return (
      <Button
        variant="outline"
        size="sm"
        className={className}
        onClick={onClick}
      >
        <FolderOpenIcon data-icon="inline-start" />
        Open folder
      </Button>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="outline"
            size="icon-sm"
            className={className}
            aria-label="Open folder"
            onClick={onClick}
          />
        }
      >
        <FolderOpenIcon />
      </TooltipTrigger>
      <TooltipContent>Open the project folder</TooltipContent>
    </Tooltip>
  )
}
