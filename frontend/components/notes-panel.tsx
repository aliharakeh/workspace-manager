import { useEffect, useId, useState } from "react"
import {
  PencilIcon,
  PlusIcon,
  StickyNoteIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { Note, NoteItem, NoteSaveInput } from "@/lib/types"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

function message(err: unknown) {
  return err instanceof Error ? err.message : String(err)
}

function NoteImage({ note }: { note: Note }) {
  const [src, setSrc] = useState("")
  const [error, setError] = useState("")
  useEffect(() => {
    let current = true
    api.notes.image(note.app_id, note.id).then(
      (data) => {
        if (current) setSrc(data)
      },
      (err) => {
        if (current) setError(message(err))
      }
    )
    return () => {
      current = false
    }
  }, [note.app_id, note.id, note.image_path])
  if (error)
    return (
      <p role="alert" className="p-3 text-sm text-destructive">
        Image unavailable: {error}
      </p>
    )
  return (
    <img
      src={src || undefined}
      alt={`Attachment for ${note.title}`}
      loading="lazy"
      className="aspect-video w-full object-contain"
    />
  )
}

function NoteEditor({
  appId,
  note,
  onClose,
  onSaved,
}: {
  appId: number
  note: Note | null
  onClose: () => void
  onSaved: (note: Note) => void
}) {
  const id = useId()
  const [title, setTitle] = useState(note?.title ?? "")
  const [content, setContent] = useState(note?.content ?? "")
  const [items, setItems] = useState(() =>
    (note?.items ?? []).map((item) => ({ ...item, key: crypto.randomUUID() }))
  )
  const [imageData, setImageData] = useState<string | undefined>()
  const [preview, setPreview] = useState("")
  const [reading, setReading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const hasImage = imageData === undefined ? !!note?.image_path : !!imageData

  async function chooseImage(file: File | undefined) {
    if (!file) return
    setError("")
    if (file.size > 5 * 1024 * 1024) {
      setError("Choose an image up to 5 MB.")
      return
    }
    setReading(true)
    try {
      const url = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(String(reader.result))
        reader.onerror = () => reject(new Error("Could not read this image."))
        reader.readAsDataURL(file)
      })
      setImageData(url.slice(url.indexOf(",") + 1))
      setPreview(url)
    } catch (err) {
      setError(message(err))
    } finally {
      setReading(false)
    }
  }

  async function save(event: React.FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError("")
    const body: NoteSaveInput = {
      title,
      content,
      items: items.map(({ text, done }) => ({ text, done })),
      ...(imageData === undefined ? {} : { image_data: imageData }),
    }
    try {
      onSaved(await api.notes.save(appId, note?.id ?? 0, body))
      toast.success("Note saved")
      onClose()
    } catch (err) {
      setError(message(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !saving && !reading) onClose()
      }}
    >
      <DialogContent
        className="max-h-[90dvh] overflow-y-auto sm:max-w-xl"
        showCloseButton={!saving && !reading}
      >
        <DialogHeader>
          <DialogTitle>{note ? "Edit note" : "New note"}</DialogTitle>
          <DialogDescription>
            Add text, an image, a checklist, or a mix of all three.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="flex flex-col gap-5">
          <FieldSet disabled={saving || reading}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor={`${id}-title`}>Title</FieldLabel>
                <Input
                  id={`${id}-title`}
                  autoFocus
                  required
                  maxLength={300}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Give this note a title"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`${id}-content`}>Content</FieldLabel>
                <Textarea
                  id={`${id}-content`}
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Ideas, context, links…"
                  rows={5}
                  maxLength={100000}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`${id}-image`}>
                  Image (optional)
                </FieldLabel>
                <Input
                  id={`${id}-image`}
                  type="file"
                  accept="image/png,image/jpeg,image/gif"
                  onChange={(e) => {
                    void chooseImage(e.target.files?.[0])
                    e.target.value = ""
                  }}
                />
                <p className="text-xs text-muted-foreground">
                  PNG, JPEG or GIF, up to 5 MB. A copy is saved with the app.
                </p>
                {hasImage && (
                  <>
                    {preview ? (
                      <img
                        src={preview}
                        alt="Selected attachment"
                        className="aspect-video w-full object-contain"
                      />
                    ) : (
                      note && <NoteImage key={note.image_path} note={note} />
                    )}
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setImageData("")
                        setPreview("")
                      }}
                    >
                      <XIcon data-icon="inline-start" />
                      Remove image
                    </Button>
                  </>
                )}
              </Field>
              <FieldSet>
                <FieldLegend>Checklist (optional)</FieldLegend>
                <FieldGroup>
                  {items.map((item, index) => (
                    <Field orientation="horizontal" key={item.key}>
                      <Checkbox
                        aria-label={`Item ${index + 1} completed`}
                        checked={item.done}
                        onCheckedChange={(done) =>
                          setItems(
                            items.map((entry) =>
                              entry.key === item.key
                                ? { ...entry, done }
                                : entry
                            )
                          )
                        }
                      />
                      <Input
                        aria-label={`Checklist item ${index + 1}`}
                        required
                        maxLength={2000}
                        className={cn(
                          "min-w-0 flex-1",
                          item.done && "line-through"
                        )}
                        value={item.text}
                        onChange={(e) =>
                          setItems(
                            items.map((entry) =>
                              entry.key === item.key
                                ? { ...entry, text: e.target.value }
                                : entry
                            )
                          )
                        }
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Remove item ${index + 1}`}
                        onClick={() =>
                          setItems(
                            items.filter((entry) => entry.key !== item.key)
                          )
                        }
                      >
                        <XIcon />
                      </Button>
                    </Field>
                  ))}
                </FieldGroup>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={items.length >= 500}
                  onClick={() =>
                    setItems([
                      ...items,
                      { text: "", done: false, key: crypto.randomUUID() },
                    ])
                  }
                >
                  <PlusIcon data-icon="inline-start" />
                  Add item
                </Button>
              </FieldSet>
            </FieldGroup>
          </FieldSet>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving || reading}
              onClick={onClose}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={saving || reading || !title.trim()}>
              {saving ? "Saving…" : reading ? "Reading image…" : "Save note"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function NotesPanel({
  appId,
  dataEpoch,
}: {
  appId: number
  dataEpoch: number
}) {
  const [notes, setNotes] = useState<Note[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [reload, setReload] = useState(0)
  const [editor, setEditor] = useState<{ note: Note | null } | null>(null)
  const [deleting, setDeleting] = useState<Note | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let current = true
    api.notes.list(appId).then(
      (result) => {
        if (current) {
          setNotes(result)
          setError("")
          setLoading(false)
        }
      },
      (err) => {
        if (current) {
          setError(message(err))
          setLoading(false)
        }
      }
    )
    return () => {
      current = false
    }
  }, [appId, dataEpoch, reload])

  function saved(note: Note) {
    setNotes((current) =>
      [note, ...current.filter((entry) => entry.id !== note.id)].sort(
        (a, b) => b.id - a.id
      )
    )
  }

  async function toggle(note: Note, index: number, done: boolean) {
    setBusy(true)
    const items: NoteItem[] = note.items.map((item, i) =>
      i === index ? { ...item, done } : item
    )
    try {
      saved(
        await api.notes.save(appId, note.id, {
          title: note.title,
          content: note.content,
          items,
        })
      )
    } catch (err) {
      toast.error(message(err))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!deleting) return
    setBusy(true)
    try {
      await api.notes.delete(appId, deleting.id)
      setNotes((current) => current.filter((entry) => entry.id !== deleting.id))
      setDeleting(null)
      toast.success("Note deleted")
    } catch (err) {
      toast.error(message(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-4 pb-4">
      <div className="flex flex-wrap items-center justify-end gap-3">
        <Button size="sm" onClick={() => setEditor({ note: null })}>
          <PlusIcon data-icon="inline-start" />
          Add note
        </Button>
      </div>
      {error && (
        <div role="alert" className="flex items-center gap-3">
          <p className="text-sm text-destructive">{error}</p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setReload((n) => n + 1)}
          >
            Retry
          </Button>
        </div>
      )}
      {loading ? (
        <p role="status" className="text-sm text-muted-foreground">
          Loading notes…
        </p>
      ) : !error && notes.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <StickyNoteIcon />
            </EmptyMedia>
            <EmptyTitle>No notes yet</EmptyTitle>
            <EmptyDescription>
              Add a note to keep ideas, screenshots and checklists with this
              app.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : null}
      <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,18rem),1fr))] items-start gap-4">
        {notes.map((note) => (
          <article key={note.id} aria-label={note.title} className="min-w-0">
            <Card size="sm">
              {note.image_path && (
                <NoteImage key={note.image_path} note={note} />
              )}
              <CardHeader>
                <CardTitle>
                  <h3 className="wrap-anywhere">{note.title}</h3>
                </CardTitle>
              </CardHeader>
              {(note.content || note.items.length > 0) && (
                <CardContent className="flex flex-col gap-4">
                  {note.content && (
                    <p className="wrap-anywhere whitespace-pre-wrap">
                      {note.content}
                    </p>
                  )}
                  {note.items.length > 0 && (
                    <ul className="flex flex-col gap-3" aria-label="Checklist">
                      {note.items.map((item, index) => (
                        <li key={index}>
                          <label className="flex cursor-pointer items-start gap-3">
                            <Checkbox
                              checked={item.done}
                              disabled={busy}
                              onCheckedChange={(done) =>
                                void toggle(note, index, done)
                              }
                            />
                            <span
                              className={cn(
                                "min-w-0 wrap-anywhere whitespace-pre-wrap",
                                item.done &&
                                  "text-muted-foreground line-through"
                              )}
                            >
                              {item.text}
                            </span>
                          </label>
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              )}
              <CardFooter className="justify-between gap-2">
                <span className="text-xs text-muted-foreground">
                  {note.items.length > 0
                    ? `${note.items.filter((item) => item.done).length}/${note.items.length} done`
                    : "Note"}
                </span>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={busy}
                    aria-label={`Edit ${note.title}`}
                    onClick={() => setEditor({ note })}
                  >
                    <PencilIcon />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={busy}
                    aria-label={`Delete ${note.title}`}
                    onClick={() => setDeleting(note)}
                  >
                    <Trash2Icon />
                  </Button>
                </div>
              </CardFooter>
            </Card>
          </article>
        ))}
      </div>
      {editor && (
        <NoteEditor
          appId={appId}
          note={editor.note}
          onClose={() => setEditor(null)}
          onSaved={saved}
        />
      )}
      <AlertDialog
        open={!!deleting}
        onOpenChange={(open) => {
          if (!open && !busy) setDeleting(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete note?</AlertDialogTitle>
            <AlertDialogDescription>
              “{deleting?.title}” and its saved image will be permanently
              deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => setDeleting(null)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => void remove()}
            >
              {busy ? "Deleting…" : "Delete note"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
