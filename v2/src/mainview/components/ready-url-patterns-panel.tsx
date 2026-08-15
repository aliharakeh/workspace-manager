import { useDeferredValue, useEffect, useMemo, useState } from "react"
import {
  Link2Icon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { ReadyUrlPattern } from "@/lib/types"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"

type ReadyUrlPatternsPanelProps = {
  active: boolean
}

type PatternRowProps = {
  item: ReadyUrlPattern
  onSaved: (item: ReadyUrlPattern) => void
  onRequestDelete: (item: ReadyUrlPattern) => void
  deleting: boolean
}

function matchesQuery(item: ReadyUrlPattern, query: string): boolean {
  if (!query) return true
  const q = query.toLowerCase()
  return (
    item.label.toLowerCase().includes(q) ||
    item.pattern.toLowerCase().includes(q) ||
    (item.key?.toLowerCase().includes(q) ?? false)
  )
}

function PatternRow({
  item,
  onSaved,
  onRequestDelete,
  deleting,
}: PatternRowProps) {
  const [label, setLabel] = useState(item.label)
  const [pattern, setPattern] = useState(item.pattern)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setLabel(item.label)
    setPattern(item.pattern)
  }, [item.id, item.label, item.pattern])

  const dirty =
    label.trim() !== item.label || pattern.trim() !== item.pattern

  async function handleSave() {
    const nextLabel = label.trim()
    const nextPattern = pattern.trim()
    if (!nextLabel) {
      toast.error("Label is required")
      return
    }
    if (!nextPattern) {
      toast.error("Pattern is required")
      return
    }
    if (!dirty) return

    setSaving(true)
    try {
      const updated = await api.readyUrlPatterns.update(item.id, {
        label: nextLabel,
        pattern: nextPattern,
      })
      onSaved(updated)
      toast.success("Pattern updated")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update")
    } finally {
      setSaving(false)
    }
  }

  return (
    <AccordionItem value={String(item.id)} className="rounded-lg border px-3">
      <AccordionTrigger className="hover:no-underline">
        <div className="flex min-w-0 flex-1 flex-col items-start gap-1 pr-2">
          <div className="flex min-w-0 max-w-full items-center gap-2">
            <span className="truncate font-medium">{item.label}</span>
            {item.key ? (
              <Badge variant="secondary" className="font-mono">
                {item.key}
              </Badge>
            ) : (
              <Badge variant="outline">Custom</Badge>
            )}
            {dirty ? (
              <Badge variant="outline" className="text-muted-foreground">
                Unsaved
              </Badge>
            ) : null}
          </div>
          <span className="w-full truncate font-mono text-xs font-normal text-muted-foreground">
            {item.pattern}
          </span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="flex flex-col gap-3 pb-3">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor={`pattern-label-${item.id}`}>Label</FieldLabel>
            <Input
              id={`pattern-label-${item.id}`}
              value={label}
              disabled={saving || deleting}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Label"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor={`pattern-source-${item.id}`}>
              Pattern
            </FieldLabel>
            <Textarea
              id={`pattern-source-${item.id}`}
              className="min-h-20 font-mono text-xs"
              value={pattern}
              disabled={saving || deleting}
              onChange={(e) => setPattern(e.target.value)}
              spellCheck={false}
            />
            <FieldDescription>
              Named group <code className="font-mono text-xs">url</code> or{" "}
              <code className="font-mono text-xs">port</code> required.
            </FieldDescription>
          </Field>
        </FieldGroup>
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={!dirty || saving || deleting}
            onClick={() => void handleSave()}
          >
            <SaveIcon data-icon="inline-start" />
            {saving ? "Saving…" : "Save"}
          </Button>
          <Button
            variant="destructive"
            disabled={deleting || saving}
            onClick={() => onRequestDelete(item)}
          >
            <Trash2Icon data-icon="inline-start" />
            Delete
          </Button>
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

export function ReadyUrlPatternsPanel({ active }: ReadyUrlPatternsPanelProps) {
  const [patterns, setPatterns] = useState<ReadyUrlPattern[]>([])
  const [label, setLabel] = useState("")
  const [pattern, setPattern] = useState("")
  const [query, setQuery] = useState("")
  const deferredQuery = useDeferredValue(query)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [openIds, setOpenIds] = useState<string[]>([])
  const [pendingDelete, setPendingDelete] = useState<ReadyUrlPattern | null>(
    null
  )
  const [deleting, setDeleting] = useState(false)

  async function load() {
    setLoading(true)
    try {
      setPatterns(await api.readyUrlPatterns.list())
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load patterns"
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (active) void load()
    else {
      setQuery("")
      setShowAdd(false)
      setOpenIds([])
      setPendingDelete(null)
    }
  }, [active])

  const filtered = useMemo(
    () => patterns.filter((item) => matchesQuery(item, deferredQuery.trim())),
    [patterns, deferredQuery]
  )

  async function handleConfirmDelete() {
    if (!pendingDelete) return
    setDeleting(true)
    try {
      await api.readyUrlPatterns.delete(pendingDelete.id)
      setPatterns((prev) => prev.filter((p) => p.id !== pendingDelete.id))
      setOpenIds((prev) =>
        prev.filter((v) => v !== String(pendingDelete.id))
      )
      setPendingDelete(null)
      toast.success("Pattern deleted")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete")
    } finally {
      setDeleting(false)
    }
  }

  async function handleAdd() {
    const nextLabel = label.trim()
    const nextPattern = pattern.trim()
    if (!nextLabel) {
      toast.error("Label is required")
      return
    }
    if (!nextPattern) {
      toast.error("Pattern is required")
      return
    }
    setAdding(true)
    try {
      const created = await api.readyUrlPatterns.create({
        label: nextLabel,
        pattern: nextPattern,
      })
      setPatterns((prev) => [...prev, created])
      setLabel("")
      setPattern("")
      setShowAdd(false)
      setOpenIds([String(created.id)])
      setQuery("")
      toast.success("Pattern added")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add")
    } finally {
      setAdding(false)
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    )
  }

  return (
    <>
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-medium">Log URL patterns</p>
          <Badge variant="secondary">{patterns.length}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">
          Detect ready URLs from process logs. First matching pattern wins.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <InputGroup className="flex-1">
          <InputGroupAddon align="inline-start">
            <SearchIcon />
          </InputGroupAddon>
          <InputGroupInput
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search label, key, or regex…"
            aria-label="Search patterns"
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
        <Button
          variant={showAdd ? "secondary" : "default"}
          onClick={() => setShowAdd((v) => !v)}
        >
          <PlusIcon data-icon="inline-start" />
          {showAdd ? "Cancel" : "Add pattern"}
        </Button>
      </div>

      {showAdd ? (
        <FieldGroup className="rounded-lg border bg-muted/30 p-3">
          <Field>
            <FieldLabel htmlFor="pattern-label">Label</FieldLabel>
            <Input
              id="pattern-label"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Vite"
              autoFocus
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="pattern-source">Pattern</FieldLabel>
            <Textarea
              id="pattern-source"
              value={pattern}
              onChange={(e) => setPattern(e.target.value)}
              placeholder={String.raw`\bLocal:\s+(?<url>https?:\/\/\S+)`}
              className="min-h-16 font-mono text-xs"
              spellCheck={false}
            />
            <FieldDescription>
              Include a named group{" "}
              <code className="font-mono text-xs">url</code> or{" "}
              <code className="font-mono text-xs">port</code>.
            </FieldDescription>
          </Field>
          <div className="flex justify-end">
            <Button disabled={adding} onClick={() => void handleAdd()}>
              <PlusIcon data-icon="inline-start" />
              {adding ? "Adding…" : "Add pattern"}
            </Button>
          </div>
        </FieldGroup>
      ) : null}

      {patterns.length === 0 ? (
        <Empty className="border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Link2Icon />
            </EmptyMedia>
            <EmptyTitle>No URL patterns</EmptyTitle>
            <EmptyDescription>
              Add a regex with a named <code>url</code> or <code>port</code>{" "}
              group to detect server addresses in logs.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : filtered.length === 0 ? (
        <Empty className="border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <SearchIcon />
            </EmptyMedia>
            <EmptyTitle>No matches</EmptyTitle>
            <EmptyDescription>
              Nothing matches “{deferredQuery.trim()}”. Try another search.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ScrollArea className="min-h-0 flex-1">
          <Accordion
            value={openIds}
            onValueChange={setOpenIds}
            keepMounted
            className="flex flex-col gap-2 pr-3"
          >
            {filtered.map((item) => (
              <PatternRow
                key={item.id}
                item={item}
                deleting={deleting && pendingDelete?.id === item.id}
                onSaved={(updated) =>
                  setPatterns((prev) =>
                    prev.map((p) => (p.id === updated.id ? updated : p))
                  )
                }
                onRequestDelete={setPendingDelete}
              />
            ))}
          </Accordion>
        </ScrollArea>
      )}

      {patterns.length > 0 && deferredQuery.trim() ? (
        <p className="text-xs text-muted-foreground">
          Showing {filtered.length} of {patterns.length}
        </p>
      ) : null}
    </div>

    <AlertDialog
      open={!!pendingDelete}
      onOpenChange={(next) => {
        if (!next && !deleting) setPendingDelete(null)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete pattern?</AlertDialogTitle>
          <AlertDialogDescription>
            This removes the pattern from ready-URL detection. This cannot be
            undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {pendingDelete ? (
          <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 rounded-lg border bg-muted/40 px-3 py-2.5 text-sm">
            <dt className="text-muted-foreground">Label</dt>
            <dd className="truncate font-medium">{pendingDelete.label}</dd>
            {pendingDelete.key ? (
              <>
                <dt className="text-muted-foreground">Key</dt>
                <dd className="truncate font-mono text-xs">
                  {pendingDelete.key}
                </dd>
              </>
            ) : null}
            <dt className="text-muted-foreground">Pattern</dt>
            <dd className="truncate font-mono text-xs">
              {pendingDelete.pattern}
            </dd>
          </dl>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={(e) => {
              e.preventDefault()
              void handleConfirmDelete()
            }}
          >
            {deleting ? "Deleting…" : "Delete pattern"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    </>
  )
}
