import {
  apps,
  configSets,
  envVars,
  runCommands,
  runConfigs,
  templates,
  workspaces,
} from "./schema"

export type Workspace = typeof workspaces.$inferSelect
export type App = typeof apps.$inferSelect
export type ConfigSet = typeof configSets.$inferSelect
export type EnvVar = typeof envVars.$inferSelect
export type Template = typeof templates.$inferSelect
export type RunConfig = typeof runConfigs.$inferSelect
export type RunCommand = typeof runCommands.$inferSelect

export type RunMode = RunConfig["mode"]

export type RunConfigWithCommands = RunConfig & {
  commands: RunCommand[]
}
