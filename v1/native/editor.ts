/** Open a path in the local editor (`$VISUAL` / `$EDITOR` / bunfig). */

export function openInEditor(path: string) {
  Bun.openInEditor(path)
}
