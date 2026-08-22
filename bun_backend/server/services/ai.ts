import { googleAI } from "@genkit-ai/googleai"
import { openAICompatible } from "@genkit-ai/compat-oai"
import { genkit, type Genkit } from "genkit"
import {
  activeAIConnection,
  aiConfigInfo,
  loadAIStore,
  oaiBaseURL,
  resolveAIConfig,
  type AIProviderConfig,
} from "../lib/ai-config"

// Lazily builds a Genkit runtime for the current active AI connection and
// reuses it until that connection's config changes. Provider plugins may throw
// on invalid credentials, so construction is wrapped into a friendly error.

let cache: {
  cfg: AIProviderConfig
  ai: Genkit
  model: string
} | null = null

function sameConfig(a: AIProviderConfig, b: AIProviderConfig) {
  return (
    a.provider === b.provider &&
    a.baseURL === b.baseURL &&
    a.apiKey === b.apiKey &&
    a.model === b.model &&
    a.temperature === b.temperature
  )
}

function buildAI(cfg: AIProviderConfig): { ai: Genkit; model: string } {
  try {
    if (cfg.provider === "google") {
      return {
        ai: genkit({ plugins: [googleAI({ apiKey: cfg.apiKey })] }),
        model: `googleai/${cfg.model}`,
      }
    }
    return {
      ai: genkit({
        plugins: [
          openAICompatible({
            name: cfg.provider,
            // Empty string (not undefined) so the SDK never falls back to
            // OPENAI_API_KEY for keyless local endpoints.
            apiKey: cfg.apiKey ?? "",
            // Normalize here (not in the stored config) so the base URL is
            // kept exactly as the user entered it while the SDK still builds
            // <base>/chat/completions instead of mangling the last path.
            baseURL: oaiBaseURL(cfg.baseURL),
          }),
        ],
      }),
      // Must match the plugin name exactly: the registry resolves
      // "<provider>/<model>" by stripping this prefix.
      model: `${cfg.provider}/${cfg.model}`,
    }
  } catch (err) {
    throw new Error(`AI provider failed to initialize: ${String(err)}`, {
      cause: err,
    })
  }
}

function ensureAI(cfg: AIProviderConfig): { ai: Genkit; model: string } {
  if (!cache || !sameConfig(cache.cfg, cfg)) {
    const built = buildAI(cfg)
    cache = { cfg, ...built }
  }
  return cache!
}

// runGeneration sends one request through the (cached) runtime for the given
// connection. Both aiChat and aiTest go through here.
async function runGeneration(
  cfg: AIProviderConfig,
  system?: string,
  prompt?: string
): Promise<string> {
  const resolved = resolveAIConfig(cfg)
  const { ai, model } = ensureAI(resolved)
  const res = await ai.generate({
    model,
    prompt: prompt ?? "",
    ...(system ? { system } : {}),
    ...(resolved.provider !== "google" && resolved.temperature != null
      ? { config: { temperature: resolved.temperature } }
      : {}),
  })
  return res.text
}

// aiChat runs one generation against the active connection and returns the
// generated text. It loads the persisted store and (re)builds the runtime when
// the active connection's config changed.
export async function aiChat(system?: string, prompt?: string): Promise<string> {
  if (!prompt?.trim()) throw new Error("prompt is required")
  const conn = activeAIConnection(loadAIStore())
  if (!conn)
    throw new Error(
      "no default AI connection configured — pick one in Settings → AI connection"
    )
  return runGeneration(conn, system, prompt)
}

// aiTest tries one minimal generation against an unsaved connection payload.
// It applies the same defaults and validation as real use, so it fails exactly
// when saving and chatting would.
export async function aiTest(cfg: AIProviderConfig): Promise<string> {
  if (!cfg.provider?.trim()) throw new Error("provider is required")
  return runGeneration(cfg, undefined, "Reply with exactly: OK")
}

// aiInfo is the config payload served to the UI (no secrets).
export function aiInfo() {
  return aiConfigInfo(loadAIStore())
}
