export interface ConfigSet { id: number; name: string; active: boolean }
export interface Template { id: number; file_path: string; content: string }
export interface Process {
  commandId: number; label: string | null; command: string; status: string;
  exitCode: number | null; urls: string[];
}
export interface Snapshot {
  app: { id: number; name: string; active_config_set_name: string } | null;
  workspaces?: { id: number; name: string }[];
  sets?: ConfigSet[];
  detail?: {
    id: number; name: string;
    env_vars: { key: string; value: string; shared_var_id?: number | null }[];
    templates: Template[];
  };
  status?: { sessionId: string; running: boolean; kind: string | null; error: string | null; processes: Process[] };
}
