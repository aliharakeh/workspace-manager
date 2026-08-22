import { useEffect, useMemo, useState } from "react"
import { BotIcon, SparklesIcon, XIcon } from "lucide-react"
import { toast } from "sonner"
import { DiffModeEnum, DiffView } from "@git-diff-view/react"
import { generateDiffFile } from "@git-diff-view/file"
import "@git-diff-view/react/styles/diff-view.css"
import { api } from "@/lib/api"
import {
  APP_AI_SYSTEM_PROMPT,
  buildAppAIPrompt,
  parseAppAIResponse,
  patchHasEdits,
  type AppAIChatTurn,
  type AppAIPatch,
} from "@/lib/app-ai"
import {
  appAIDiffCount,
  buildAppAIDiff,
  langForPath,
  normalizeNewlines,
  type AppAIDiff,
  type AppAIFileDiff,
} from "@/lib/app-ai-diff"
import type { App, ConfigSetDetail } from "@/lib/types"
import { useTheme } from "@/components/theme-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Field,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field"
import { Toggle } from "@/components/ui/toggle"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { ScrollArea } from "@/components/ui/scroll-area"

type AppAIPanelProps = {
  app: App
  onApplied: () => void
}

type ReviewStatus = "pending" | "applied" | "discarded"

type ChatMessage = AppAIChatTurn & {
  id: number
  diff?: AppAIDiff
  patch?: AppAIPatch
  status?: ReviewStatus
}

function SplitDiff({
  file,
  theme,
  ignoreWhitespace,
}: {
  file: AppAIFileDiff
  theme: "light" | "dark"
  ignoreWhitespace: boolean
}) {
  const lang = langForPath(file.path)
  const diffFile = useMemo(() => {
    const oldText = normalizeNewlines(file.oldText)
    const newText = normalizeNewlines(file.newText)
    const instance = generateDiffFile(
      file.path,
      oldText,
      file.path,
      newText,
      lang,
      lang,
      { ignoreWhitespace, stripTrailingCr: true }
    )
    instance.initTheme(theme)
    instance.initRaw()
    instance.buildSplitDiffLines()
    return instance
  }, [file.path, file.oldText, file.newText, lang, theme, ignoreWhitespace])

  return (
    <div className="overflow-auto rounded-md border">
      <DiffView
        diffFile={diffFile}
        diffViewMode={DiffModeEnum.SplitGitHub}
        diffViewTheme={theme}
        diffViewWrap
        diffViewFontSize={12}
      />
    </div>
  )
}

function AppAIDiffView({
  diff,
  status,
  applying,
  onApply,
  onDiscard,
}: {
  diff: AppAIDiff
  status?: ReviewStatus
  applying?: boolean
  onApply?: () => void
  onDiscard?: () => void
}) {
  const { resolvedTheme } = useTheme()
  const [hideWhitespace, setHideWhitespace] = useState(false)
  const n = appAIDiffCount(diff)
  if (n === 0) return null
  const pending = status === "pending" && diff.files.length > 0

  return (
    <div className="mt-2 flex flex-col gap-2">
      <details open={pending || status == null}>
        <summary className="cursor-pointer text-xs text-muted-foreground">
          {n} {n === 1 ? "change" : "changes"}
          {status === "applied" ? " · applied" : null}
          {status === "discarded" ? " · discarded" : null}
        </summary>
        <div className="mt-2 flex flex-col gap-3">
          {diff.files.length > 0 ? (
            <Toggle
              variant="outline"
              size="sm"
              pressed={hideWhitespace}
              onPressedChange={setHideWhitespace}
            >
              Hide whitespace
            </Toggle>
          ) : null}
          {diff.files.map((file) => (
            <div key={file.path} className="flex min-w-0 flex-col gap-1">
              <p className="font-mono text-xs font-medium">{file.path}</p>
              <SplitDiff
                file={file}
                theme={resolvedTheme}
                ignoreWhitespace={hideWhitespace}
              />
            </div>
          ))}
          {diff.skipped.map((path) => (
            <p key={path} className="font-mono text-xs text-muted-foreground">
              {path} — not on this config set, skipped
            </p>
          ))}
        </div>
      </details>
      {pending ? (
        <div className="flex gap-2">
          <Button size="sm" disabled={applying} onClick={onApply}>
            {applying ? "Applying…" : "Apply"}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={applying}
            onClick={onDiscard}
          >
            Discard
          </Button>
        </div>
      ) : status === "applied" ? (
        <Badge variant="secondary">Applied</Badge>
      ) : status === "discarded" ? (
        <Badge variant="outline">Discarded</Badge>
      ) : null}
    </div>
  )
}

async function applyAppAIPatch(
  appId: number,
  current: ConfigSetDetail,
  patch: AppAIPatch
): Promise<void> {
  const varsByKey = new Map(current.env_vars.map((v) => [v.key, v]))

  for (const key of patch.env?.delete ?? []) {
    const existing = varsByKey.get(key)
    if (!existing) continue
    await api.envVars.delete(existing.id)
    varsByKey.delete(key)
  }

  for (const item of patch.env?.upsert ?? []) {
    const existing = varsByKey.get(item.key)
    if (existing) {
      if (existing.value === item.value) continue
      await api.envVars.update(existing.id, { value: item.value })
    } else {
      await api.envVars.create(appId, { key: item.key, value: item.value })
    }
  }

  const templatesByPath = new Map(
    current.templates.map((t) => [t.file_path, t])
  )
  for (const t of patch.templates ?? []) {
    const existing = templatesByPath.get(t.file_path)
    if (!existing || existing.content === t.content) continue
    await api.templates.update(existing.id, { content: t.content })
  }

  if (patch.run) {
    const mode = patch.run.mode ?? current.run_config?.mode ?? "parallel"
    const commands =
      patch.run.commands ??
      (current.run_config?.commands ?? []).map((c) => ({
        label: c.label,
        command: c.command,
      }))
    await api.runConfig.save(appId, { mode, commands })
  }
}

function syncTemplateSelection(
  paths: string[],
  prevPaths: string[],
  prevSelected: Set<string>
): Set<string> {
  const known = new Set(prevPaths)
  const next = new Set<string>()
  for (const p of paths) {
    if (!known.has(p) || prevSelected.has(p)) next.add(p)
  }
  return next
}

export function AppAIPanel({ app, onApplied }: AppAIPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState("")
  const [sending, setSending] = useState(false)
  const [applyingId, setApplyingId] = useState<number | null>(null)
  const [templatePaths, setTemplatePaths] = useState<string[]>([])
  const [selectedTemplates, setSelectedTemplates] = useState<Set<string>>(
    new Set()
  )
  const setId = app.active_config_set_id
  const setName = app.active_config_set_name

  useEffect(() => {
    if (setId == null) {
      setTemplatePaths([])
      setSelectedTemplates(new Set())
      return
    }
    let cancelled = false
    void api.configSets.getDetail(setId).then(
      (d) => {
        if (cancelled) return
        const paths = d.templates.map((t) => t.file_path)
        setTemplatePaths(paths)
        setSelectedTemplates(new Set(paths))
      },
      () => {
        if (cancelled) return
        setTemplatePaths([])
        setSelectedTemplates(new Set())
      }
    )
    return () => {
      cancelled = true
    }
  }, [setId])

  async function handleSend() {
    const instruction = input.trim()
    if (!instruction) return
    if (setId == null) {
      toast.error("Select a config set first")
      return
    }

    const userMsg: ChatMessage = {
      id: Date.now(),
      role: "user",
      text: instruction,
    }
    const history = messages.map(({ role, text }) => ({ role, text }))
    setMessages((prev) => [...prev, userMsg])
    setInput("")
    setSending(true)
    try {
      const [detail, latestApp] = await Promise.all([
        api.configSets.getDetail(setId),
        api.apps.get(app.id),
      ])
      if (latestApp.active_config_set_id !== setId) {
        toast.error("Config set changed — send again against the selected set")
        return
      }
      const paths = detail.templates.map((t) => t.file_path)
      setTemplatePaths(paths)
      setSelectedTemplates((prev) =>
        syncTemplateSelection(paths, templatePaths, prev)
      )
      const included =
        templatePaths.length === 0
          ? paths
          : paths.filter((p) => selectedTemplates.has(p))
      const prompt = buildAppAIPrompt({
        appName: app.name,
        projectPath: app.project_path,
        configSet: detail,
        history,
        instruction,
        templatePaths: included,
      })
      const res = await api.ai.chat({
        system: APP_AI_SYSTEM_PROMPT,
        prompt,
      })
      const parsed = parseAppAIResponse(res.text)
      const allow = new Set(included)
      const patch: AppAIPatch = {
        ...parsed,
        templates: parsed.templates?.filter((t) => allow.has(t.file_path)),
      }
      if (patch.templates?.length === 0) delete patch.templates
      const diff = patchHasEdits(patch)
        ? buildAppAIDiff(detail, patch)
        : undefined
      const pending = diff && diff.files.length > 0
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now() + 1,
          role: "assistant",
          text: patch.message,
          diff,
          patch: pending ? patch : undefined,
          status: pending ? "pending" : undefined,
        },
      ])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "AI request failed")
    } finally {
      setSending(false)
    }
  }

  async function handleApply(msg: ChatMessage) {
    if (!msg.patch || setId == null) return
    setApplyingId(msg.id)
    try {
      const [detail, latestApp] = await Promise.all([
        api.configSets.getDetail(setId),
        api.apps.get(app.id),
      ])
      if (latestApp.active_config_set_id !== setId) {
        toast.error("Config set changed — edits were not applied")
        return
      }
      await applyAppAIPatch(app.id, detail, msg.patch)
      setMessages((prev) =>
        prev.map((m) => (m.id === msg.id ? { ...m, status: "applied" } : m))
      )
      onApplied()
      toast.success("Changes applied")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to apply")
    } finally {
      setApplyingId(null)
    }
  }

  function handleDiscard(id: number) {
    setMessages((prev) =>
      prev.map((m) => (m.id === id ? { ...m, status: "discarded" } : m))
    )
  }

  if (setId == null) {
    return (
      <Empty className="border">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <BotIcon />
          </EmptyMedia>
          <EmptyTitle>No config set selected</EmptyTitle>
          <EmptyDescription>
            Pick a config set above. AI only edits the selected set.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm text-muted-foreground">
            Ask to change env vars, templates, or run commands. Review, then
            apply — only for
          </p>
          <Badge variant="secondary">{setName ?? `set ${setId}`}</Badge>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={messages.length === 0 && !input && !sending}
          onClick={() => {
            setMessages([])
            setInput("")
            setApplyingId(null)
          }}
        >
          Clear
        </Button>
      </div>

      {templatePaths.length > 0 ? (
        <details className="rounded-lg border px-3 py-2">
          <summary className="cursor-pointer text-sm text-muted-foreground">
            Templates in prompt ({selectedTemplates.size} of{" "}
            {templatePaths.length})
          </summary>
          <FieldSet className="mt-2 gap-2">
            <FieldLegend className="sr-only">Templates to send</FieldLegend>
            <Field orientation="horizontal">
              <Checkbox
                id="ai-templates-all"
                checked={selectedTemplates.size === templatePaths.length}
                onCheckedChange={(checked) =>
                  setSelectedTemplates(
                    checked === true
                      ? new Set(templatePaths)
                      : new Set()
                  )
                }
              />
              <FieldLabel htmlFor="ai-templates-all">Select all</FieldLabel>
            </Field>
            <div className="flex max-h-40 flex-col gap-1 overflow-y-auto">
              {templatePaths.map((path, index) => {
                const id = `ai-template-${index}`
                return (
                  <Field key={path} orientation="horizontal" className="min-w-0">
                    <Checkbox
                      id={id}
                      checked={selectedTemplates.has(path)}
                      onCheckedChange={(checked) => {
                        setSelectedTemplates((prev) => {
                          const next = new Set(prev)
                          if (checked === true) next.add(path)
                          else next.delete(path)
                          return next
                        })
                      }}
                    />
                    <FieldLabel
                      htmlFor={id}
                      className="min-w-0 truncate font-normal"
                      title={path}
                    >
                      {path}
                    </FieldLabel>
                  </Field>
                )
              })}
            </div>
          </FieldSet>
        </details>
      ) : null}

      <ScrollArea className="min-h-0 flex-1 rounded-lg border">
        <div className="flex flex-col gap-3 p-3">
          {messages.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              e.g. “Set PORT to 5173 and use {`{{PORT}}`} in the env template”,
              or “Run web with npm run dev in parallel”.
            </p>
          ) : (
            messages.map((m) => (
              <div
                key={m.id}
                className={
                  m.role === "user"
                    ? "ml-8 rounded-lg bg-muted px-3 py-2 text-sm"
                    : m.diff
                      ? "rounded-lg border px-3 py-2 text-sm"
                      : "mr-8 rounded-lg border px-3 py-2 text-sm"
                }
              >
                <p className="whitespace-pre-wrap">{m.text}</p>
                {m.diff ? (
                  <AppAIDiffView
                    diff={m.diff}
                    status={m.status}
                    applying={applyingId === m.id}
                    onApply={() => void handleApply(m)}
                    onDiscard={() => handleDiscard(m.id)}
                  />
                ) : null}
              </div>
            ))
          )}
          {sending ? (
            <p className="text-xs text-muted-foreground">Updating…</p>
          ) : null}
        </div>
      </ScrollArea>

      <div className="flex items-center gap-2">
        <InputGroup className="flex-1">
          <InputGroupAddon align="inline-start">
            <BotIcon />
          </InputGroupAddon>
          <InputGroupInput
            value={input}
            disabled={sending}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault()
                void handleSend()
              }
            }}
            placeholder="Ask AI to update this config set…"
            aria-label="AI prompt for app config"
          />
          {input ? (
            <InputGroupAddon align="inline-end">
              <InputGroupButton
                size="icon-xs"
                aria-label="Clear prompt"
                onClick={() => setInput("")}
              >
                <XIcon />
              </InputGroupButton>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
        <Button
          disabled={sending || !input.trim()}
          onClick={() => void handleSend()}
        >
          <SparklesIcon data-icon="inline-start" />
          {sending ? "Updating…" : "Ask AI"}
        </Button>
      </div>
    </div>
  )
}
