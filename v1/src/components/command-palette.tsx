import { useEffect, useMemo, useRef, useState } from "react"
import { AppWindowIcon, FolderIcon, SearchIcon } from "lucide-react"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { Kbd } from "@/components/ui/kbd"
import { cn } from "@/lib/utils"
import { shortcutParts } from "@/lib/shortcuts"

export type PaletteItem = {
  type: "workspace" | "app"
  /** Workspace id for workspaces, app id for apps. */
  id: number
  workspaceId: number | null
  title: string
  subtitle?: string
}

type CommandPaletteProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: PaletteItem[]
  shortcut: string
  onSelect: (item: PaletteItem) => void
}

function Highlighted({ text, query }: { text: string; query: string }) {
  if (!query) return text
  const index = text.toLowerCase().indexOf(query.toLowerCase())
  if (index === -1) return text
  return (
    <>
      {text.slice(0, index)}
      <mark className="bg-transparent font-medium text-foreground">
        {text.slice(index, index + query.length)}
      </mark>
      {text.slice(index + query.length)}
    </>
  )
}

function matches(item: PaletteItem, query: string): boolean {
  if (!query) return true
  const q = query.toLowerCase()
  return (
    item.title.toLowerCase().includes(q) ||
    (item.subtitle?.toLowerCase().includes(q) ?? false)
  )
}

export function CommandPalette({
  open,
  onOpenChange,
  items,
  shortcut,
  onSelect,
}: CommandPaletteProps) {
  const [query, setQuery] = useState("")
  const [activeIndex, setActiveIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  const results = useMemo(() => {
    const q = query.trim().toLowerCase()
    const scored = items
      .filter((item) => matches(item, q))
      .map((item) => ({
        item,
        score: !q
          ? 0
          : item.title.toLowerCase().startsWith(q)
            ? 0
            : item.subtitle?.toLowerCase().startsWith(q)
              ? 1
              : 2,
      }))
    scored.sort((a, b) => a.score - b.score || a.item.title.localeCompare(b.item.title))
    return scored.map((entry) => entry.item)
  }, [items, query])

  const safeActiveIndex = Math.min(activeIndex, Math.max(results.length - 1, 0))

  useEffect(() => {
    const el = listRef.current?.querySelector('[data-active="true"]')
    el?.scrollIntoView({ block: "nearest" })
  }, [safeActiveIndex])

  function handleOpenChange(next: boolean) {
    if (next) {
      setQuery("")
      setActiveIndex(0)
    }
    onOpenChange(next)
  }

  function handleKeyDown(event: React.KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault()
      setActiveIndex((i) => Math.min(i + 1, results.length - 1))
    } else if (event.key === "ArrowUp") {
      event.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (event.key === "Enter") {
      event.preventDefault()
      // Activate the highlighted row exactly like a mouse click on it.
      const activeButton =
        listRef.current?.querySelector<HTMLButtonElement>(
          '[data-active="true"]'
        )
      if (activeButton) {
        activeButton.click()
        return
      }
      const item = results[safeActiveIndex]
      if (item) onSelect(item)
    }
  }

  const workspaces = results.filter((r) => r.type === "workspace")
  const apps = results.filter((r) => r.type === "app")

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        showCloseButton={false}
        className="top-[20vh] gap-0 overflow-hidden rounded-xl p-0 sm:max-w-2xl"
      >
        <div className="flex items-center gap-2.5 border-b px-4 py-3">
          <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setActiveIndex(0)
            }}
            onKeyDown={handleKeyDown}
            placeholder="Search workspaces and apps…"
            aria-label="Search workspaces and apps"
            autoComplete="off"
            spellCheck={false}
            className="h-6 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-muted-foreground"
          />
        </div>

        <div ref={listRef} className="max-h-[min(60vh,26rem)] overflow-y-auto p-1.5">
          {results.length === 0 ? (
            <p className="px-3 py-8 text-center text-sm text-muted-foreground">
              No results for “{query.trim()}”.
            </p>
          ) : (
            <>
              {workspaces.length > 0 ? (
                <PaletteSection
                  label="Workspaces"
                  items={workspaces}
                  offset={0}
                  query={query}
                  activeIndex={safeActiveIndex}
                  onHover={setActiveIndex}
                  onSelect={onSelect}
                />
              ) : null}
              {apps.length > 0 ? (
                <PaletteSection
                  label="Apps"
                  items={apps}
                  offset={workspaces.length}
                  query={query}
                  activeIndex={safeActiveIndex}
                  onHover={setActiveIndex}
                  onSelect={onSelect}
                />
              ) : null}
            </>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 border-t bg-muted/40 px-4 py-2">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Kbd>↑</Kbd>
            <Kbd>↓</Kbd>
            <span>to navigate</span>
            <Kbd>↵</Kbd>
            <span>to open</span>
            <Kbd>esc</Kbd>
            <span>to close</span>
          </div>
          {shortcut ? (
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              <span className="hidden sm:inline">Shortcut</span>
              {shortcutParts(shortcut).map((part) => (
                <Kbd key={part}>{part}</Kbd>
              ))}
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  )
}

type PaletteSectionProps = {
  label: string
  items: PaletteItem[]
  offset: number
  query: string
  activeIndex: number
  onHover: (index: number) => void
  onSelect: (item: PaletteItem) => void
}

function PaletteSection({
  label,
  items,
  offset,
  query,
  activeIndex,
  onHover,
  onSelect,
}: PaletteSectionProps) {
  return (
    <div className="mb-1">
      <p className="px-3 py-1 text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <ul>
        {items.map((item, i) => {
          const index = offset + i
          const active = index === activeIndex
          return (
            <li key={`${item.type}-${item.id}`}>
              <button
                type="button"
                data-active={active || undefined}
                onMouseEnter={() => onHover(index)}
                onClick={() => onSelect(item)}
                className={cn(
                  "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm outline-none",
                  active && "bg-accent text-accent-foreground"
                )}
              >
                {item.type === "workspace" ? (
                  <FolderIcon className="size-4 shrink-0 text-muted-foreground" />
                ) : (
                  <AppWindowIcon className="size-4 shrink-0 text-muted-foreground" />
                )}
                <span className="min-w-0 flex-1 truncate">
                  <Highlighted text={item.title} query={query.trim()} />
                </span>
                {item.subtitle ? (
                  <span className="max-w-56 shrink-0 truncate text-xs text-muted-foreground">
                    {item.subtitle}
                  </span>
                ) : null}
              </button>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
