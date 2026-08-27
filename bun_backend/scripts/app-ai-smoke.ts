import {
  APP_AI_SYSTEM_PROMPT,
  buildAppAIPrompt,
  parseAppAIResponse,
  patchHasEdits,
  stripAIFences,
  summarizeAIToolInput,
} from "../../frontend/lib/app-ai"
import {
  appAIDiffCount,
  buildAppAIDiff,
  normalizeNewlines,
} from "../../frontend/lib/app-ai-diff"
import { mkdirSync, writeFileSync, rmSync } from "node:fs"
import { join } from "node:path"
import { spawnSync } from "node:child_process"
import { AppAIAgentState } from "../server/lib/app-ai-agent"
import { globToRegExp, searchProjectFiles } from "../server/lib/gitignore-glob"

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
      include_in_ai: true,
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
check("tool input summary truncates", summarizeAIToolInput({ content: "x".repeat(80) }).endsWith("…"), true)
check("empty object input summary", summarizeAIToolInput({}), "")
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
check("prompt omits env dump", prompt.includes("PORT=3000"), false)
check("prompt omits template dump", prompt.includes("file_path: .env"), false)
check("prompt omits run dump", prompt.includes("npm run dev"), false)
check("prompt includes prior chat", prompt.includes("what is PORT?"), true)
check("prompt includes instruction", prompt.includes("set PORT to 5173"), true)
check("system forbids other sets", APP_AI_SYSTEM_PROMPT.includes("any other config set"), true)

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

const agent = new AppAIAgentState(detail, ".")
check("agent starts with no tool calls", agent.toolCalls.length, 0)
check("list vars", agent.listVars(), [{ key: "PORT", value: "3000" }])

const hiddenAgent = new AppAIAgentState(
  {
    ...detail,
    env_vars: [
      detail.env_vars[0]!,
      {
        id: 9,
        config_set_id: 7,
        key: "SECRET",
        value: "shh",
        include_in_ai: false,
        created_at: "",
        updated_at: "",
      },
    ],
  },
  "."
)
check("hidden listed without value", hiddenAgent.listVars(), [
  { key: "PORT", value: "3000" },
  { key: "SECRET" },
])
check("hidden get is not found", hiddenAgent.getVar("SECRET"), {
  error: "env var not found: SECRET",
})
check("hidden update is not found", hiddenAgent.updateVar("SECRET", "x"), {
  error: "env var not found: SECRET",
})
check("hidden delete is not found", hiddenAgent.deleteVar("SECRET"), {
  error: "env var not found: SECRET",
})
check("hidden not in patch after update attempt", hiddenAgent.patch().env, undefined)
agent.updateVar("PORT", "5173")
agent.updateVar("HOST", "localhost")
check("get after update", agent.getVar("PORT"), { key: "PORT", value: "5173" })
const envPatch = agent.patch()
check(
  "agent env patch",
  (envPatch.env?.upsert ?? []).slice().sort((a, b) => a.key.localeCompare(b.key)),
  [
    { key: "HOST", value: "localhost" },
    { key: "PORT", value: "5173" },
  ]
)
agent.deleteVar("HOST")
check("delete new key leaves no upsert", agent.patch().env?.upsert, [{ key: "PORT", value: "5173" }])
agent.updateTemplate(".env", "PORT={{PORT}}")
check("agent template patch", agent.patch().templates, [
  { file_path: ".env", content: "PORT={{PORT}}" },
])
check("unknown template rejected", agent.updateTemplate("nope", "x").error, "template not on this config set: nope")
agent.updateRun({ mode: "sequential", commands: [{ label: "api", command: "go run ." }] })
check("agent run patch mode", agent.patch().run?.mode, "sequential")

check("glob **/*.env", globToRegExp("**/*.env").test("a/.env") && globToRegExp("**/*.env").test(".env"), true)
check("glob rejects parent", "error" in (searchProjectFiles(".", "../x") as object), true)

const tmp = join(process.cwd(), ".tmp-app-ai-glob")
rmSync(tmp, { recursive: true, force: true })
mkdirSync(join(tmp, "src"), { recursive: true })
writeFileSync(join(tmp, ".gitignore"), "secret.txt\n")
writeFileSync(join(tmp, "keep.ts"), "a")
writeFileSync(join(tmp, "secret.txt"), "no")
writeFileSync(join(tmp, "src", "app.ts"), "b")
const gitInit = spawnSync("git", ["-C", tmp, "init"], { encoding: "utf8", windowsHide: true })
if (gitInit.status === 0) {
  const found = searchProjectFiles(tmp, "**/*.ts")
  if ("error" in found) {
    check("glob git search", found, { files: ["keep.ts", "src/app.ts"] })
  } else {
    check(
      "glob git search",
      found.files.slice().sort(),
      ["keep.ts", "src/app.ts"]
    )
    check("glob omits gitignored", found.files.includes("secret.txt"), false)
  }
} else {
  console.log("skip glob git test (git init failed)")
}
const reader = new AppAIAgentState(detail, tmp)
check("read_file", reader.readFile("src/app.ts"), {
  file_path: "src/app.ts",
  content: "b",
})
check("read_file escape", "error" in reader.readFile("../package.json"), true)
rmSync(tmp, { recursive: true, force: true })

console.log(failures === 0 ? "\nAll app AI tests passed!" : `\n${failures} FAILURES`)
