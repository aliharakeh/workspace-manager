import SimpleCodeEditor from "react-simple-code-editor"
import type { ComponentType } from "react"
import Prism from "prismjs"
import "prismjs/components/prism-markup"
import "prismjs/components/prism-markup-templating"
import "prismjs/components/prism-handlebars"
import "prismjs/components/prism-javascript"
import "prismjs/components/prism-jsx"
import "prismjs/components/prism-typescript"
import "prismjs/components/prism-tsx"
import "prismjs/components/prism-json"
import "prismjs/components/prism-css"
import "prismjs/components/prism-scss"
import "prismjs/components/prism-java"
import "prismjs/components/prism-python"
import "prismjs/components/prism-yaml"
import "prismjs/components/prism-markdown"
import "prismjs/components/prism-bash"
import "prismjs/components/prism-sql"
import "prismjs/components/prism-ruby"
import "prismjs/components/prism-go"
import "prismjs/components/prism-rust"
import "prismjs/components/prism-php"
import { cn } from "@/lib/utils"

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
  /** Path of the template's target file; drives syntax highlighting. */
  filePath: string
  className?: string
}

const LANGUAGE_BY_EXTENSION: Record<string, string> = {
  ts: "typescript",
  tsx: "tsx",
  js: "javascript",
  jsx: "jsx",
  mjs: "javascript",
  cjs: "javascript",
  json: "json",
  html: "markup",
  htm: "markup",
  xml: "markup",
  vue: "markup",
  svelte: "markup",
  hbs: "handlebars",
  handlebars: "handlebars",
  css: "css",
  scss: "scss",
  sass: "scss",
  less: "css",
  java: "java",
  py: "python",
  yaml: "yaml",
  yml: "yaml",
  md: "markdown",
  markdown: "markdown",
  sh: "bash",
  bash: "bash",
  zsh: "bash",
  sql: "sql",
  rb: "ruby",
  go: "go",
  rs: "rust",
  php: "php",
}

/** Fall back to Handlebars so {{var}} syntax still stands out. */
function languageForPath(filePath: string): string {
  const fileName = filePath.split("/").pop() ?? filePath
  const dot = fileName.lastIndexOf(".")
  if (dot === -1) return "handlebars"
  return LANGUAGE_BY_EXTENSION[fileName.slice(dot + 1).toLowerCase()] ?? "handlebars"
}

function highlightTemplate(code: string, filePath: string): string {
  const language = languageForPath(filePath)
  const grammar =
    Prism.languages[language] ??
    Prism.languages.handlebars ??
    Prism.languages.markup ??
    Prism.languages.plain
  if (!grammar) return code
  try {
    return Prism.highlight(code, grammar, language)
  } catch {
    return code
  }
}

export function TemplateEditor({
  value,
  onChange,
  filePath,
  className,
}: TemplateEditorProps) {
  return (
    <div
      className={cn(
        "max-h-80 min-h-48 overflow-auto rounded-lg border font-mono text-sm",
        className
      )}
      style={{
        backgroundColor: "var(--editor-bg)",
        color: "var(--editor-fg)",
      }}
    >
      <Editor
        value={value}
        onValueChange={onChange}
        highlight={(code) => highlightTemplate(code, filePath)}
        padding={12}
        style={{
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
          fontSize: 13,
          minHeight: 192,
        }}
      />
    </div>
  )
}
