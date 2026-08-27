import { Fragment, useEffect, useMemo, useRef, useState } from "react"
import Markdown, { type Components } from "react-markdown"
import { BotIcon, SparklesIcon, XIcon } from "lucide-react"
import { toast } from "sonner"
import { DiffModeEnum, DiffView } from "@git-diff-view/react"
import { generateDiffFile } from "@git-diff-view/file"
import "@git-diff-view/react/styles/diff-view.css"
import { api } from "@/lib/api"
import {
  patchHasEdits,
  summarizeAIToolInput,
  type AppAIChatTurn,
  type AppAIPatch,
  type AppAIStreamEvent,
  type AppAIToolCall,
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
import { Toggle } from "@/components/ui/toggle"
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

type AppAIPanelProps = {
  app: App
  onApplied: () => void
}

type ReviewStatus = "pending" | "applied" | "discarded"

type ChatMessage = AppAIChatTurn & {
  id: number
  tools?: AppAIToolCall[]
  diff?: AppAIDiff
  patch?: AppAIPatch
  status?: ReviewStatus
}

function jsonBlock(value: unknown) {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

const mdComponents: Components = {
  h1: ({ children }) => <p className="font-medium">{children}</p>,
  h2: ({ children }) => <p className="font-medium">{children}</p>,
  h3: ({ children }) => <p className="font-medium">{children}</p>,
  ul: ({ children }) => <ul className="list-disc pl-4">{children}</ul>,
  ol: ({ children }) => <ol className="list-decimal pl-4">{children}</ol>,
  pre: ({ children }) => (
    <pre className="overflow-auto rounded-md bg-muted p-2 font-mono text-xs">
      {children}
    </pre>
  ),
  code: ({ className, children }) =>
    className ? (
      <code className={className}>{children}</code>
    ) : (
      <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">
        {children}
      </code>
    ),
  a: ({ href, children }) => (
    <a
      href={href}
      className="underline underline-offset-3"
      target="_blank"
      rel="noreferrer"
    >
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="border-l pl-3 text-muted-foreground">
      {children}
    </blockquote>
  ),
}

function AssistantMarkdown({ text }: { text: string }) {
  return (
    <div className="flex flex-col gap-2">
      <Markdown components={mdComponents}>{text}</Markdown>
    </div>
  )
}

function AppAIToolCallRow({ call }: { call: AppAIToolCall }) {
  const hint = summarizeAIToolInput(call.input)
  return (
    <details className="rounded-lg border px-3 py-2">
      <summary className="cursor-pointer truncate text-xs">
        <span className="font-mono font-medium">{call.name}</span>
        {hint ? (
          <span className="ml-2 text-muted-foreground">{hint}</span>
        ) : null}
      </summary>
      <pre className="mt-2 max-h-40 overflow-auto rounded-md bg-muted p-2 text-xs">
        {jsonBlock({ input: call.input, output: call.output })}
      </pre>
    </details>
  )
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
    <div className="thin-scrollbar max-h-80 overflow-auto rounded-md border">
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
    if (!existing || !existing.include_in_ai) continue
    await api.envVars.delete(existing.id)
    varsByKey.delete(key)
  }

  for (const item of patch.env?.upsert ?? []) {
    const existing = varsByKey.get(item.key)
    if (existing && !existing.include_in_ai) continue
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

export function AppAIPanel({ app, onApplied }: AppAIPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState("")
  const [sending, setSending] = useState(false)
  const [applyingId, setApplyingId] = useState<number | null>(null)
  const [confirmClear, setConfirmClear] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const setId = app.active_config_set_id
  const setName = app.active_config_set_name

  useEffect(() => {
    if (!sending) inputRef.current?.focus()
  }, [sending])

  useEffect(() => {
    const el = listRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [messages, sending])

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
    const history = messages.map(({ role, text, tools }) => ({
      role,
      text,
      ...(tools?.length ? { tools } : {}),
    }))
    setMessages((prev) => [...prev, userMsg])
    setInput("")
    setSending(true)
    const assistantId = Date.now() + 1
    setMessages((prev) => [
      ...prev,
      { id: assistantId, role: "assistant", text: "", tools: [] },
    ])
    const applyEvent = (ev: AppAIStreamEvent) => {
      setMessages((prev) =>
        prev.map((m) => {
          if (m.id !== assistantId) return m
          if (ev.type === "tool") {
            return { ...m, tools: [...(m.tools ?? []), ev.call] }
          }
          if (ev.type === "text") {
            return { ...m, text: m.text + ev.text }
          }
          return m
        })
      )
    }
    try {
      const [detail, latestApp] = await Promise.all([
        api.configSets.getDetail(setId),
        api.apps.get(app.id),
      ])
      if (latestApp.active_config_set_id !== setId) {
        toast.error("Config set changed — send again against the selected set")
        setMessages((prev) => prev.filter((m) => m.id !== assistantId))
        return
      }
      const res = await api.ai.appChat(
        {
          appId: app.id,
          configSetId: setId,
          history,
          instruction,
        },
        applyEvent
      )
      const patch: AppAIPatch = {
        message: res.text.trim() || res.patch.message || "Done.",
        env: res.patch.env,
        templates: res.patch.templates,
        run: res.patch.run,
      }
      const diff = patchHasEdits(patch)
        ? buildAppAIDiff(detail, patch)
        : undefined
      const pending = diff && diff.files.length > 0
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId
            ? {
                ...m,
                text: m.text.trim() || patch.message,
                tools: res.toolCalls?.length ? res.toolCalls : m.tools,
                diff,
                patch: pending ? patch : undefined,
                status: pending ? "pending" : undefined,
              }
            : m
        )
      )
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "AI request failed")
      setMessages((prev) =>
        prev.filter(
          (m) => m.id !== assistantId || !!m.text || !!m.tools?.length
        )
      )
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

  function handleConfirmClear() {
    setMessages([])
    setInput("")
    setApplyingId(null)
    setConfirmClear(false)
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
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2">
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
          onClick={() => setConfirmClear(true)}
        >
          Clear
        </Button>
      </div>

      <div
        ref={listRef}
        className="thin-scrollbar min-h-0 flex-1 overflow-y-auto rounded-lg border"
      >
        <div className="flex flex-col gap-3 p-3">
          {messages.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              e.g. “Set PORT to 5173 and use {`{{PORT}}`} in the env template”,
              or “Run web with npm run dev in parallel”.
            </p>
          ) : (
            messages.map((m) => (
              <Fragment key={m.id}>
                {m.role === "assistant" && m.tools?.length
                  ? m.tools.map((c, i) => (
                      <AppAIToolCallRow key={`${m.id}-${c.name}-${i}`} call={c} />
                    ))
                  : null}
                {m.role === "user" || m.text || m.diff ? (
                <div
                  className={
                    m.role === "user"
                      ? "ml-8 rounded-lg bg-muted px-3 py-2 text-sm"
                      : m.diff
                        ? "rounded-lg border px-3 py-2 text-sm"
                        : "mr-8 rounded-lg border px-3 py-2 text-sm"
                  }
                >
                  {m.role === "user" ? (
                    <p className="whitespace-pre-wrap">{m.text}</p>
                  ) : (
                    <AssistantMarkdown text={m.text} />
                  )}
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
                ) : null}
              </Fragment>
            ))
          )}
          {sending ? (
            <p className="text-xs text-muted-foreground">Working…</p>
          ) : null}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <InputGroup className="flex-1">
          <InputGroupAddon align="inline-start">
            <BotIcon />
          </InputGroupAddon>
          <InputGroupInput
            ref={inputRef}
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
          {sending ? "Working…" : "Ask AI"}
        </Button>
      </div>

      <AlertDialog open={confirmClear} onOpenChange={setConfirmClear}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Clear chat?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the conversation and any unapplied edits in this
              panel. Applied changes stay on the config set.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={handleConfirmClear}>
              Clear
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
