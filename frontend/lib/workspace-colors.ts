// Colors a workspace can be given, stored as "#rrggbb" (no color = null). The
// sidebar tints the workspace with the color and draws the line beside its apps
// in it; Go only accepts a hex value (see normalizeWorkspaceColor in bind.go).
export const WORKSPACE_COLORS = [
  { name: "Red", value: "#ef4444" },
  { name: "Orange", value: "#f97316" },
  { name: "Amber", value: "#f59e0b" },
  { name: "Green", value: "#22c55e" },
  { name: "Teal", value: "#14b8a6" },
  { name: "Blue", value: "#3b82f6" },
  { name: "Violet", value: "#8b5cf6" },
  { name: "Pink", value: "#ec4899" },
  { name: "Gray", value: "#64748b" },
] as const

// The color at the given strength (percent) over whatever is behind it, so a
// low value stays a soft tint in both the light and the dark theme.
export function workspaceTint(color: string, percent = 5): string {
  return `color-mix(in oklab, ${color} ${percent}%, transparent)`
}
