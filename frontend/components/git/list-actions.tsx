import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { Trash2Icon } from "lucide-react"

/** An icon-only button; the label is its tooltip and accessible name. */
export function IconAction({
  label,
  variant = "outline",
  disabled,
  onClick,
  children,
}: {
  label: string
  variant?: React.ComponentProps<typeof Button>["variant"]
  disabled?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant={variant}
            size="icon-sm"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

/** Select-all checkbox with a "Delete selected" button above a list. */
export function BulkBar({
  label,
  total,
  selected,
  onToggleAll,
  onDelete,
}: {
  label: string
  total: number
  selected: number
  onToggleAll: (all: boolean) => void
  onDelete: () => void
}) {
  return (
    <div className="flex items-center gap-2 px-1 text-sm">
      <Checkbox
        aria-label={`Select all ${label}`}
        checked={selected === total}
        onCheckedChange={(checked) => onToggleAll(checked === true)}
      />
      <span className="text-xs text-muted-foreground">
        {selected > 0 ? `${selected} of ${total} selected` : `Select ${label}`}
      </span>
      {/* Always laid out, so showing it does not change the bar's height. */}
      <Button
        variant="destructive"
        size="sm"
        className={cn("ml-auto", selected === 0 && "invisible")}
        tabIndex={selected === 0 ? -1 : undefined}
        aria-hidden={selected === 0}
        onClick={onDelete}
      >
        <Trash2Icon data-icon="inline-start" />
        Delete selected
      </Button>
    </div>
  )
}
