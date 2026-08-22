import { useEffect, useState } from "react"
import {
  BotIcon,
  KeyRoundIcon,
  PencilIcon,
  StarIcon,
  Trash2Icon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { slugify } from "@/lib/routes"
import type { AIConfigInfo, AIConnectionInfo } from "@/lib/types"
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
import { Checkbox } from "@/components/ui/checkbox"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"

export type AIPanelProps = {
  active: boolean
}

type Preset = {
  value: string
  label: string
  /** Shown as the model placeholder; empty means "required, no default". */
  model: string
  /** Google takes no base URL; everything else does. */
  noBaseUrl?: boolean
}

const PRESETS: Preset[] = [
  { value: "openrouter", label: "OpenRouter", model: "" },
  {
    value: "google",
    label: "Google Gemini",
    model: "gemini-2.5-flash",
    noBaseUrl: true,
  },
  { value: "openai", label: "OpenAI", model: "gpt-4o-mini" },
  { value: "anthropic", label: "Anthropic", model: "" },
  { value: "deepseek", label: "DeepSeek", model: "deepseek-chat" },
  { value: "xai", label: "xAI (Grok)", model: "" },
  {
    value: "opencode",
    label: "OpenCode (local)",
    model: "claude-sonnet-4-20250514",
  },
  { value: "custom", label: "Custom endpoint…", model: "" },
]

function presetOf(provider: string): Preset {
  return (
    PRESETS.find((p) => p.value === provider) ?? {
      value: provider,
      label: provider,
      model: "",
    }
  )
}

type FormState = {
  name: string
  provider: string
  baseUrl: string
  apiKey: string
  model: string
  temperature: string
  clearApiKey: boolean
}

function emptyForm(): FormState {
  return {
    name: "",
    provider: "",
    baseUrl: "",
    apiKey: "",
    model: "",
    temperature: "",
    clearApiKey: false,
  }
}

function formFor(conn: AIConnectionInfo): FormState {
  return {
    name: conn.name,
    provider: conn.provider,
    baseUrl: conn.baseURL ?? "",
    apiKey: "",
    model: conn.model ?? "",
    temperature: conn.temperature != null ? String(conn.temperature) : "",
    clearApiKey: false,
  }
}

function ConnectionRow({
  conn,
  isActive,
  onEdit,
  onActivate,
  onRequestDelete,
  activating,
}: {
  conn: AIConnectionInfo
  isActive: boolean
  onEdit: () => void
  onActivate: () => void
  onRequestDelete: () => void
  activating: boolean
}) {
  const preset = presetOf(conn.provider)
  return (
    <div className="flex items-center gap-3 rounded-lg border px-3 py-2.5">
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{conn.name}</span>
          {isActive ? (
            <Badge>
              <StarIcon data-icon="inline-start" />
              Default
            </Badge>
          ) : null}
          <Badge variant="secondary" className="font-mono text-xs">
            {preset.label}
          </Badge>
          {conn.model ? (
            <Badge variant="secondary" className="font-mono">
              {conn.model}
            </Badge>
          ) : null}
        </div>
        <div className="mt-0.5 flex items-center gap-3 truncate font-mono text-xs font-normal text-muted-foreground">
          {conn.baseURL ? (
            <span className="truncate">{conn.baseURL}</span>
          ) : null}
          <span className="flex shrink-0 items-center gap-1">
            <KeyRoundIcon className="size-3" />
            {conn.hasApiKey ? "key stored" : "no key"}
          </span>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {!isActive ? (
          <Button
            size="sm"
            variant="secondary"
            disabled={activating}
            onClick={onActivate}
          >
            Set default
          </Button>
        ) : null}
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Edit ${conn.name}`}
          onClick={onEdit}
        >
          <PencilIcon />
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Remove ${conn.name}`}
          onClick={onRequestDelete}
        >
          <Trash2Icon />
        </Button>
      </div>
    </div>
  )
}

export function AIPanel({ active }: AIPanelProps) {
  const [info, setInfo] = useState<AIConfigInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState<FormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [activating, setActivating] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<AIConnectionInfo | null>(
    null
  )
  const [deleting, setDeleting] = useState(false)

  async function load() {
    setLoading(true)
    try {
      setInfo(await api.ai.getConfig())
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load AI connections"
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (active) void load()
    else {
      setForm(emptyForm())
      setPendingDelete(null)
      setTesting(false)
    }
  }, [active]) // eslint-disable-line react-hooks/exhaustive-deps

  // A saved connection with this name exists -> the form edits it (its name
  // stays fixed); otherwise the form adds a new connection.
  const savedConn =
    form.name !== ""
      ? info?.providers.find((p) => p.name === form.name)
      : undefined

  function handleProviderSelect(value: string | null) {
    const provider = value ?? ""
    if (!provider) {
      setForm(emptyForm())
      return
    }
    // Start a fresh form for the chosen provider; saved connections are
    // edited through their rows below. Suggest a name that is not taken yet
    // so several connections can share one provider.
    let name = provider === "custom" ? "" : provider
    if (name && info?.providers.some((p) => p.name === name)) {
      let n = 2
      while (info.providers.some((p) => p.name === `${name}-${n}`)) n++
      name = `${name}-${n}`
    }
    setForm({ ...emptyForm(), provider, name })
  }

  async function handleSave() {
    if (!form.provider) {
      toast.error("Pick a provider first")
      return
    }
    if (!form.name.trim()) {
      toast.error("Give this connection a name")
      return
    }
    let temperature: number | undefined
    if (form.temperature.trim()) {
      temperature = Number(form.temperature)
      if (!Number.isFinite(temperature) || temperature < 0 || temperature > 2) {
        toast.error("Temperature must be between 0 and 2")
        return
      }
    }
    setSaving(true)
    try {
      const next = await api.ai.saveConfig({
        name: form.name,
        provider: form.provider,
        baseURL: form.baseUrl.trim() || undefined,
        // Empty keeps the stored key; clearApiKey removes it.
        apiKey: form.apiKey.trim() || undefined,
        model: form.model.trim() || undefined,
        temperature,
        clearApiKey: form.clearApiKey,
      })
      setInfo(next)
      // The backend stores under the slugified name; match it for re-fill.
      const savedName = slugify(form.name.trim())
      const saved = next.providers.find((p) => p.name === savedName)
      setForm(saved ? formFor(saved) : emptyForm())
      toast.success("Connection saved")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  async function handleTest() {
    if (!form.provider) {
      toast.error("Pick a provider first")
      return
    }
    setTesting(true)
    try {
      const res = await api.ai.test({
        provider: form.provider,
        baseURL: form.baseUrl.trim() || undefined,
        apiKey: form.apiKey.trim() || undefined,
        model: form.model.trim() || undefined,
      })
      toast.success(
        `Connection works — model replied: ${res.text.slice(0, 80)}`
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Connection test failed",
        { duration: 8000 }
      )
    } finally {
      setTesting(false)
    }
  }

  async function handleActivate(name: string) {
    setActivating(name)
    try {
      setInfo(await api.ai.activate(name))
      toast.success(`${name} is now the default`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to activate")
    } finally {
      setActivating(null)
    }
  }

  async function handleConfirmDelete() {
    if (!pendingDelete) return
    setDeleting(true)
    try {
      setInfo(await api.ai.deleteConfig(pendingDelete.name))
      setForm((prev) =>
        prev.name === pendingDelete.name ? emptyForm() : prev
      )
      setPendingDelete(null)
      toast.success("Connection removed")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove")
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    )
  }

  const preset = form.provider ? presetOf(form.provider) : null
  const providerItems = PRESETS.map((p) => ({
    value: p.value,
    label: `${p.label}${
      info?.providers.some((c) => c.provider === p.value) ? " (saved)" : ""
    }`,
  }))

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col gap-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium">AI connection</p>
            {info ? (
              <Badge variant="secondary">{info.providers.length}</Badge>
            ) : null}
          </div>
          <p className="text-sm text-muted-foreground">
            Connect AI providers and pick the default used by AI features.
          </p>
        </div>

        <ScrollArea className="min-h-0 flex-1">
          <div className="flex flex-col gap-4 pr-3">
            <FieldGroup className="rounded-lg border bg-muted/30 p-3">
              <Field>
                <FieldLabel htmlFor="ai-provider">Provider</FieldLabel>
                <Select
                  items={providerItems}
                  value={form.provider || null}
                  onValueChange={(value) => handleProviderSelect(value)}
                >
                  <SelectTrigger id="ai-provider" className="w-full">
                    <SelectValue placeholder="Choose a provider…" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {PRESETS.map((p) => {
                        const connected = info?.providers.some(
                          (c) => c.provider === p.value
                        )
                        return (
                          <SelectItem key={p.value} value={p.value}>
                            {p.label}
                            {connected ? " (saved)" : ""}
                          </SelectItem>
                        )
                      })}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                {savedConn ? (
                  <FieldDescription>
                    Editing the saved “{savedConn.name}” connection.
                  </FieldDescription>
                ) : null}
              </Field>

              {form.provider && preset ? (
                <>
                  <Field>
                    <FieldLabel htmlFor="ai-name">Name</FieldLabel>
                    <Input
                      id="ai-name"
                      value={form.name}
                      disabled={!!savedConn}
                      onChange={(e) =>
                        setForm((prev) => ({ ...prev, name: e.target.value }))
                      }
                      placeholder="e.g. openrouter-work"
                    />
                    <FieldDescription>
                      {savedConn
                        ? "Names identify connections and cannot be changed."
                        : "Uniquely identifies this connection — one provider can have several."}
                    </FieldDescription>
                  </Field>
                  {!preset.noBaseUrl ? (
                    <Field>
                      <FieldLabel htmlFor="ai-base-url">Base URL</FieldLabel>
                      <Input
                        id="ai-base-url"
                        value={form.baseUrl}
                        onChange={(e) =>
                          setForm((prev) => ({
                            ...prev,
                            baseUrl: e.target.value,
                          }))
                        }
                        placeholder={
                          preset.value === "openrouter"
                            ? "https://openrouter.ai/api/v1"
                            : preset.value === "opencode"
                              ? "http://localhost:4096/v1"
                              : "Default endpoint"
                        }
                      />
                      <FieldDescription>
                        Any OpenAI-compatible chat completions endpoint.
                      </FieldDescription>
                    </Field>
                  ) : null}

                  <Field>
                    <FieldLabel htmlFor="ai-model">Model</FieldLabel>
                    <Input
                      id="ai-model"
                      value={form.model}
                      onChange={(e) =>
                        setForm((prev) => ({ ...prev, model: e.target.value }))
                      }
                      placeholder={
                        preset.model || "e.g. meta-llama/llama-3.1-70b-instruct"
                      }
                    />
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="ai-api-key">API key</FieldLabel>
                    <Input
                      id="ai-api-key"
                      type="password"
                      autoComplete="off"
                      value={form.apiKey}
                      onChange={(e) =>
                        setForm((prev) => ({ ...prev, apiKey: e.target.value }))
                      }
                      placeholder={
                        savedConn?.hasApiKey
                          ? "•••••••• (stored)"
                          : "Paste API key"
                      }
                    />
                    <FieldDescription>
                      {savedConn?.hasApiKey
                        ? "A key is already stored. Leave blank to keep it."
                        : preset.value === "google"
                          ? "Or provide it via the GEMINI_API_KEY environment variable."
                          : "Stored locally in plain text next to the app database."}
                    </FieldDescription>
                    {savedConn?.hasApiKey ? (
                      <Field orientation="horizontal" className="gap-2">
                        <Checkbox
                          id="ai-clear-key"
                          checked={form.clearApiKey}
                          onCheckedChange={(checked) =>
                            setForm((prev) => ({
                              ...prev,
                              clearApiKey: checked === true,
                            }))
                          }
                        />
                        <FieldLabel
                          htmlFor="ai-clear-key"
                          className="text-xs font-normal text-muted-foreground"
                        >
                          Remove stored key
                        </FieldLabel>
                      </Field>
                    ) : null}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="ai-temperature">
                      Temperature
                    </FieldLabel>
                    <Input
                      id="ai-temperature"
                      type="number"
                      min={0}
                      max={2}
                      step={0.1}
                      value={form.temperature}
                      onChange={(e) =>
                        setForm((prev) => ({
                          ...prev,
                          temperature: e.target.value,
                        }))
                      }
                      placeholder="Provider default"
                    />
                  </Field>

                  <div className="flex justify-end gap-2">
                    <Button
                      variant="outline"
                      disabled={saving || testing}
                      onClick={() => void handleTest()}
                    >
                      {testing ? "Testing…" : "Test connection"}
                    </Button>
                    <Button
                      disabled={saving || testing}
                      onClick={() => void handleSave()}
                    >
                      {saving
                        ? "Saving…"
                        : savedConn
                          ? "Save changes"
                          : "Add connection"}
                    </Button>
                  </div>
                </>
              ) : null}
            </FieldGroup>

            {info && info.providers.length > 0 ? (
              <div className="flex flex-col gap-2">
                {info.providers.map((conn) => (
                  <ConnectionRow
                    key={conn.name}
                    conn={conn}
                    isActive={info.active === conn.name}
                    activating={activating === conn.name}
                    onActivate={() => void handleActivate(conn.name)}
                    onEdit={() => setForm(formFor(conn))}
                    onRequestDelete={() => setPendingDelete(conn)}
                  />
                ))}
              </div>
            ) : (
              <Empty className="border">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <BotIcon />
                  </EmptyMedia>
                  <EmptyTitle>No connections yet</EmptyTitle>
                  <EmptyDescription>
                    Pick a provider above, give the connection a name, paste
                    its API key, and save. The first connection becomes the
                    default automatically.
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            )}

            {info && info.providers.length > 1 && !info.active ? (
              <p className="text-xs text-muted-foreground">
                No default selected — choose one with “Set default”.
              </p>
            ) : null}
          </div>
        </ScrollArea>
      </div>

      <AlertDialog
        open={!!pendingDelete}
        onOpenChange={(next) => {
          if (!next && !deleting) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove connection?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the saved settings and API key for this provider. If
              it was the default you will need to pick another one before using
              AI features.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingDelete ? (
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 rounded-lg border bg-muted/40 px-3 py-2.5 text-sm">
              <dt className="text-muted-foreground">Name</dt>
              <dd className="truncate font-medium">{pendingDelete.name}</dd>
              <dt className="text-muted-foreground">Provider</dt>
              <dd className="truncate font-mono text-xs">
                {presetOf(pendingDelete.provider).label}
              </dd>
              {pendingDelete.model ? (
                <>
                  <dt className="text-muted-foreground">Model</dt>
                  <dd className="truncate font-mono text-xs">
                    {pendingDelete.model}
                  </dd>
                </>
              ) : null}
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
              {deleting ? "Removing…" : "Remove connection"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
