import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Input } from "@/components/ui/input"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { CalendarIcon } from "lucide-react"

// "YYYY-MM-DDTHH:mm" in local time, the value format of the range filter.
function splitStamp(v: string): { date?: Date; time: string } {
  if (!v) return { time: "" }
  const [d, t = ""] = v.split("T")
  const [y, m, day] = d.split("-").map(Number)
  return { date: new Date(y, m - 1, day), time: t }
}

function joinStamp(date: Date | undefined, time: string, fallback: string) {
  if (!date) return ""
  const p = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}T${time || fallback}`
}

export function DateRangeFilter({
  from,
  to,
  disabled,
  onChange,
}: {
  from: string
  to: string
  disabled?: boolean
  onChange: (from: string, to: string) => void
}) {
  const f = splitStamp(from)
  const t = splitStamp(to)
  const fmt = (v: string) =>
    new Date(v).toLocaleString([], {
      dateStyle: "medium",
      timeStyle: "short",
    })
  const label =
    from || to
      ? `${from ? fmt(from) : "…"} – ${to ? fmt(to) : "…"}`
      : "Filter by date"
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button variant="outline" disabled={disabled}>
            <CalendarIcon data-icon="inline-start" />
            {label}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-auto gap-3">
        <Calendar
          mode="range"
          numberOfMonths={1}
          selected={{ from: f.date, to: t.date }}
          onSelect={(r) =>
            onChange(
              joinStamp(r?.from, f.time, "00:00"),
              joinStamp(r?.to, t.time, "23:59")
            )
          }
        />
        <div className="flex items-center gap-2">
          <Input
            type="time"
            aria-label="From time"
            disabled={!f.date}
            value={f.time}
            onChange={(e) =>
              onChange(joinStamp(f.date, e.target.value, "00:00"), to)
            }
          />
          <span className="text-muted-foreground">–</span>
          <Input
            type="time"
            aria-label="To time"
            disabled={!t.date}
            value={t.time}
            onChange={(e) =>
              onChange(from, joinStamp(t.date, e.target.value, "23:59"))
            }
          />
        </div>
        <Button
          variant="ghost"
          size="sm"
          disabled={!from && !to}
          onClick={() => onChange("", "")}
        >
          Clear
        </Button>
      </PopoverContent>
    </Popover>
  )
}
