import { FolderOpenIcon } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"

type OpenFolderButtonProps = {
  appId: number
  iconOnly?: boolean
}

export function OpenFolderButton({ appId, iconOnly }: OpenFolderButtonProps) {
  async function open() {
    try {
      await api.apps.openFolder(appId)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not open folder")
    }
  }

  return (
    <Button
      variant="outline"
      size={iconOnly ? "icon-sm" : "sm"}
      title="Open folder"
      aria-label="Open folder"
      onClick={(event) => {
        event.stopPropagation()
        void open()
      }}
    >
      <FolderOpenIcon data-icon={iconOnly ? undefined : "inline-start"} />
      {iconOnly ? null : "Open folder"}
    </Button>
  )
}
