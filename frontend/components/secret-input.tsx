import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { EyeIcon, EyeOffIcon } from "lucide-react"
import * as React from "react"

type SecretInputProps = Omit<React.ComponentProps<typeof Input>, "type"> & {
  /**
   * A private value is masked until the eye is clicked. Otherwise this is a
   * plain text input.
   */
  secret: boolean
}

/**
 * An input that masks a private value by default and shows it while the eye is
 * toggled on. The reveal is per input and forgotten when the row unmounts.
 */
export function SecretInput({
  secret,
  className,
  disabled,
  ...props
}: SecretInputProps) {
  const [revealed, setRevealed] = React.useState(false)

  if (!secret) {
    return <Input className={className} disabled={disabled} {...props} />
  }

  return (
    <div className="relative flex flex-1 items-center">
      <Input
        {...props}
        type={revealed ? "text" : "password"}
        autoComplete="off"
        disabled={disabled}
        className={cn("pr-8", className)}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="absolute right-1"
        onClick={() => setRevealed((v) => !v)}
        aria-label={revealed ? "Hide value" : "Show value"}
        aria-pressed={revealed}
        title={revealed ? "Hide value" : "Show value"}
      >
        {revealed ? <EyeOffIcon /> : <EyeIcon />}
      </Button>
    </div>
  )
}
