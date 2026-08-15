export const SEARCH_SHORTCUT_KEY = "shortcut.search"
export const THEME_SHORTCUT_KEY = "shortcut.theme"

export const DEFAULT_SHORTCUTS: Record<string, string> = {
  [SEARCH_SHORTCUT_KEY]: "ctrl+p",
  [THEME_SHORTCUT_KEY]: "d",
}

export type ShortcutBinding = {
  ctrl: boolean
  alt: boolean
  shift: boolean
  meta: boolean
  /** Normalized key: lowercase letter, "space", "f1", "arrowup", … */
  key: string
}

const MODIFIER_KEYS = new Set(["control", "alt", "shift", "meta"])

/**
 * Set while the settings UI is capturing a new shortcut, so global shortcut
 * listeners (e.g. the search palette) ignore keys pressed during recording.
 */
let shortcutRecorderActive = false
export function setShortcutRecorderActive(active: boolean) {
  shortcutRecorderActive = active
}
export function isShortcutRecorderActive() {
  return shortcutRecorderActive
}

function normalizeKey(key: string): string {
  if (key === " ") return "space"
  if (key.length === 1) return key.toLowerCase()
  return key.toLowerCase()
}

/** Parse a stored shortcut like "ctrl+p" into its parts. Null if malformed. */
export function parseShortcut(raw: string | null | undefined): ShortcutBinding | null {
  if (!raw) return null
  const binding: ShortcutBinding = {
    ctrl: false,
    alt: false,
    shift: false,
    meta: false,
    key: "",
  }
  for (const part of raw.toLowerCase().split("+")) {
    if (part === "ctrl" || part === "control") binding.ctrl = true
    else if (part === "alt" || part === "option") binding.alt = true
    else if (part === "shift") binding.shift = true
    else if (part === "meta" || part === "cmd" || part === "command") binding.meta = true
    else if (part === "super" || part === "win" || part === "windows") binding.meta = true
    else if (part) binding.key = part
  }
  if (!binding.key) return null
  return binding
}

/** Shortcut is usable: at least one modifier plus a non-modifier key. */
export function isValidShortcut(raw: string): boolean {
  const binding = parseShortcut(raw)
  if (!binding) return false
  return binding.ctrl || binding.alt || binding.shift || binding.meta
}

/** Normalize a KeyboardEvent into a stored shortcut string, or null for invalid combos. */
export function eventToShortcut(event: KeyboardEvent): string | null {
  const key = normalizeKey(event.key)
  if (MODIFIER_KEYS.has(key)) return null
  const parts: string[] = []
  if (event.ctrlKey) parts.push("ctrl")
  if (event.metaKey) parts.push("meta")
  if (event.altKey) parts.push("alt")
  if (event.shiftKey) parts.push("shift")
  if (parts.length === 0) return null
  parts.push(key)
  return parts.join("+")
}

/** Human-readable key chips, e.g. ["Ctrl", "P"] or ["⌘", "P"] on macOS. */
export function shortcutParts(raw: string | null | undefined): string[] {
  const binding = parseShortcut(raw)
  if (!binding) return []
  const isMac =
    typeof navigator !== "undefined" &&
    /mac|iphone|ipad|ipod/i.test(navigator.platform ?? "")
  const displayKey =
    binding.key.length === 1
      ? binding.key.toUpperCase()
      : binding.key.charAt(0).toUpperCase() + binding.key.slice(1)

  const parts: string[] = []
  if (isMac) {
    if (binding.meta) parts.push("⌘")
    if (binding.ctrl) parts.push("⌃")
    if (binding.alt) parts.push("⌥")
    if (binding.shift) parts.push("⇧")
  } else {
    if (binding.ctrl) parts.push("Ctrl")
    if (binding.meta) parts.push("Meta")
    if (binding.alt) parts.push("Alt")
    if (binding.shift) parts.push("Shift")
  }
  parts.push(displayKey)
  return parts
}

export function formatShortcut(raw: string | null | undefined): string {
  return shortcutParts(raw).join(" ")
}

export function matchesShortcut(
  raw: string | null | undefined,
  event: KeyboardEvent
): boolean {
  if (event.repeat) return false
  const binding = parseShortcut(raw)
  if (!binding) return false
  if (event.ctrlKey !== binding.ctrl) return false
  if (event.metaKey !== binding.meta) return false
  if (event.altKey !== binding.alt) return false
  if (event.shiftKey !== binding.shift) return false
  return normalizeKey(event.key) === binding.key
}
