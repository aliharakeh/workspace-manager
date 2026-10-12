import { useEffect, useRef, useState } from "react"
import { BanIcon, CheckIcon } from "lucide-react"
import { cn } from "@/lib/utils"
import { WORKSPACE_COLORS } from "@/lib/workspace-colors"

type WorkspaceColorPickerProps = {
  value: string | null
  onChange: (color: string | null) => void
  disabled?: boolean
  /** Smaller swatches, for a page header. */
  compact?: boolean
  className?: string
}

const DEFAULT_CUSTOM = "#3b82f6"

export function WorkspaceColorPicker({
  value,
  onChange,
  disabled,
  compact,
  className,
}: WorkspaceColorPickerProps) {
  const isPreset = WORKSPACE_COLORS.some((c) => c.value === value)
  const customValue = value !== null && !isPreset ? value : null

  // While the native picker is open its color changes continuously; that is
  // only previewed here, and onChange runs once when the picker closes.
  const [draft, setDraft] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const onChangeRef = useRef(onChange)
  useEffect(() => {
    onChangeRef.current = onChange
  })
  useEffect(() => {
    const input = inputRef.current
    if (!input) return
    // React's onChange is the native "input" event; "change" is the commit.
    const commit = () => {
      setDraft(null)
      onChangeRef.current(input.value)
    }
    input.addEventListener("change", commit)
    return () => input.removeEventListener("change", commit)
  }, [])

  const customShown = draft ?? customValue
  const dot = compact ? "size-4" : "size-6"
  const glyph = compact ? "size-2.5" : "size-3.5"

  return (
    <div
      role="group"
      aria-label="Workspace color"
      className={cn("flex flex-wrap items-center", compact ? "gap-1.5" : "gap-2", className)}
    >
      <button
        type="button"
        title="No color"
        aria-label="No color"
        aria-pressed={value === null}
        disabled={disabled}
        onClick={() => onChange(null)}
        className={cn(
          dot,
          "flex items-center justify-center rounded-full border border-dashed text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50",
          value === null && "border-solid border-foreground"
        )}
      >
        <BanIcon className={glyph} />
      </button>
      {WORKSPACE_COLORS.map((c) => (
        <button
          key={c.value}
          type="button"
          title={c.name}
          aria-label={c.name}
          aria-pressed={value === c.value}
          disabled={disabled}
          onClick={() => onChange(c.value)}
          style={{ backgroundColor: c.value }}
          className={cn(
            dot,
            "flex items-center justify-center rounded-full text-white outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
          )}
        >
          {value === c.value ? <CheckIcon className={glyph} /> : null}
        </button>
      ))}
      <label
        title="Custom color"
        className={cn(
          dot,
          "relative flex cursor-pointer items-center justify-center rounded-full text-white focus-within:ring-2 focus-within:ring-ring",
          disabled && "pointer-events-none opacity-50"
        )}
        style={{
          background:
            customShown ??
            "conic-gradient(#ef4444, #f59e0b, #22c55e, #3b82f6, #8b5cf6, #ec4899, #ef4444)",
        }}
      >
        {customShown ? <CheckIcon className={glyph} /> : null}
        <input
          ref={inputRef}
          type="color"
          aria-label="Custom color"
          disabled={disabled}
          value={customShown ?? DEFAULT_CUSTOM}
          onChange={(e) => setDraft(e.target.value)}
          className="absolute inset-0 size-full cursor-pointer opacity-0"
        />
      </label>
    </div>
  )
}
