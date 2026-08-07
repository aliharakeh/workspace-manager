import SimpleCodeEditor from "react-simple-code-editor"
import type { ComponentType } from "react"
import Prism from "prismjs"
import "prismjs/components/prism-markup"
import "prismjs/components/prism-markup-templating"
import "prismjs/components/prism-handlebars"
import "prismjs/themes/prism-tomorrow.css"

type EditorProps = {
  value: string
  onValueChange: (value: string) => void
  highlight: (value: string) => string
  padding?: number
  style?: React.CSSProperties
}

function resolveDefaultExport<T>(mod: unknown): T {
  if (typeof mod === "function") return mod as T
  if (mod && typeof mod === "object") {
    // React forwardRef / memo exotic components
    if ("$$typeof" in mod) return mod as T
    const nested = (mod as { default?: unknown }).default
    if (nested && nested !== mod) return resolveDefaultExport<T>(nested)
  }
  throw new Error("react-simple-code-editor: invalid module export")
}

const Editor = resolveDefaultExport<ComponentType<EditorProps>>(SimpleCodeEditor)

type TemplateEditorProps = {
  value: string
  onChange: (value: string) => void
  className?: string
}

function highlightTemplate(code: string) {
  try {
    const grammar =
      Prism.languages.handlebars ??
      Prism.languages.markup ??
      Prism.languages.plain
    if (!grammar) return code
    return Prism.highlight(
      code,
      grammar,
      Prism.languages.handlebars ? "handlebars" : "markup"
    )
  } catch {
    return code
  }
}

export function TemplateEditor({
  value,
  onChange,
  className,
}: TemplateEditorProps) {
  return (
    <div
      className={
        className ??
        "max-h-80 min-h-48 overflow-auto rounded-lg border bg-[#2d2d2d] font-mono text-sm"
      }
    >
      <Editor
        value={value}
        onValueChange={onChange}
        highlight={highlightTemplate}
        padding={12}
        style={{
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
          fontSize: 13,
          minHeight: 192,
          color: "#ccc",
        }}
      />
    </div>
  )
}
