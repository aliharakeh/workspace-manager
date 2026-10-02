import { useEffect, useRef, useState } from "react"
import { Terminal as XTerm, type ITheme } from "@xterm/xterm"
import { ClipboardAddon, type IClipboardProvider } from "@xterm/addon-clipboard"
import { FitAddon } from "@xterm/addon-fit"
import { SearchAddon, type ISearchOptions } from "@xterm/addon-search"
import { Unicode11Addon } from "@xterm/addon-unicode11"
import { WebLinksAddon } from "@xterm/addon-web-links"
import { WebglAddon } from "@xterm/addon-webgl"
import "@xterm/xterm/css/xterm.css"
import {
  ArrowDownToLineIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  EraserIcon,
  SearchIcon,
  XIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useTheme } from "@/components/theme-provider"

export type TerminalHandle = {
  write: (data: Uint8Array | string) => void
  reset: () => void
  focus: () => void
}

type TerminalProps = {
  /** Output only: keystrokes are ignored and never sent anywhere. */
  readOnly?: boolean
  /** Called with a handle once the terminal exists, and with null when it is gone. */
  onHandle?: (handle: TerminalHandle | null) => void
  /** Keystrokes and pastes, as the bytes a terminal sends to its program. */
  onData?: (data: string) => void
  /** The terminal's size in characters, after it fitted its container. */
  onResize?: (cols: number, rows: number) => void
  className?: string
}

const FONT_FAMILY =
  '"Cascadia Code", "Cascadia Mono", "JetBrains Mono", "SF Mono", Menlo, Consolas, ui-monospace, monospace'

const SEARCH_OPTIONS: ISearchOptions = {
  caseSensitive: false,
  decorations: {
    matchBackground: "#facc1566",
    matchBorder: "#facc15",
    matchOverviewRuler: "#facc15",
    activeMatchBackground: "#fb923c99",
    activeMatchBorder: "#fb923c",
    activeMatchColorOverviewRuler: "#fb923c",
  },
}

const DARK: ITheme = {
  background: "#0b0b0d",
  foreground: "#d4d4d8",
  cursor: "#d4d4d8",
  cursorAccent: "#0b0b0d",
  selectionBackground: "#3f3f46",
  black: "#27272a",
  red: "#f87171",
  green: "#4ade80",
  yellow: "#facc15",
  blue: "#60a5fa",
  magenta: "#c084fc",
  cyan: "#22d3ee",
  white: "#e4e4e7",
  brightBlack: "#71717a",
  brightRed: "#fca5a5",
  brightGreen: "#86efac",
  brightYellow: "#fde047",
  brightBlue: "#93c5fd",
  brightMagenta: "#d8b4fe",
  brightCyan: "#67e8f9",
  brightWhite: "#fafafa",
}

const LIGHT: ITheme = {
  background: "#fafafa",
  foreground: "#18181b",
  cursor: "#18181b",
  cursorAccent: "#fafafa",
  selectionBackground: "#d4d4d8",
  black: "#18181b",
  red: "#dc2626",
  green: "#15803d",
  yellow: "#a16207",
  blue: "#2563eb",
  magenta: "#9333ea",
  cyan: "#0e7490",
  white: "#52525b",
  brightBlack: "#71717a",
  brightRed: "#ef4444",
  brightGreen: "#16a34a",
  brightYellow: "#ca8a04",
  brightBlue: "#3b82f6",
  brightMagenta: "#a855f7",
  brightCyan: "#0891b2",
  brightWhite: "#27272a",
}

function terminalTheme(resolved: "dark" | "light", readOnly: boolean): ITheme {
  const base = resolved === "dark" ? DARK : LIGHT
  // A read-only terminal has no caret to show.
  return readOnly
    ? { ...base, cursor: base.background, cursorAccent: base.background }
    : base
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    // clipboard access can be denied
  }
}

// Programs may ask the terminal to write to the clipboard (OSC 52) but never to
// read it: output must not be able to leak what the user copied.
const writeOnlyClipboard: IClipboardProvider = {
  readText: () => "",
  writeText: (_selection, text) => copyText(text),
}

/**
 * A full terminal (xterm.js): colors, cursor control, scrollback, search, links,
 * copy/paste and resizing. It only displays what is written to it through the
 * handle and reports keystrokes and size; where the bytes come from is up to
 * the caller.
 */
export function Terminal({
  readOnly = false,
  onHandle,
  onData,
  onResize,
  className,
}: TerminalProps) {
  const { resolvedTheme } = useTheme()
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<XTerm | null>(null)
  const searchRef = useRef<SearchAddon | null>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const callbacks = useRef({ onHandle, onData, onResize })
  const themeRef = useRef(resolvedTheme)
  const [searchOpen, setSearchOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<{ index: number; count: number } | null>(
    null
  )

  useEffect(() => {
    callbacks.current = { onHandle, onData, onResize }
    themeRef.current = resolvedTheme
  })

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const term = new XTerm({
      allowProposedApi: true,
      scrollback: 10000,
      fontFamily: FONT_FAMILY,
      fontSize: 12,
      lineHeight: 1.15,
      cursorBlink: !readOnly,
      cursorInactiveStyle: readOnly ? "none" : "outline",
      disableStdin: readOnly,
      macOptionIsMeta: true,
      scrollOnUserInput: true,
      theme: terminalTheme(themeRef.current, readOnly),
    })
    const fit = new FitAddon()
    const search = new SearchAddon()
    term.loadAddon(fit)
    term.loadAddon(search)
    term.loadAddon(
      new WebLinksAddon((_event, uri) => {
        void api.openExternal(uri)
      })
    )
    term.loadAddon(new Unicode11Addon())
    term.loadAddon(new ClipboardAddon(undefined, writeOnlyClipboard))
    term.open(container)
    term.unicode.activeVersion = "11"
    try {
      const webgl = new WebglAddon()
      webgl.onContextLoss(() => webgl.dispose())
      term.loadAddon(webgl)
    } catch {
      // the DOM renderer is used when WebGL is unavailable
    }
    termRef.current = term
    searchRef.current = search

    const searchSub = search.onDidChangeResults((r) =>
      setResults(r ? { index: r.resultIndex, count: r.resultCount } : null)
    )
    const dataSub = readOnly
      ? null
      : term.onData((data) => callbacks.current.onData?.(data))

    term.attachCustomKeyEventHandler((event) => {
      if (event.type !== "keydown") return true
      const mod = event.ctrlKey || event.metaKey
      const key = event.key.toLowerCase()
      if (mod && !event.shiftKey && !event.altKey && key === "f") {
        setSearchOpen(true)
        return false
      }
      if (mod && key === "c" && (event.shiftKey || term.hasSelection())) {
        const selection = term.getSelection()
        if (selection) void copyText(selection)
        return false
      }
      return true
    })

    // Fit to the container, and tell the program once the size settles.
    let frame = 0
    let reportTimer = 0
    let reported = ""
    const report = () => {
      reportTimer = 0
      const size = `${term.cols}x${term.rows}`
      if (size === reported) return
      reported = size
      callbacks.current.onResize?.(term.cols, term.rows)
    }
    const doFit = () => {
      frame = 0
      if (container.clientWidth === 0 || container.clientHeight === 0) return
      try {
        fit.fit()
      } catch {
        return
      }
      if (reported === "") report()
      else {
        window.clearTimeout(reportTimer)
        reportTimer = window.setTimeout(report, 150)
      }
    }
    const observer = new ResizeObserver(() => {
      if (!frame) frame = requestAnimationFrame(doFit)
    })
    observer.observe(container)
    doFit()
    if (!readOnly) term.focus()

    let disposed = false
    callbacks.current.onHandle?.({
      write: (data) => {
        if (!disposed) term.write(data)
      },
      reset: () => {
        if (!disposed) term.reset()
      },
      focus: () => {
        if (!disposed) term.focus()
      },
    })

    return () => {
      disposed = true
      callbacks.current.onHandle?.(null)
      cancelAnimationFrame(frame)
      window.clearTimeout(reportTimer)
      observer.disconnect()
      searchSub.dispose()
      dataSub?.dispose()
      termRef.current = null
      searchRef.current = null
      term.dispose()
    }
  }, [readOnly])

  useEffect(() => {
    if (termRef.current) {
      termRef.current.options.theme = terminalTheme(resolvedTheme, readOnly)
    }
  }, [resolvedTheme, readOnly])

  useEffect(() => {
    if (searchOpen) searchInputRef.current?.select()
  }, [searchOpen])

  function find(direction: "next" | "previous", incremental = false) {
    const search = searchRef.current
    if (!search) return
    if (!query) {
      search.clearDecorations()
      setResults(null)
      return
    }
    const options = { ...SEARCH_OPTIONS, incremental }
    if (direction === "next") search.findNext(query, options)
    else search.findPrevious(query, options)
  }

  function closeSearch() {
    searchRef.current?.clearDecorations()
    setResults(null)
    setSearchOpen(false)
    termRef.current?.focus()
  }

  const background = terminalTheme(resolvedTheme, readOnly).background

  return (
    <div
      className={cn(
        "group/terminal relative min-h-0 overflow-hidden rounded-lg border",
        className
      )}
      style={{ backgroundColor: background }}
    >
      <div ref={containerRef} className="absolute inset-2" />

      <div
        className={cn(
          "absolute top-1.5 right-3 z-10 flex items-center gap-0.5 rounded-md border bg-popover/90 p-0.5 shadow-sm backdrop-blur transition-opacity",
          searchOpen
            ? "opacity-100"
            : "opacity-0 group-focus-within/terminal:opacity-100 group-hover/terminal:opacity-100"
        )}
      >
        {searchOpen ? (
          <>
            <Input
              ref={searchInputRef}
              value={query}
              placeholder="Find in output"
              aria-label="Find in output"
              className="h-6 w-40 text-xs"
              onChange={(e) => {
                setQuery(e.target.value)
                if (!e.target.value) {
                  searchRef.current?.clearDecorations()
                  setResults(null)
                } else {
                  searchRef.current?.findNext(e.target.value, {
                    ...SEARCH_OPTIONS,
                    incremental: true,
                  })
                }
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault()
                  find(e.shiftKey ? "previous" : "next")
                } else if (e.key === "Escape") {
                  e.preventDefault()
                  e.stopPropagation()
                  closeSearch()
                }
              }}
            />
            <span className="min-w-10 px-1 text-center text-[11px] text-muted-foreground tabular-nums">
              {query && results
                ? results.count > 0
                  ? `${results.index + 1}/${results.count}`
                  : "0"
                : ""}
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Previous match (Shift+Enter)"
              onClick={() => find("previous")}
            >
              <ChevronUpIcon />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Next match (Enter)"
              onClick={() => find("next")}
            >
              <ChevronDownIcon />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Close (Esc)"
              onClick={closeSearch}
            >
              <XIcon />
            </Button>
          </>
        ) : (
          <>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Find (Ctrl+F)"
              onClick={() => setSearchOpen(true)}
            >
              <SearchIcon />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Scroll to bottom"
              onClick={() => termRef.current?.scrollToBottom()}
            >
              <ArrowDownToLineIcon />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              title="Clear the screen (the command keeps running)"
              onClick={() => termRef.current?.clear()}
            >
              <EraserIcon />
            </Button>
          </>
        )}
      </div>
    </div>
  )
}
