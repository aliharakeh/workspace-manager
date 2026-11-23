import type { CSSProperties } from "react"
import { FolderIcon, ImageUpIcon, XIcon } from "lucide-react"
import type { Workspace } from "@/lib/types"
import { useLogoUpload } from "@/hooks/use-logo-upload"
import { useWorkspaceLogo } from "@/hooks/use-workspace-logo"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

type WorkspaceLogoProps = {
  workspace: Pick<Workspace, "id" | "logo_path" | "color">
  className?: string
  style?: CSSProperties
}

/**
 * The workspace's uploaded logo, or a folder in the workspace's color when it
 * has none (or while it loads). Size it with `className`; it fills the box,
 * keeping its shape.
 */
export function WorkspaceLogo({ workspace, className, style }: WorkspaceLogoProps) {
  const src = useWorkspaceLogo(workspace)
  return (
    <LogoImage src={src} color={workspace.color} className={className} style={style} />
  )
}

type LogoImageProps = {
  /** An image source, or null to show the folder. */
  src: string | null
  color: string | null
  className?: string
  style?: CSSProperties
}

function LogoImage({ src, color, className, style }: LogoImageProps) {
  if (src) {
    return (
      <img
        src={src}
        alt=""
        draggable={false}
        className={cn("shrink-0 rounded-sm object-contain", className)}
        style={style}
      />
    )
  }
  return (
    <FolderIcon
      className={cn("shrink-0", className)}
      style={{ ...(color ? { color } : {}), ...style }}
    />
  )
}

type WorkspaceLogoPickerProps = {
  /** The logo to preview (a data URL), or null for none. */
  value: string | null
  color: string | null
  onChange: (logo: string | null) => void
  disabled?: boolean
}

/** A preview with Upload and Remove, for the workspace dialog. */
export function WorkspaceLogoPicker({
  value,
  color,
  onChange,
  disabled,
}: WorkspaceLogoPickerProps) {
  const upload = useLogoUpload(onChange)
  return (
    <div className="flex items-center gap-3">
      <button
        type="button"
        disabled={disabled}
        onClick={upload.open}
        title="Upload image"
        className="flex size-12 shrink-0 items-center justify-center rounded-lg border border-dashed bg-muted/40 outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      >
        <LogoImage
          src={value}
          color={color}
          className={value ? "size-10" : "size-5 opacity-70"}
        />
      </button>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={upload.open}
        >
          <ImageUpIcon data-icon="inline-start" />
          {value ? "Change image…" : "Upload image…"}
        </Button>
        {value ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onChange(null)}
          >
            <XIcon data-icon="inline-start" />
            Remove
          </Button>
        ) : null}
      </div>
      {upload.input}
    </div>
  )
}
