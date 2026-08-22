// Smoke test for the AI setup: store operations, config resolution, URL
// normalization, and runtime construction (no network calls). Run from repo
// root: bun run bun_backend/scripts/ai-smoke.ts
import {
  aiConfigInfo,
  activateAIConnection,
  deleteAIConnection,
  loadAIStore,
  migrateLegacy,
  normalizeProvider,
  oaiBaseURL,
  resolveAIConfig,
  saveAIStore,
  upsertAIConnection,
  type AIStore,
} from "../server/lib/ai-config"
import { aiChat } from "../server/services/ai"
import { existsSync, rmSync } from "node:fs"
import { join } from "node:path"
import { dataDir } from "@db/paths"

function aiConfigFile() {
  return join(dataDir(), "ai.json")
}

let failures = 0
function check(name: string, got: unknown, want: unknown) {
  const ok = JSON.stringify(got) === JSON.stringify(want)
  console.log(`${ok ? "ok  " : "FAIL"} ${name}: ${JSON.stringify(got)}`)
  if (!ok) {
    console.log(`     wanted ${JSON.stringify(want)}`)
    failures++
    process.exitCode = 1
  }
}

check("base url adds slash", oaiBaseURL("https://openrouter.ai/api/v1"), "https://openrouter.ai/api/v1/")
check("base url keeps trailing slash", oaiBaseURL("http://localhost:4096/v1/"), "http://localhost:4096/v1/")
check(
  "base url drops full endpoint",
  oaiBaseURL("https://api.deepseek.com/v1/chat/completions"),
  "https://api.deepseek.com/v1/"
)
check("base url empty", oaiBaseURL(""), "")
check("base url undefined", oaiBaseURL(undefined), "")

check("alias gemini -> google", normalizeProvider("Gemini"), "google")
check("slugify unknown provider", normalizeProvider("Local LLM!"), "local-llm")

check(
  "resolve google defaults",
  resolveAIConfig({ provider: "google", apiKey: "k" }),
  { provider: "google", apiKey: "k", model: "gemini-2.5-flash" }
)
try {
  resolveAIConfig({ provider: "openrouter" })
  console.log("FAIL openrouter without model should throw")
  failures++
} catch (err) {
  check(
    "openrouter without model throws",
    err instanceof Error ? err.message : err,
    'no model configured for provider "openrouter"'
  )
}
try {
  resolveAIConfig({ provider: "custom", model: "m" })
  console.log("FAIL custom without baseURL should throw")
  failures++
} catch (err) {
  check(
    "custom without baseURL throws",
    err instanceof Error ? err.message : err,
    'provider "custom" needs a baseURL pointing at an OpenAI-compatible endpoint'
  )
}

// --- Store operations (synthetic store so a real ai.json never interferes) ---
const store: AIStore = { providers: {}, active: "" }
upsertAIConnection(store, { name: "Work Router", provider: "OpenRouter", apiKey: "k1" })
upsertAIConnection(store, { name: "google", provider: "google", apiKey: "" })
check("first connection becomes active", store.active, "work-router")
check("provider normalized on save", store.providers["work-router"]?.provider, "openrouter")

// One provider can back several named connections.
upsertAIConnection(store, { name: "openrouter-personal", provider: "OpenRouter", apiKey: "k3" })
check("second connection same provider", Object.keys(store.providers).length, 3)

upsertAIConnection(store, { name: "work-router", provider: "openrouter", baseURL: "https://x/v1" })
check(
  "empty key keeps stored key",
  store.providers["work-router"]?.apiKey,
  "k1"
)
upsertAIConnection(store, { name: "work-router", provider: "openrouter", clearApiKey: true })
check("clearApiKey wipes key", store.providers["work-router"]?.apiKey, "")

activateAIConnection(store, "google")
check("activate switches default", store.active, "google")
deleteAIConnection(store, "google")
check("deleting active clears it", store.active, "")
try {
  activateAIConnection(store, "google")
  console.log("FAIL activating missing should throw")
  failures++
} catch (err) {
  check(
    "activating missing throws",
    err instanceof Error ? err.message : err
  , 'no connection saved for "google"')
}

const info = aiConfigInfo(store)
check("info has no secrets", info.providers.every((p) => !("apiKey" in p)), true)
check("info carries names", info.providers.map((p) => p.name).sort(), ["openrouter-personal", "work-router"])

// Legacy single-provider file migrates to a one-connection store.
const legacy = migrateLegacy({ provider: "Gemini", apiKey: "legacy", model: "m" })
check(
  "legacy migration",
  { active: legacy.active, providers: Object.keys(legacy.providers) },
  { active: "google", providers: ["google"] }
)
check(
  "legacy migration keeps key+model",
  legacy.providers["google"],
  { provider: "google", apiKey: "legacy", model: "m" }
)

// Exercise the full path: store -> resolve -> genkit() + plugin init ->
// model resolution -> request against the local opencode endpoint. Only run
// when no real config exists that this section would clobber and delete.
if (!existsSync(aiConfigFile())) {
  const probe = loadAIStore()
  upsertAIConnection(probe, { name: "opencode", provider: "opencode", model: "smoke-test-model" })
  saveAIStore(probe)
  try {
    await aiChat("system", "ping")
    console.log("ok   local generation succeeded (unexpected but fine)")
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    const networkish = /fetch failed|connect|econnrefused|refused|status|api key|401/i
    console.log(
      `${networkish.test(msg) ? "ok  " : "FAIL"} runtime built; request failed as expected: ${msg}`
    )
    if (!networkish.test(msg)) failures++
  } finally {
    rmSync(aiConfigFile(), { force: true })
  }

  try {
    await aiChat(undefined, "ping")
    console.log("FAIL chat without default should throw")
    failures++
  } catch (err) {
    check(
      "chat without default rejected",
      err instanceof Error ? err.message : err,
      "no default AI connection configured — pick one in Settings → AI connection"
    )
  }
} else {
  console.log("skip local generation test (a real config exists)")
}

console.log(failures === 0 ? "\nall checks passed" : `\n${failures} FAILURES`)
