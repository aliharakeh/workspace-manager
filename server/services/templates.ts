import Handlebars from "handlebars"
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { dirname, join, normalize, resolve, sep } from "node:path"
import { appsRepo } from "../db/apps"
import { configSetsRepo } from "../db/config-sets"
import { envVarsRepo } from "../db/env-vars"
import { templatesRepo } from "../db/templates"

function backupRoot(appId: number, sessionId: string) {
  return join(process.cwd(), "data", "backups", String(appId), sessionId)
}

function resolveSafePath(projectPath: string, relativePath: string): string {
  const cleaned = relativePath.replace(/\\/g, "/").replace(/^\/+/, "")
  const full = resolve(projectPath, cleaned)
  const root = resolve(projectPath)
  const normalizedFull = normalize(full)
  const normalizedRoot = normalize(root)
  if (
    normalizedFull !== normalizedRoot &&
    !normalizedFull.startsWith(normalizedRoot + sep)
  ) {
    throw new Error(`Template path escapes project directory: ${relativePath}`)
  }
  return full
}

export type TemplateApplyResult = {
  applied: string[]
  backupDir: string
}

export function applyTemplates(
  appId: number,
  sessionId: string
): TemplateApplyResult {
  const app = appsRepo.get(appId)
  if (!app) throw new Error("App not found")

  const set = configSetsRepo.resolveActive(appId)
  const templates = templatesRepo.listByConfigSet(set.id)
  const env = envVarsRepo.toRecord(set.id)
  const backupDir = backupRoot(appId, sessionId)
  const applied: string[] = []

  mkdirSync(backupDir, { recursive: true })

  try {
    for (const template of templates) {
      const targetPath = resolveSafePath(app.project_path, template.file_path)
      if (!existsSync(targetPath)) {
        throw new Error(`Target file does not exist: ${template.file_path}`)
      }

      const backupPath = join(backupDir, template.file_path.replace(/\\/g, "/"))
      mkdirSync(dirname(backupPath), { recursive: true })
      cpSync(targetPath, backupPath)

      const compiled = Handlebars.compile(template.content, { noEscape: true })
      const rendered = compiled(env)
      writeFileSync(targetPath, rendered, "utf8")
      applied.push(template.file_path)
    }
  } catch (err) {
    restoreTemplates(appId, sessionId)
    throw err
  }

  return { applied, backupDir }
}

export function restoreTemplates(appId: number, sessionId: string): string[] {
  const app = appsRepo.get(appId)
  if (!app) return []

  const backupDir = backupRoot(appId, sessionId)
  if (!existsSync(backupDir)) return []

  const restored: string[] = []
  const set = configSetsRepo.resolveActive(appId)
  const templates = templatesRepo.listByConfigSet(set.id)

  for (const template of templates) {
    const backupPath = join(backupDir, template.file_path.replace(/\\/g, "/"))
    if (!existsSync(backupPath)) continue
    const targetPath = resolveSafePath(app.project_path, template.file_path)
    mkdirSync(dirname(targetPath), { recursive: true })
    const content = readFileSync(backupPath)
    writeFileSync(targetPath, content)
    restored.push(template.file_path)
  }

  rmSync(backupDir, { recursive: true, force: true })
  return restored
}
