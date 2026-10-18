import { useEffect, useState } from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type {
  CompareStatus,
  ConfigSet,
  ConfigSetCompare,
  CopyParts,
} from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  sets: ConfigSet[]
  /** The set being compared (the left side, and the target of a copy). */
  current: ConfigSet | null
  /** Called after a part was copied into `current`. */
  onCopied: () => void
}

const STATUS_LABEL: Record<CompareStatus, string> = {
  only_left: "only here",
  only_right: "only there",
  same: "same",
  different: "different",
  changed: "changed",
}

const STATUS_CLASS: Record<CompareStatus, string> = {
  only_left: "text-amber-600 dark:text-amber-400",
  only_right: "text-sky-600 dark:text-sky-400",
  same: "text-muted-foreground",
  different: "text-red-600 dark:text-red-400",
  changed: "text-red-600 dark:text-red-400",
}

function StatusTag({ status }: { status: CompareStatus }) {
  return (
    <span className={`text-xs ${STATUS_CLASS[status]}`}>
      {STATUS_LABEL[status]}
    </span>
  )
}

function Side({
  value,
  redacted,
}: {
  value: string | null
  redacted?: boolean
}) {
  if (redacted) {
    return <span className="italic text-muted-foreground">hidden</span>
  }
  if (value === null) return <span className="text-muted-foreground">—</span>
  return (
    <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-xs">
      {value === "" ? <span className="text-muted-foreground">(empty)</span> : value}
    </pre>
  )
}

function Row({
  title,
  status,
  left,
  right,
  redacted,
}: {
  title: string
  status: CompareStatus
  left: string | null
  right: string | null
  redacted?: boolean
}) {
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-x-3 gap-y-1 border-b py-2">
      <div className="col-span-2 flex items-center gap-2">
        <span className="truncate font-mono text-xs font-medium">{title}</span>
        <StatusTag status={status} />
      </div>
      <Side value={left} redacted={redacted} />
      <Side value={right} redacted={redacted} />
    </div>
  )
}

function SectionHeader({
  leftName,
  rightName,
  copyLabel,
  copying,
  disabled,
  onCopy,
}: {
  leftName: string
  rightName: string
  copyLabel: string
  copying: boolean
  disabled: boolean
  onCopy: () => void
}) {
  const [armed, setArmed] = useState(false)
  return (
    <div className="flex items-center gap-3 border-b pb-2">
      <div className="grid flex-1 grid-cols-2 gap-3 text-xs font-medium">
        <span className="truncate">{leftName}</span>
        <span className="truncate">{rightName}</span>
      </div>
      <Button
        size="xs"
        variant={armed ? "destructive" : "outline"}
        disabled={disabled || copying}
        onBlur={() => setArmed(false)}
        onClick={() => {
          if (!armed) {
            setArmed(true)
            return
          }
          setArmed(false)
          onCopy()
        }}
      >
        {armed ? "Confirm: replace" : copyLabel}
      </Button>
    </div>
  )
}

export function ConfigSetCompareDialog({
  open,
  onOpenChange,
  sets,
  current,
  onCopied,
}: Props) {
  // The picked set; null means "the first other set of the app".
  const [picked, setPicked] = useState<number | null>(null)
  const [outcome, setOutcome] = useState<{
    key: string
    result: ConfigSetCompare | null
    error: string | null
  } | null>(null)
  const [onlyDiff, setOnlyDiff] = useState(true)
  const [copying, setCopying] = useState(false)
  const [reload, setReload] = useState(0)

  const currentId = current?.id
  const fallbackId =sets.find((s) => s.id !== current?.id)?.id ?? ""
  const otherId: number | "" =
    picked !== null && picked !== current?.id && sets.some((s) => s.id === picked)
      ? picked
      : fallbackId
  const key = `${current?.id}:${otherId}:${reload}`
  const fresh = outcome?.key === key ? outcome : null
  const result = fresh?.result ?? outcome?.result ?? null
  const error = fresh?.error ?? null
  const loading = open && otherId !== "" && fresh === null

  useEffect(() => {
    if (!open || currentId === undefined || otherId === "") return
    let stale = false
    api.configSets
      .compare(currentId, otherId)
      .then((r) => {
        if (!stale) setOutcome({ key, result: r as ConfigSetCompare, error: null })
      })
      .catch((err) => {
        if (stale) return
        setOutcome({
          key,
          result: null,
          error: err instanceof Error ? err.message : "Failed to compare",
        })
      })
    return () => {
      stale = true
    }
  }, [open, currentId, otherId, key])

  async function copyPart(parts: CopyParts, what: string) {
    if (!current || otherId === "") return
    setCopying(true)
    try {
      await api.configSets.copyFrom(current.id, otherId, parts)
      toast.success(`Replaced ${what} in “${current.name}”`)
      onCopied()
      setReload((n) => n + 1)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to copy")
    } finally {
      setCopying(false)
    }
  }

  const show = (status: string) => !onlyDiff || status !== "same"
  const leftName = result?.left.name ?? current?.name ?? ""
  const rightName = result?.right.name ?? ""
  const count = (items: { status: string }[]) =>
    items.filter((i) => i.status !== "same").length

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-hidden sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Compare “{current?.name}” with…</DialogTitle>
          <DialogDescription>
            Left is the current set, right is the one you pick. Sets of the
            same app only.
          </DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-3">
          <Select
            items={sets.map((s) => ({ value: String(s.id), label: s.name }))}
            value={otherId === "" ? null : String(otherId)}
            onValueChange={(v) => setPicked(v === null ? null : Number(v))}
          >
            <SelectTrigger className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {sets
                  .filter((s) => s.id !== current?.id)
                  .map((s) => (
                    <SelectItem key={s.id} value={String(s.id)}>
                      {s.name}
                    </SelectItem>
                  ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <label className="flex items-center gap-2 text-xs">
            <Checkbox
              checked={onlyDiff}
              onCheckedChange={(c) => setOnlyDiff(c === true)}
            />
            Show only differences
          </label>
        </div>
        {loading && !result ? (
          <p className="text-xs text-muted-foreground">Comparing…</p>
        ) : null}
        {error ? <p className="text-xs text-destructive">{error}</p> : null}
        {result ? (
          <Tabs defaultValue="env" className="min-h-0 flex-1">
            <TabsList>
              <TabsTrigger value="env">
                Env ({count(result.env)})
              </TabsTrigger>
              <TabsTrigger value="templates">
                Templates ({count(result.templates)})
              </TabsTrigger>
              {result.commands.map((c) => (
                <TabsTrigger key={c.kind} value={c.kind}>
                  {c.kind[0].toUpperCase() + c.kind.slice(1)} (
                  {c.status === "same" ? 0 : 1})
                </TabsTrigger>
              ))}
            </TabsList>
            <TabsContent value="env" className="mt-3 max-h-[50vh] overflow-auto">
              <SectionHeader
                leftName={leftName}
                rightName={rightName}
                copyLabel={`Replace with “${rightName}”`}
                copying={copying}
                disabled={count(result.env) === 0}
                onCopy={() => void copyPart({ env: true }, "env vars")}
              />
              {result.env.filter((e) => show(e.status)).map((e) => (
                <Row
                  key={e.key}
                  title={e.key}
                  status={e.status}
                  left={e.left}
                  right={e.right}
                  redacted={e.redacted}
                />
              ))}
              {result.env.filter((e) => show(e.status)).length === 0 ? (
                <p className="py-3 text-xs text-muted-foreground">
                  No differences.
                </p>
              ) : null}
            </TabsContent>
            <TabsContent
              value="templates"
              className="mt-3 max-h-[50vh] overflow-auto"
            >
              <SectionHeader
                leftName={leftName}
                rightName={rightName}
                copyLabel={`Replace with “${rightName}”`}
                copying={copying}
                disabled={count(result.templates) === 0}
                onCopy={() => void copyPart({ templates: true }, "templates")}
              />
              {result.templates.filter((t) => show(t.status)).map((t) => (
                <Row
                  key={t.file_path}
                  title={t.file_path}
                  status={t.status}
                  left={t.left}
                  right={t.right}
                  redacted={t.redacted}
                />
              ))}
              {result.templates.filter((t) => show(t.status)).length === 0 ? (
                <p className="py-3 text-xs text-muted-foreground">
                  No differences.
                </p>
              ) : null}
            </TabsContent>
            {result.commands.map((list) => (
              <TabsContent
                key={list.kind}
                value={list.kind}
                className="mt-3 max-h-[50vh] overflow-auto"
              >
                <SectionHeader
                  leftName={leftName}
                  rightName={rightName}
                  copyLabel={`Replace with “${rightName}”`}
                  copying={copying}
                  disabled={list.status === "same"}
                  onCopy={() =>
                    void copyPart(
                      { [list.kind]: true } as CopyParts,
                      `${list.kind} commands`
                    )
                  }
                />
                {list.left_mode !== list.right_mode ? (
                  <p className="py-2 text-xs">
                    Mode: <span className="font-mono">{list.left_mode || "—"}</span>
                    {" vs "}
                    <span className="font-mono">{list.right_mode || "—"}</span>
                  </p>
                ) : null}
                {list.commands.filter((c) => show(c.status)).map((c, i) => (
                  <Row
                    key={`${c.label}-${i}`}
                    title={c.label || `#${i + 1}`}
                    status={c.status}
                    left={c.left}
                    right={c.right}
                  />
                ))}
                {list.commands.filter((c) => show(c.status)).length === 0 &&
                list.left_mode === list.right_mode ? (
                  <p className="py-3 text-xs text-muted-foreground">
                    No differences.
                  </p>
                ) : null}
              </TabsContent>
            ))}
          </Tabs>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
