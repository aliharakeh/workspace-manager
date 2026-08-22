import { mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { dataDir } from "@db/paths"
import type {
  AIConfigInfo,
  AIConnectionInfo,
  AIProviderConfig,
} from "@/lib/types"

// AI provider settings live in ai.json next to the SQLite database. Multiple
// OpenAI-compatible (or Google) connections are stored keyed by a normalized
// connection name (so one provider can have several connections), with one
// marked active ("default") for chat. An empty active means no default is
// chosen yet.
//
// v1 files stored a single flat config ({provider, apiKey, ...}) and v2 files
// keyed connections by provider slug; both are migrated on load (the name
// defaults to the provider slug).

export type AIStore = {
  providers: Record<string, AIProviderConfig>
  active: string
}

function aiConfigPath() {
  return join(dataDir(), "ai.json")
}

function emptyStore(): AIStore {
  return { providers: {}, active: "" }
}

function loadRaw(): unknown {
  try {
    return JSON.parse(readFileSync(aiConfigPath(), "utf8"))
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return null
    throw err
  }
}

// migrateLegacy converts the old flat single-provider file into a store with
// one connection (named after its provider) that is also the active one.
export function migrateLegacy(raw: Record<string, unknown>): AIStore {
  const cfg = raw as unknown as AIProviderConfig
  const slug = normalizeProvider(cfg.provider ?? "")
  if (!slug) return emptyStore()
  const connection = { ...cfg }
  delete connection.provider
  delete connection.clearApiKey
  return {
    providers: { [slug]: { provider: slug, ...connection } },
    active: slug,
  }
}

export function loadAIStore(): AIStore {
  const raw = loadRaw()
  if (!raw || typeof raw !== "object") return emptyStore()
  const record = raw as Record<string, unknown>
  if (!("providers" in record)) return migrateLegacy(record)

  const providers: Record<string, AIProviderConfig> = {}
  for (const [name, value] of Object.entries(
    record.providers as Record<string, AIProviderConfig>
  )) {
    if (!value || typeof value !== "object") continue
    // v2 files keyed by provider slug stored no inner provider; backfill it
    // from the key so every entry carries the provider it talks to.
    providers[name] = {
      ...value,
      provider: normalizeProvider(value.provider ?? "") || name,
    }
  }
  return { providers, active: String(record.active ?? "") }
}

export function saveAIStore(store: AIStore): void {
  const p = aiConfigPath()
  mkdirSync(dirname(p), { recursive: true })
  writeFileSync(p, JSON.stringify(store, null, 2), { mode: 0o600 })
}

/** What the UI sees per connection: config without the API key, plus hasApiKey. */
export function aiConnectionInfo(
  name: string,
  cfg: AIProviderConfig
): AIConnectionInfo {
  return {
    name,
    provider: cfg.provider,
    baseURL: cfg.baseURL,
    model: cfg.model,
    hasApiKey: Boolean(cfg.apiKey),
    temperature: cfg.temperature,
  }
}

/** Full info payload: every saved connection plus the active name. */
export function aiConfigInfo(store: AIStore): AIConfigInfo {
  const providers = Object.keys(store.providers)
    .sort()
    .map((name) => aiConnectionInfo(name, store.providers[name]!))
  return { providers, active: store.active }
}

/**
 * Upserts one connection under its normalized name. Empty apiKey keeps the
 * stored key; clearApiKey removes it. The first saved connection becomes the
 * active default automatically.
 */
export function upsertAIConnection(store: AIStore, input: AIProviderConfig): void {
  const name = slugify(input.name ?? "")
  if (!name) throw new Error("connection name is required")
  const provider = normalizeProvider(input.provider)
  if (!provider) throw new Error("provider is required")

  const current = store.providers[name]
  let apiKey: string
  if (input.clearApiKey) {
    apiKey = ""
  } else if (input.apiKey?.trim()) {
    apiKey = input.apiKey.trim()
  } else {
    apiKey = current?.apiKey ?? ""
  }

  store.providers[name] = {
    provider,
    baseURL: input.baseURL?.trim() ?? "",
    apiKey,
    model: input.model?.trim() ?? "",
    temperature: input.temperature,
  }
  if (!store.active) store.active = name
}

export function deleteAIConnection(store: AIStore, name: string): boolean {
  const key = slugify(name)
  if (!key || !store.providers[key]) return false
  delete store.providers[key]
  if (store.active === key) store.active = ""
  return true
}

export function activateAIConnection(store: AIStore, name: string): string {
  const key = slugify(name)
  if (!key || !store.providers[key])
    throw new Error(`no connection saved for "${name}"`)
  store.active = key
  return key
}

/** Returns the active connection, or null when no default is configured. */
export function activeAIConnection(store: AIStore): AIProviderConfig | null {
  if (!store.active) return null
  const conn = store.providers[store.active]
  return conn ? { ...conn } : null
}

// slugify turns a user-typed connection name into a URL-safe store key.
export function slugify(s?: string): string {
  return (s ?? "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
}

// normalizeProvider maps the provider names a user might type onto the
// canonical ids used by the plugins. "google" is served by the googlegenai
// plugin; every other provider goes through the OpenAI-compatible plugin.
// Unknown names become URL-safe slugs so they can be used in API paths and as
// store keys.
export function normalizeProvider(p?: string): string {
  const s = (p ?? "").toLowerCase().trim()
  switch (s) {
    case "gemini":
    case "googleai":
    case "google-genai":
      return "google"
    default:
      return slugify(s)
  }
}

// Sensible starting values for well-known OpenAI-compatible providers. An
// empty baseURL means "use the provider's default endpoint" (the OpenAI SDK
// default). An empty model means the model is required from config.
const openAICompatDefaults: Record<string, { baseURL: string; model: string }> =
  {
    opencode: { baseURL: "http://localhost:4096/v1", model: "claude-sonnet-4-20250514" },
    openai: { baseURL: "", model: "gpt-4o-mini" },
    openrouter: { baseURL: "https://openrouter.ai/api/v1", model: "" },
    anthropic: { baseURL: "https://api.anthropic.com/v1", model: "" },
    deepseek: { baseURL: "https://api.deepseek.com/v1", model: "deepseek-chat" },
    xai: { baseURL: "https://api.x.ai/v1", model: "" },
  }

// resolveAIConfig normalizes the provider and applies defaults, returning a
// config that is guaranteed to be buildable (or throwing an actionable error).
export function resolveAIConfig(cfg: AIProviderConfig): AIProviderConfig {
  const p = normalizeProvider(cfg.provider)
  if (!p) throw new Error("no AI provider configured")
  cfg.provider = p

  if (p === "google") {
    cfg.apiKey ||= process.env.GEMINI_API_KEY || ""
    if (!cfg.apiKey)
      throw new Error("google models need an API key (GEMINI_API_KEY)")
    if (!cfg.model) cfg.model = "gemini-2.5-flash"
    return cfg
  }

  if (p === "openai" && !cfg.apiKey) {
    cfg.apiKey = process.env.OPENAI_API_KEY || ""
  }

  const d = openAICompatDefaults[p]
  if (d) {
    if (!cfg.baseURL) cfg.baseURL = d.baseURL
    if (!cfg.model) cfg.model = d.model
  } else if (!cfg.baseURL) {
    throw new Error(
      `provider "${p}" needs a baseURL pointing at an OpenAI-compatible endpoint`
    )
  }
  if (!cfg.model) throw new Error(`no model configured for provider "${p}"`)
  return cfg
}

// oaiBaseURL normalizes a user-supplied OpenAI-compatible base URL so the SDK
// builds the endpoint the user intended. The openai SDK resolves the
// per-request path ("chat/completions") against the base URL, and that
// resolution silently drops the last path segment when the base URL has no
// trailing slash (so ".../v1" would become ".../chat/completions"). Ending the
// root with a slash yields exactly <base>/chat/completions. If the user pasted
// the full endpoint, that suffix is dropped first so it is not doubled.
export function oaiBaseURL(base?: string): string {
  const b = (base ?? "").trim()
  if (!b) return ""
  const trimmed = b.replace(/\/+$/, "")
  const suffix = "/chat/completions"
  return (
    trimmed.toLowerCase().endsWith(suffix)
      ? trimmed.slice(0, trimmed.length - suffix.length)
      : trimmed
  ) + "/"
}
