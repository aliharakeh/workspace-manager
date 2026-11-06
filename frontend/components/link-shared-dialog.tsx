import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api } from "@/lib/api"
import type { SharedSet, SharedSetDetail } from "@/lib/types"
import { useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

type LinkSharedDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspaceId: number
  /** The app config set that takes the variables. */
  configSetId: number
  configSetName: string
  /** The shared set it is linked to now (null: not linked yet). */
  currentSetId: number | null
  /** Keys of the config set's own (not shared) variables. */
  ownKeys: string[]
  onLinked: () => void
}

/**
 * Links an app config set to a shared set of its workspace. The config set
 * takes every variable of that set, now and whenever the set gains one (for
 * example from its script). They are read-only in the app and follow the
 * workspace's changes. Any shared set can be chosen, whatever the names.
 */
export function LinkSharedDialog({
  open,
  onOpenChange,
  workspaceId,
  configSetId,
  configSetName,
  currentSetId,
  ownKeys,
  onLinked,
}: LinkSharedDialogProps) {
  const [sets, setSets] = useState<SharedSet[] | null>(null)
  const [setId, setSetId] = useState<number | null>(null)
  const [detail, setDetail] = useState<SharedSetDetail | null>(null)
  const [replace, setReplace] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setSets(null)
    setDetail(null)
    setReplace(false)
    api.sharedSets
      .list(workspaceId)
      .then((list) => {
        if (cancelled) return
        setSets(list)
        setSetId(
          list.some((s) => s.id === currentSetId)
            ? currentSetId
            : (list[0]?.id ?? null)
        )
      })
      .catch((err) =>
        toast.error(err instanceof Error ? err.message : "Failed to load shared config")
      )
    return () => {
      cancelled = true
    }
  }, [open, workspaceId, currentSetId])

  useEffect(() => {
    if (!open || setId == null) return
    let cancelled = false
    setDetail(null)
    api.sharedSets
      .get(setId)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((err) =>
        toast.error(err instanceof Error ? err.message : "Failed to load variables")
      )
    return () => {
      cancelled = true
    }
  }, [open, setId])

  const vars = useMemo(() => detail?.vars ?? [], [detail])
  const clashes = useMemo(
    () =>
      setId === currentSetId ? [] : vars.filter((v) => ownKeys.includes(v.key)),
    [vars, ownKeys, setId, currentSetId]
  )

  async function handleSave() {
    if (setId == null) return
    setSaving(true)
    try {
      await api.sharedSets.link(configSetId, { shared_set_id: setId, replace })
      toast.success(
        currentSetId == null ? "Shared variables linked" : "Shared set changed"
      )
      onLinked()
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to link")
    } finally {
      setSaving(false)
    }
  }

  const switching = currentSetId != null && setId !== currentSetId

  return (
    <Dialog open={open} onOpenChange={(next) => !saving && onOpenChange(next)}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            Shared variables for <span className="font-mono">{configSetName}</span>
          </DialogTitle>
          <DialogDescription>
            Pick a workspace config set. This config set takes all of its
            variables, including ones added later by its script. They are
            read-only here and change whenever the workspace changes them.
          </DialogDescription>
        </DialogHeader>

        {sets === null ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : sets.length === 0 ? (
          <p className="rounded-lg border p-4 text-sm text-muted-foreground">
            This workspace has no shared config yet. Open the workspace and use
            “Shared config” to create a set and add variables.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            <Select
              items={sets.map((s) => ({ value: String(s.id), label: s.name }))}
              value={setId == null ? "" : String(setId)}
              onValueChange={(next) => {
                if (next !== null) setSetId(Number(next))
              }}
              disabled={saving}
            >
              <SelectTrigger aria-label="Workspace config set">
                <SelectValue placeholder="Workspace config set" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {sets.map((s) => (
                    <SelectItem key={s.id} value={String(s.id)}>
                      {s.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>

            {switching ? (
              <p className="text-xs text-muted-foreground">
                Switching sets replaces the variables taken from the current one.
              </p>
            ) : null}

            {detail === null ? (
              <p className="text-sm text-muted-foreground">Loading variables…</p>
            ) : vars.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                This set has no variables yet. Variables added to it later are
                linked automatically.
              </p>
            ) : (
              <>
                <p className="text-xs text-muted-foreground">
                  {vars.length} variable{vars.length === 1 ? "" : "s"} will be
                  linked.
                </p>
                <div className="flex max-h-72 flex-col gap-1 overflow-y-auto">
                  {vars.map((v) => (
                    <div
                      key={v.id}
                      className="flex items-center gap-3 rounded-md border px-3 py-2"
                    >
                      <span className="font-mono text-sm">{v.key}</span>
                      <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
                        {v.private ? "••••••" : v.value}
                      </span>
                      {ownKeys.includes(v.key) ? (
                        <Badge variant="outline">already defined here</Badge>
                      ) : null}
                    </div>
                  ))}
                </div>
                {clashes.length > 0 ? (
                  <label className="flex items-start gap-2 rounded-md border border-amber-500/50 p-3 text-sm">
                    <Checkbox
                      checked={replace}
                      disabled={saving}
                      onCheckedChange={(checked) => setReplace(checked === true)}
                    />
                    <span>
                      Replace this config set&apos;s own{" "}
                      <span className="font-mono">
                        {clashes.map((v) => v.key).join(", ")}
                      </span>{" "}
                      with the shared value
                      {clashes.length === 1 ? "" : "s"}.
                    </span>
                  </label>
                ) : null}
              </>
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" disabled={saving} onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={
              saving ||
              setId == null ||
              setId === currentSetId ||
              (clashes.length > 0 && !replace)
            }
            onClick={() => void handleSave()}
          >
            {saving ? "Saving…" : "Link"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
