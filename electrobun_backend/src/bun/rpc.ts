import { readFileSync } from "node:fs"
import { basename } from "node:path"
import { BrowserView, Utils } from "electrobun/bun"
import type { AppRPC } from "../shared/rpc"
import type { CopyParts } from "../shared/types"
import { appsRepo } from "../../../db/apps"
import {
  configSetsRepo,
  type ConfigCopyParts,
} from "../../../db/config-sets"
import { envVarsRepo } from "../../../db/env-vars"
import { readyUrlPatternsRepo } from "../../../db/ready-url-patterns"
import { runConfigsRepo } from "../../../db/run-configs"
import { settingsRepo } from "../../../db/settings"
import { templatesRepo } from "../../../db/templates"
import { workspacesRepo } from "../../../db/workspaces"
import type { RunMode } from "../../../db/types"
import { detectImportFormat } from "./lib/import-formats"
import {
  readProjectFile,
  toProjectRelative,
  validateProjectPath,
} from "./lib/fs"
import { openInEditor } from "../../../native/editor"
import { pickNativeFile, pickNativeFolder } from "./native/dialog"
import {
  listListeningProcesses,
  USER_PORT_MAX,
  USER_PORT_MIN,
} from "../../../native/ports"
import { killPid } from "../../../native/process"
import {
  invalidateReadyUrlPatternsCache,
  validateReadyUrlPattern,
} from "./services/ready-url"
import { runner } from "./services/runner"

function notFound(message: string): never {
  throw new Error(message)
}

function requireWorkspace(id: number) {
  const workspace = workspacesRepo.get(id)
  if (!workspace) notFound("Workspace not found")
  return workspace
}

function requireApp(id: number) {
  const app = appsRepo.get(id)
  if (!app) notFound("App not found")
  return app
}

function isPartEnabled(part: boolean | string[] | number[] | undefined) {
  if (part === undefined || part === true) return true
  if (Array.isArray(part)) return part.length > 0
  return false
}

function hasAnyPart(parts?: ConfigCopyParts) {
  if (!parts) return true
  return (
    isPartEnabled(parts.env) ||
    isPartEnabled(parts.templates) ||
    isPartEnabled(parts.run)
  )
}

function asCopyParts(parts?: CopyParts): ConfigCopyParts | undefined {
  return parts
}

function normalizeFlags(flags: string | undefined) {
  const next = (flags ?? "i").trim() || "i"
  if (!/^[gimsuy]*$/.test(next)) throw new Error("Invalid regex flags")
  return next
}

function readPickedFile(absolutePath: string) {
  try {
    return readFileSync(absolutePath, "utf8")
  } catch (err) {
    throw new Error(
      err instanceof Error ? err.message : "Failed to read selected file"
    )
  }
}

export function createAppRPC() {
  return BrowserView.defineRPC<AppRPC>({
    maxRequestTime: Infinity,
    handlers: {
      requests: {
        workspacesList: () => workspacesRepo.list(),
        workspacesCreate: ({ name, icon }) => {
          if (!name?.trim()) throw new Error("name is required")
          return workspacesRepo.create({
            name: name.trim(),
            icon: icon ?? null,
          })
        },
        workspacesUpdate: ({ id, name, icon }) => {
          if (name !== undefined && !name.trim()) {
            throw new Error("name cannot be empty")
          }
          const workspace = workspacesRepo.update(id, {
            name: name?.trim(),
            icon,
          })
          if (!workspace) notFound("Workspace not found")
          return workspace
        },
        workspacesDelete: ({ id }) => {
          if (!workspacesRepo.delete(id)) notFound("Workspace not found")
          return { ok: true as const }
        },

        appsList: ({ workspaceId }) => {
          requireWorkspace(workspaceId)
          return appsRepo.listByWorkspace(workspaceId)
        },
        appsGet: ({ id }) => requireApp(id),
        appsCreate: ({ workspaceId, name, project_path }) => {
          requireWorkspace(workspaceId)
          if (!name?.trim()) throw new Error("name is required")
          if (!project_path?.trim()) throw new Error("project_path is required")
          const pathCheck = validateProjectPath(project_path)
          if (!pathCheck.ok) throw new Error(pathCheck.error)
          return appsRepo.create({
            workspace_id: workspaceId,
            name: name.trim(),
            project_path: pathCheck.resolved!,
          })
        },
        appsUpdate: ({ id, name, project_path }) => {
          if (name !== undefined && !name.trim()) {
            throw new Error("name cannot be empty")
          }
          let projectPath: string | undefined
          if (project_path !== undefined) {
            const pathCheck = validateProjectPath(project_path)
            if (!pathCheck.ok) throw new Error(pathCheck.error)
            projectPath = pathCheck.resolved
          }
          const app = appsRepo.update(id, {
            name: name?.trim(),
            project_path: projectPath,
          })
          if (!app) notFound("App not found")
          return app
        },
        appsDelete: ({ id }) => {
          if (!appsRepo.delete(id)) notFound("App not found")
          return { ok: true as const }
        },
        appsOpenInEditor: ({ id }) => {
          const app = requireApp(id)
          const pathCheck = validateProjectPath(app.project_path)
          if (!pathCheck.ok) throw new Error(pathCheck.error)
          openInEditor(pathCheck.resolved!)
          return { ok: true as const }
        },

        configSetsList: ({ appId }) => {
          requireApp(appId)
          return configSetsRepo.listByApp(appId)
        },
        configSetsGetDetail: ({ id }) => {
          const set = configSetsRepo.get(id)
          if (!set) notFound("Config set not found")
          return {
            ...set,
            env_vars: envVarsRepo.listByConfigSet(id),
            templates: templatesRepo.listByConfigSet(id),
            run_config: runConfigsRepo.getByConfigSet(id),
          }
        },
        configSetsCreate: ({ appId, name, copy_from_id, activate, parts }) => {
          requireApp(appId)
          if (!name?.trim()) throw new Error("name is required")
          const copyParts = asCopyParts(parts)
          if (copy_from_id != null && !hasAnyPart(copyParts)) {
            throw new Error("Select at least one part to copy")
          }
          if (copy_from_id != null) {
            const source = configSetsRepo.get(copy_from_id)
            if (!source || source.app_id !== appId) {
              throw new Error("copy_from_id must be a config set on this app")
            }
          }
          try {
            const set = configSetsRepo.create({
              app_id: appId,
              name: name.trim(),
            })
            if (copy_from_id != null) {
              configSetsRepo.copyFrom(copy_from_id, set.id, copyParts)
            }
            if (activate !== false) {
              appsRepo.setActiveConfigSet(appId, set.id)
            }
            return set
          } catch {
            throw new Error("A config set with this name already exists")
          }
        },
        configSetsUpdate: ({ id, name }) => {
          if (!name?.trim()) throw new Error("name is required")
          try {
            const set = configSetsRepo.update(id, { name: name.trim() })
            if (!set) notFound("Config set not found")
            return set
          } catch {
            throw new Error("A config set with this name already exists")
          }
        },
        configSetsDelete: ({ id }) => {
          const set = configSetsRepo.get(id)
          if (!set) notFound("Config set not found")
          const siblings = configSetsRepo.listByApp(set.app_id)
          if (siblings.length <= 1) {
            throw new Error("Cannot delete the last config set")
          }
          const app = appsRepo.get(set.app_id)
          configSetsRepo.delete(id)
          if (app?.active_config_set_id === id) {
            const next = configSetsRepo.listByApp(set.app_id)[0]
            if (next) appsRepo.setActiveConfigSet(set.app_id, next.id)
          }
          return { ok: true as const }
        },
        configSetsActivate: ({ id }) => {
          const set = configSetsRepo.get(id)
          if (!set) notFound("Config set not found")
          const app = appsRepo.setActiveConfigSet(set.app_id, set.id)
          return { ...set, app: app! }
        },
        configSetsCopyFrom: ({ id, sourceId, parts }) => {
          const target = configSetsRepo.get(id)
          if (!target) notFound("Config set not found")
          const copyParts = asCopyParts(parts)
          if (!hasAnyPart(copyParts)) {
            throw new Error("Select at least one part to copy")
          }
          const source = configSetsRepo.get(sourceId)
          if (!source || source.app_id !== target.app_id) {
            throw new Error("source_id must be a config set on the same app")
          }
          configSetsRepo.copyFrom(sourceId, id, copyParts)
          return configSetsRepo.get(id)!
        },

        envVarsList: ({ appId }) => {
          requireApp(appId)
          const set = configSetsRepo.resolveActive(appId)
          return envVarsRepo.listByConfigSet(set.id)
        },
        envVarsCreate: ({ appId, key, value }) => {
          requireApp(appId)
          if (!key?.trim()) throw new Error("key is required")
          const set = configSetsRepo.resolveActive(appId)
          try {
            return envVarsRepo.create({
              config_set_id: set.id,
              key: key.trim(),
              value: value ?? "",
            })
          } catch {
            throw new Error("Env var key already exists for this config set")
          }
        },
        envVarsUpdate: ({ id, key, value }) => {
          if (key !== undefined && !key.trim()) {
            throw new Error("key cannot be empty")
          }
          try {
            const envVar = envVarsRepo.update(id, {
              key: key?.trim(),
              value,
            })
            if (!envVar) notFound("Env var not found")
            return envVar
          } catch {
            throw new Error("Env var key already exists for this config set")
          }
        },
        envVarsDelete: ({ id }) => {
          if (!envVarsRepo.delete(id)) notFound("Env var not found")
          return { ok: true as const }
        },
        envVarsImport: async ({ appId }) => {
          const app = requireApp(appId)
          const picked = await pickNativeFile(app.project_path)
          if (!picked.ok) {
            if ("cancelled" in picked && picked.cancelled) {
              return { cancelled: true as const }
            }
            throw new Error(
              "error" in picked ? picked.error : "File dialog failed"
            )
          }
          const content = readPickedFile(picked.path)
          const format = detectImportFormat(picked.path)
          if (!format) {
            throw new Error(`Unsupported file format: ${basename(picked.path)}`)
          }
          const entries = format.parse(content)
          const set = configSetsRepo.resolveActive(appId)
          entries.forEach((e) => envVarsRepo.upsertByKey(set.id, e.key, e.value))

          let template: {
            id: number
            file_path: string
            created: boolean
          } | null = null
          try {
            const relPath =
              toProjectRelative(app.project_path, picked.path) ??
              basename(picked.path.replace(/\\/g, "/"))
            const templateContent = format.toTemplate(content)
            const existing = templatesRepo
              .listByConfigSet(set.id)
              .find((t) => t.file_path === relPath)
            if (existing) {
              templatesRepo.update(existing.id, { content: templateContent })
              template = { id: existing.id, file_path: relPath, created: false }
            } else {
              const created = templatesRepo.create({
                config_set_id: set.id,
                file_path: relPath,
                content: templateContent,
              })
              template = { id: created.id, file_path: relPath, created: true }
            }
          } catch (err) {
            console.error(
              `[env-vars/import] template creation failed for "${picked.path}":`,
              err
            )
          }

          return {
            cancelled: false as const,
            path: picked.path,
            format: format.label,
            imported: entries.length,
            vars: envVarsRepo.listByConfigSet(set.id),
            template,
          }
        },

        templatesList: ({ appId }) => {
          requireApp(appId)
          const set = configSetsRepo.resolveActive(appId)
          return templatesRepo.listByConfigSet(set.id)
        },
        templatesCreate: ({ appId, file_path, content }) => {
          requireApp(appId)
          if (!file_path?.trim()) throw new Error("file_path is required")
          const set = configSetsRepo.resolveActive(appId)
          try {
            return templatesRepo.create({
              config_set_id: set.id,
              file_path: file_path.trim().replace(/\\/g, "/"),
              content: content ?? "",
            })
          } catch {
            throw new Error("Template for this file already exists")
          }
        },
        templatesUpdate: ({ id, file_path, content }) => {
          if (file_path !== undefined && !file_path.trim()) {
            throw new Error("file_path cannot be empty")
          }
          try {
            const template = templatesRepo.update(id, {
              file_path: file_path?.trim().replace(/\\/g, "/"),
              content,
            })
            if (!template) notFound("Template not found")
            return template
          } catch {
            throw new Error("Template for this file already exists")
          }
        },
        templatesDelete: ({ id }) => {
          if (!templatesRepo.delete(id)) notFound("Template not found")
          return { ok: true as const }
        },

        runConfigGet: ({ appId }) => {
          requireApp(appId)
          const set = configSetsRepo.resolveActive(appId)
          return runConfigsRepo.getOrCreate(set.id)
        },
        runConfigSave: ({ appId, mode, commands }) => {
          requireApp(appId)
          if (mode && mode !== "sequential" && mode !== "parallel") {
            throw new Error("mode must be sequential or parallel")
          }
          if (commands) {
            for (const cmd of commands) {
              if (!cmd.command?.trim()) {
                throw new Error("Each command must have a non-empty command string")
              }
            }
          }
          const set = configSetsRepo.resolveActive(appId)
          return runConfigsRepo.upsert(set.id, {
            mode: mode as RunMode | undefined,
            commands: commands?.map((c) => ({
              label: c.label ?? null,
              command: c.command.trim(),
            })),
          })
        },

        runnerStatus: ({ appId }) =>
          runner.getStatus(appId) ?? {
            running: false,
            appId,
            processes: [],
          },
        runnerWorkspaceStatus: ({ workspaceId }) =>
          appsRepo.listByWorkspace(workspaceId).map((app) => {
            return (
              runner.getStatus(app.id) ?? {
                running: false,
                appId: app.id,
                processes: [],
              }
            )
          }),
        runnerLogs: ({ appId }) => runner.getSnapshot(appId),
        runnerRun: ({ appId }) => {
          requireApp(appId)
          return runner.start(appId)
        },
        runnerStop: ({ appId }) => runner.stop(appId),
        runnerReload: ({ appId }) => {
          requireApp(appId)
          return runner.reload(appId)
        },

        portsList: async () => ({
          min: USER_PORT_MIN,
          max: USER_PORT_MAX,
          processes: await listListeningProcesses(),
        }),
        portsKill: async ({ pid }) => {
          await killPid(pid)
          return { ok: true as const, pid }
        },

        readyUrlPatternsList: () => {
          readyUrlPatternsRepo.ensureSeeded()
          return readyUrlPatternsRepo.list()
        },
        readyUrlPatternsCreate: ({ label, pattern, flags }) => {
          if (!label?.trim()) throw new Error("label is required")
          if (!pattern?.trim()) throw new Error("pattern is required")
          const normalized = normalizeFlags(flags)
          const validated = validateReadyUrlPattern(pattern.trim(), normalized)
          if (!validated.ok) throw new Error(validated.error)
          const row = readyUrlPatternsRepo.create({
            label: label.trim(),
            pattern: pattern.trim(),
            flags: normalized,
          })
          invalidateReadyUrlPatternsCache()
          return row
        },
        readyUrlPatternsUpdate: ({ id, label, pattern, flags }) => {
          const existing = readyUrlPatternsRepo.get(id)
          if (!existing) notFound("Pattern not found")
          const nextLabel = label !== undefined ? label.trim() : existing.label
          const nextPattern =
            pattern !== undefined ? pattern.trim() : existing.pattern
          const nextFlags =
            flags !== undefined ? normalizeFlags(flags) : existing.flags
          if (!nextLabel) throw new Error("label cannot be empty")
          if (!nextPattern) throw new Error("pattern cannot be empty")
          const validated = validateReadyUrlPattern(nextPattern, nextFlags)
          if (!validated.ok) throw new Error(validated.error)
          const row = readyUrlPatternsRepo.update(id, {
            label: nextLabel,
            pattern: nextPattern,
            flags: nextFlags,
          })
          invalidateReadyUrlPatternsCache()
          return row!
        },
        readyUrlPatternsDelete: ({ id }) => {
          if (!readyUrlPatternsRepo.delete(id)) notFound("Pattern not found")
          invalidateReadyUrlPatternsCache()
          return { ok: true as const }
        },

        settingsGet: () => settingsRepo.getAll(),
        settingsSet: ({ key, value }) => {
          if (!key?.trim()) throw new Error("key is required")
          settingsRepo.set(key.trim(), value ?? "")
          return settingsRepo.getAll()
        },

        fsValidatePath: ({ path }) => {
          if (!path) throw new Error("path is required")
          const result = validateProjectPath(path)
          if (!result.ok) return { ok: false as const, error: result.error! }
          return { ok: true as const, path: result.resolved! }
        },
        fsPickFolder: async ({ startDir }) => {
          const picked = await pickNativeFolder(startDir?.trim() || undefined)
          if (!picked.ok) {
            if ("cancelled" in picked && picked.cancelled) {
              return { cancelled: true as const }
            }
            throw new Error(
              "error" in picked ? picked.error : "Folder dialog failed"
            )
          }
          return { cancelled: false as const, path: picked.path }
        },
        fsPickFile: async ({ startDir, appId }) => {
          let dir = startDir?.trim() || undefined
          let projectRoot: string | undefined
          if (appId != null) {
            const app = requireApp(appId)
            projectRoot = app.project_path
            dir = dir || app.project_path
          }
          const picked = await pickNativeFile(dir)
          if (!picked.ok) {
            if ("cancelled" in picked && picked.cancelled) {
              return { cancelled: true as const }
            }
            throw new Error(
              "error" in picked ? picked.error : "File dialog failed"
            )
          }
          const content = readPickedFile(picked.path)
          if (projectRoot) {
            const relative = toProjectRelative(projectRoot, picked.path)
            if (!relative) {
              throw new Error(
                "Selected file must be inside the app project directory"
              )
            }
            return {
              cancelled: false as const,
              path: picked.path,
              relative_path: relative,
              content,
            }
          }
          return { cancelled: false as const, path: picked.path, content }
        },
        fsPickAppFile: async ({ appId }) => {
          const app = requireApp(appId)
          const picked = await pickNativeFile(app.project_path)
          if (!picked.ok) {
            if ("cancelled" in picked && picked.cancelled) {
              return { cancelled: true as const }
            }
            throw new Error(
              "error" in picked ? picked.error : "File dialog failed"
            )
          }
          const relative = toProjectRelative(app.project_path, picked.path)
          if (!relative) {
            throw new Error(
              "Selected file must be inside the app project directory"
            )
          }
          return {
            cancelled: false as const,
            path: picked.path,
            relative_path: relative,
            content: readPickedFile(picked.path),
          }
        },
        fsReadAppFile: ({ appId, path }) => {
          const app = requireApp(appId)
          if (!path?.trim()) throw new Error("path is required")
          const result = readProjectFile(app.project_path, path)
          if (!result.ok) throw new Error(result.error)
          return result
        },

        openExternal: ({ url }) => {
          if (!url?.trim()) throw new Error("url is required")
          Utils.openExternal(url.trim())
          return { ok: true as const }
        },
      },
    },
  })
}
