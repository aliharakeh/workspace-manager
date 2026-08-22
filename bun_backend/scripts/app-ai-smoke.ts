import {
  APP_AI_SYSTEM_PROMPT,
  buildAppAIPrompt,
  parseAppAIResponse,
  patchHasEdits,
  stripAIFences,
} from "../../frontend/lib/app-ai"
import {
  appAIDiffCount,
  buildAppAIDiff,
  normalizeNewlines,
} from "../../frontend/lib/app-ai-diff"
import type { ConfigSetDetail } from "../../frontend/lib/types"

let failures = 0
function check(name: string, got: unknown, want: unknown) {
  const ok = JSON.stringify(got) === JSON.stringify(want)
  console.log(`${ok ? "ok  " : "FAIL"} ${name}`)
  if (!ok) {
    console.log(`     got:  ${JSON.stringify(got)}`)
    console.log(`     want: ${JSON.stringify(want)}`)
    failures++
    process.exitCode = 1
  }
}

const detail: ConfigSetDetail = {
  id: 7,
  app_id: 1,
  name: "local",
  created_at: "",
  updated_at: "",
  env_vars: [
    {
      id: 1,
      config_set_id: 7,
      key: "PORT",
      value: "3000",
      created_at: "",
      updated_at: "",
    },
  ],
  templates: [
    {
      id: 2,
      config_set_id: 7,
      file_path: ".env",
      content: "PORT=3000",
      created_at: "",
      updated_at: "",
    },
  ],
  run_config: {
    id: 3,
    config_set_id: 7,
    mode: "parallel",
    created_at: "",
    updated_at: "",
    commands: [
      {
        id: 4,
        run_config_id: 3,
        label: "web",
        command: "npm run dev",
        sort_order: 0,
        created_at: "",
        updated_at: "",
      },
    ],
  },
}

check("strip fenced json", stripAIFences('```json\n{"message":"ok"}\n```'), '{"message":"ok"}')
check(
  "parse fenced patch",
  parseAppAIResponse(
    '```json\n{"message":"set port","env":{"upsert":[{"key":"PORT","value":"5173"}]}}\n```'
  ),
  { message: "set port", env: { upsert: [{ key: "PORT", value: "5173" }] } }
)
check(
  "parse ignores unknown template fields",
  parseAppAIResponse(
    '{"message":"hi","templates":[{"file_path":".env","content":"PORT={{PORT}}"}]}'
  ),
  {
    message: "hi",
    templates: [{ file_path: ".env", content: "PORT={{PORT}}" }],
  }
)
check(
  "parse run commands",
  parseAppAIResponse(
    '{"message":"run","run":{"mode":"sequential","commands":[{"label":"api","command":"go run ."}]}}'
  ),
  {
    message: "run",
    run: { mode: "sequential", commands: [{ label: "api", command: "go run ." }] },
  }
)
check(
  "non-json is a message only",
  parseAppAIResponse("just an answer"),
  { message: "just an answer" }
)
check("empty patch has no edits", patchHasEdits({ message: "ok" }), false)
check(
  "env patch has edits",
  patchHasEdits({ message: "ok", env: { delete: ["X"] } }),
  true
)

const prompt = buildAppAIPrompt({
  appName: "shop",
  projectPath: "/src/shop",
  configSet: detail,
  history: [{ role: "user", text: "what is PORT?" }],
  instruction: "set PORT to 5173",
})
check("prompt names active set only", prompt.includes("name: local") && prompt.includes("id: 7"), true)
check("prompt includes env", prompt.includes("PORT=3000"), true)
check("prompt includes template", prompt.includes("file_path: .env"), true)
check("prompt includes run command", prompt.includes("npm run dev"), true)
check("prompt includes prior chat", prompt.includes("what is PORT?"), true)
check("prompt includes instruction", prompt.includes("set PORT to 5173"), true)
check("system forbids other sets", APP_AI_SYSTEM_PROMPT.includes("any other config set"), true)

const filtered = buildAppAIPrompt({
  appName: "shop",
  projectPath: "/src/shop",
  configSet: {
    ...detail,
    templates: [
      detail.templates[0]!,
      { ...detail.templates[0]!, id: 9, file_path: "other.env", content: "SECRET=1" },
    ],
  },
  history: [],
  instruction: "go",
  templatePaths: [".env"],
})
check("prompt omits unselected template content", filtered.includes("SECRET=1"), false)
check("prompt lists omitted template path", filtered.includes("other.env"), true)
check(
  "empty selection sends no template content",
  buildAppAIPrompt({
    appName: "shop",
    projectPath: "/src/shop",
    configSet: detail,
    history: [],
    instruction: "go",
    templatePaths: [],
  }).includes("PORT=3000\n\nTemplates:\n(none)"),
  true
)

const envDiff = buildAppAIDiff(detail, {
  message: "x",
  env: { upsert: [{ key: "PORT", value: "5173" }, { key: "HOST", value: "localhost" }], delete: ["MISSING"] },
})
check("env file old", envDiff.files[0]?.oldText, "PORT=3000")
check("env file new", envDiff.files[0]?.newText, "PORT=5173\nHOST=localhost")
check("env skip missing delete", envDiff.skipped.length, 0)

const tmplDiff = buildAppAIDiff(detail, {
  message: "x",
  templates: [
    { file_path: ".env", content: "PORT={{PORT}}" },
    { file_path: "nope.txt", content: "x" },
  ],
})
check("template file diff", tmplDiff.files[0], {
  path: ".env",
  oldText: "PORT=3000",
  newText: "PORT={{PORT}}",
})
check("unknown template skipped", tmplDiff.skipped, ["nope.txt"])

const runDiff = buildAppAIDiff(detail, {
  message: "x",
  run: { mode: "sequential", commands: [{ label: "api", command: "go run ." }] },
})
check("run file old", runDiff.files[0]?.oldText, "mode: parallel\n1. web: npm run dev")
check("run file new", runDiff.files[0]?.newText, "mode: sequential\n1. api: go run .")
check("noop env not counted", appAIDiffCount(buildAppAIDiff(detail, { message: "x", env: { upsert: [{ key: "PORT", value: "3000" }] } })), 0)
check(
  "crlf matches lf",
  normalizeNewlines("a\r\nb\r\n"),
  normalizeNewlines("a\nb\n")
)

const crlfOnly = buildAppAIDiff(
  {
    ...detail,
    templates: [{ ...detail.templates[0]!, content: "PORT=3000\r\n" }],
  },
  { message: "x", templates: [{ file_path: ".env", content: "PORT=3000\n" }] }
)
check("line-ending-only template omitted", crlfOnly.files.length, 0)

console.log(failures === 0 ? "\nAll app AI tests passed!" : `\n${failures} FAILURES`)
