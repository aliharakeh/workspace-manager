import { useDeferredValue, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import {
  EyeIcon,
  FileCodeIcon,
  FolderOpenIcon,
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import { renderTemplatePreview } from "@/lib/template-preview"
import type { Template } from "@/lib/types"
import { TemplateEditor } from "@/components/template-editor"
import { Button } from "@/components/ui/button"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"

type TemplatesPanelProps = {
  appId: number
}

type TemplateItemProps = {
  appId: number
  template: Template
  env: Record<string, string>
  onUpdated: (template: Template) => void
  onDeleted: (id: number) => void
}

function fileName(path: string) {
  const parts = path.replace(/\\/g, "/").split("/")
  return parts[parts.length - 1] || path
}

function TemplateItem({
  appId,
  template,
  env,
  onUpdated,
  onDeleted,
}: TemplateItemProps) {
  const [content, setContent] = useState(template.content)
  const [mode, setMode] = useState<"edit" | "preview">("edit")
  const [saving, setSaving] = useState(false)
  const [reloading, setReloading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const dirty = content !== template.content

  useEffect(() => {
    setContent(template.content)
  }, [template.id, template.content])

  const preview = useMemo(
    () => renderTemplatePreview(content, env),
    [content, env]
  )

  async function handleSave() {
    setSaving(true)
    try {
      const updated = await api.templates.update(template.id, { content })
      onUpdated(updated)
      toast.success("Template saved")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  async function handleReloadFromFile() {
    setReloading(true)
    try {
      const file = await api.fs.readAppFile(appId, template.file_path)
      setContent(file.content)
      toast.success("Loaded current file contents")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to read file")
    } finally {
      setReloading(false)
    }
  }

  async function handleDelete() {
    setDeleting(true)
    try {
      await api.templates.delete(template.id)
      onDeleted(template.id)
      toast.success("Template deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setDeleting(false)
    }
  }

  return (
    <AccordionItem
      value={String(template.id)}
      className="rounded-lg border px-3"
    >
      <AccordionTrigger className="hover:no-underline">
        <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5 pr-2">
          <span className="truncate font-medium">
            {fileName(template.file_path)}
          </span>
          <span className="truncate font-mono text-xs font-normal text-muted-foreground">
            {template.file_path}
          </span>
        </div>
        {dirty ? (
          <span className="mr-2 rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
            Unsaved
          </span>
        ) : null}
      </AccordionTrigger>
      <AccordionContent className="flex flex-col gap-3 pb-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground">
            {mode === "edit"
              ? "Edit the template source. Use Handlebars like {{API_URL}}."
              : "Preview with current env vars applied (as on Run)."}
          </p>
          <ToggleGroup
            value={[mode]}
            onValueChange={(v) => {
              if (v[0] === "edit" || v[0] === "preview") setMode(v[0])
            }}
            variant="outline"
            size="sm"
            spacing={0}
          >
            <ToggleGroupItem value="edit" aria-label="Edit template">
              <PencilIcon />
              Edit
            </ToggleGroupItem>
            <ToggleGroupItem
              value="preview"
              aria-label="Preview rendered template"
            >
              <EyeIcon />
              Preview
            </ToggleGroupItem>
          </ToggleGroup>
        </div>

        {mode === "edit" ? (
          <TemplateEditor
            value={content}
            onChange={setContent}
            filePath={template.file_path}
          />
        ) : preview.ok ? (
          <pre className="max-h-80 min-h-48 overflow-auto rounded-lg border bg-muted/40 p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap">
            {preview.text.length > 0 ? preview.text : "(empty)"}
          </pre>
        ) : (
          <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
            {preview.error}
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button disabled={saving || !dirty} onClick={() => void handleSave()}>
            {saving ? "Saving…" : "Save"}
          </Button>
          <Button
            variant="outline"
            disabled={reloading}
            onClick={() => void handleReloadFromFile()}
          >
            <RefreshCwIcon data-icon="inline-start" />
            {reloading ? "Loading…" : "Reload from file"}
          </Button>
          <Button
            variant="destructive"
            disabled={deleting}
            onClick={() => void handleDelete()}
          >
            <Trash2Icon data-icon="inline-start" />
            {deleting ? "Deleting…" : "Delete"}
          </Button>
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

export function TemplatesPanel({ appId }: TemplatesPanelProps) {
  const [templates, setTemplates] = useState<Template[]>([])
  const [env, setEnv] = useState<Record<string, string>>({})
  const [query, setQuery] = useState("")
  const deferredQuery = useDeferredValue(query)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [openIds, setOpenIds] = useState<string[]>([])

  const filtered = useMemo(() => {
    const q = deferredQuery.trim().toLowerCase()
    if (!q) return templates
    return templates.filter(
      (t) =>
        t.file_path.toLowerCase().includes(q) ||
        fileName(t.file_path).toLowerCase().includes(q)
    )
  }, [templates, deferredQuery])

  async function load() {
    setLoading(true)
    try {
      const [list, vars] = await Promise.all([
        api.templates.list(appId),
        api.envVars.list(appId),
      ])
      setTemplates(list)
      setEnv(Object.fromEntries(vars.map((v) => [v.key, v.value])))
      setOpenIds((current) =>
        current.filter((id) => list.some((t) => String(t.id) === id))
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load templates"
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [appId])

  // Refresh env when returning to this panel / periodically while open would be nice,
  // but re-fetch when accordion opens is enough if user edited env in another tab.
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const vars = await api.envVars.list(appId)
        if (!cancelled) {
          setEnv(Object.fromEntries(vars.map((v) => [v.key, v.value])))
        }
      } catch {
        // ignore background refresh errors
      }
    })()
    return () => {
      cancelled = true
    }
  }, [appId, openIds.join(",")])

  async function handleAdd() {
    setAdding(true)
    try {
      const picked = await api.fs.pickAppFile(appId)
      if (picked.cancelled || !picked.relative_path) return

      const created = await api.templates.create(appId, {
        file_path: picked.relative_path,
        content: picked.content ?? "",
      })
      setTemplates((prev) =>
        [...prev, created].sort((a, b) =>
          a.file_path.localeCompare(b.file_path)
        )
      )
      setOpenIds([String(created.id)])
      toast.success("Template created from file")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add template")
    } finally {
      setAdding(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading…</p>
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-muted-foreground">
          Each template targets a project file. Content starts as a copy of that
          file — add variables, then save.
        </p>
        <Button disabled={adding} onClick={() => void handleAdd()}>
          {adding ? (
            "Browsing…"
          ) : (
            <>
              <PlusIcon data-icon="inline-start" />
              Add template
            </>
          )}
        </Button>
      </div>

      {templates.length > 0 ? (
        <div className="flex items-center gap-2">
          <InputGroup className="max-w-sm flex-1">
            <InputGroupAddon align="inline-start">
              <SearchIcon />
            </InputGroupAddon>
            <InputGroupInput
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search file path…"
              aria-label="Search templates"
            />
            {query ? (
              <InputGroupAddon align="inline-end">
                <InputGroupButton
                  size="icon-xs"
                  aria-label="Clear search"
                  onClick={() => setQuery("")}
                >
                  <XIcon />
                </InputGroupButton>
              </InputGroupAddon>
            ) : null}
          </InputGroup>
          <span className="text-xs text-muted-foreground">
            {filtered.length} of {templates.length}
          </span>
        </div>
      ) : null}

      {templates.length === 0 ? (
        <Empty className="border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <FileCodeIcon />
            </EmptyMedia>
            <EmptyTitle>No templates</EmptyTitle>
            <EmptyDescription>
              Browse a file from the project. We’ll load its contents so you can
              turn values into {"{{env}}"} variables.
            </EmptyDescription>
          </EmptyHeader>
          <Button
            variant="outline"
            disabled={adding}
            onClick={() => void handleAdd()}
          >
            <FolderOpenIcon data-icon="inline-start" />
            {adding ? "Browsing…" : "Choose a file"}
          </Button>
        </Empty>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No templates match “{query.trim()}”.
        </p>
      ) : (
        <Accordion
          value={openIds}
          onValueChange={setOpenIds}
          keepMounted
          className="gap-2"
        >
          {filtered.map((template) => (
            <TemplateItem
              key={template.id}
              appId={appId}
              template={template}
              env={env}
              onUpdated={(updated) =>
                setTemplates((prev) =>
                  prev.map((t) => (t.id === updated.id ? updated : t))
                )
              }
              onDeleted={(id) => {
                setTemplates((prev) => prev.filter((t) => t.id !== id))
                setOpenIds((prev) => prev.filter((v) => v !== String(id)))
              }}
            />
          ))}
        </Accordion>
      )}
    </div>
  )
}
