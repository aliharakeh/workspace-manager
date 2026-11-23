import { useRef } from "react"
import { toast } from "sonner"
import { readLogoFile } from "@/lib/workspace-logo"

/** Opens the file chooser for a logo image and returns it ready to save. */
export function useLogoUpload(onPicked: (logo: string) => void) {
  const inputRef = useRef<HTMLInputElement>(null)
  const input = (
    <input
      ref={inputRef}
      type="file"
      accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml,image/bmp,image/x-icon"
      className="hidden"
      onChange={async (e) => {
        const file = e.target.files?.[0]
        e.target.value = "" // so the same file can be chosen again
        if (!file) return
        try {
          onPicked(await readLogoFile(file))
        } catch (err) {
          toast.error(err instanceof Error ? err.message : "Could not read the image")
        }
      }}
    />
  )
  return { input, open: () => inputRef.current?.click() }
}
