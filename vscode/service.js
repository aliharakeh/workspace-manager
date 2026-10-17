import fs from 'node:fs';
import path from 'node:path';

// Everything the extension does with the daemon, expressed per project folder
// ("dir"): a VS Code folder maps to the daemon app whose project_path is that
// folder. The UI and the agent commands both call this, so they behave the same.
// Nothing here imports vscode.

export const KINDS = ['setup', 'dev', 'build', 'prod'];

const RUN_METHOD = { setup: 'runner.setup', dev: 'runner.dev', build: 'runner.build', prod: 'runner.prod' };

/** Compares folders the way the OS does: slashes, trailing slash, and case on Windows/macOS. */
export function normalizeDir(dir) {
  let p = path.resolve(String(dir ?? '')).replace(/\\/g, '/');
  if (p.length > 1 && p.endsWith('/') && !/^[A-Za-z]:\/$/.test(p)) p = p.replace(/\/+$/, '');
  return process.platform === 'linux' ? p : p.toLowerCase();
}

/** A path inside dir, as the daemon stores it: relative, slash-separated. */
export function toProjectRelative(dir, target) {
  const raw = String(target ?? '').trim();
  if (!raw) throw new Error('Target file path is required.');
  const root = path.resolve(dir);
  const abs = path.isAbsolute(raw) ? path.resolve(raw) : path.resolve(root, raw);
  const rel = path.relative(root, abs);
  if (!rel || rel === '.' || rel.startsWith('..') || path.isAbsolute(rel)) {
    throw new Error(`Path "${target}" must be inside the project folder.`);
  }
  return rel.split(path.sep).join('/');
}

const sameName = (a, b) => String(a).trim().toLowerCase() === String(b).trim().toLowerCase();

export class Service {
  /** @param {import('./daemon.js').DaemonClient} client */
  constructor(client) {
    this.client = client;
  }

  call(method, params) {
    return this.client.call(method, params);
  }

  // ---- projects (apps) ----

  async workspaces() {
    return this.call('workspaces.list');
  }

  /** Every app with its workspace name. */
  async apps() {
    const workspaces = await this.workspaces();
    const lists = await Promise.all(
      workspaces.map(async (ws) => (await this.call('apps.list', { workspaceID: ws.id })).map((app) => ({ ...app, workspace: ws.name })))
    );
    return lists.flat();
  }

  /** The daemon app for a folder, or undefined when the folder is not tracked. */
  async findApp(dir) {
    const want = normalizeDir(dir);
    return (await this.apps()).find((app) => normalizeDir(app.project_path) === want);
  }

  async requireApp(dir) {
    const app = await this.findApp(dir);
    if (!app) {
      throw new Error(`"${dir}" is not tracked by Workspace Manager. Add the folder first (command: Add Folder to Workspace Manager).`);
    }
    return app;
  }

  /** Tracks a folder as an app, in the named workspace (created when missing). */
  async addApp(dir, { name, workspace } = {}) {
    const existing = await this.findApp(dir);
    if (existing) return existing;
    const workspaces = await this.workspaces();
    let ws = workspace ? workspaces.find((w) => sameName(w.name, workspace) || String(w.id) === String(workspace)) : undefined;
    if (!ws) {
      if (!workspace && workspaces.length > 0) ws = workspaces[0];
      else ws = await this.call('workspaces.create', { name: workspace || 'VS Code' });
    }
    return this.call('apps.create', {
      workspaceID: ws.id,
      body: { name: name || path.basename(path.resolve(dir)), project_path: path.resolve(dir) },
    });
  }

  // ---- config sets ----

  async sets(dir) {
    const app = await this.requireApp(dir);
    const sets = await this.call('configSets.list', { appID: app.id });
    return {
      app,
      activeSetId: app.active_config_set_id,
      activeSet: app.active_config_set_name,
      sets: sets.map((s) => ({ ...s, active: s.id === app.active_config_set_id })),
    };
  }

  async #setByName(dir, name) {
    const info = await this.sets(dir);
    const found = info.sets.find((s) => sameName(s.name, name));
    if (!found) throw new Error(`Config set "${name}" not found.`);
    return { ...info, set: found };
  }

  /** A config set's details (env vars, templates, run configs); the active set when no name is given. */
  async detail(dir, setName) {
    const info = setName ? await this.#setByName(dir, setName) : await this.sets(dir);
    const id = setName ? info.set.id : info.activeSetId;
    const detail = await this.call('configSets.detail', { id });
    return { app: info.app, set: { id: detail.id, name: detail.name, active: detail.id === info.activeSetId }, detail };
  }

  async switchSet(dir, name) {
    const { set } = await this.#setByName(dir, name);
    await this.call('configSets.activate', { id: set.id });
    return { activeSet: set.name };
  }

  async createSet(dir, name, { copyFrom, parts, activate = true } = {}) {
    const info = await this.sets(dir);
    const trimmed = String(name ?? '').trim();
    if (!trimmed) throw new Error('Config set name is required.');
    const body = { name: trimmed, activate };
    if (copyFrom) {
      const source = info.sets.find((s) => sameName(s.name, copyFrom));
      if (!source) throw new Error(`Source config set "${copyFrom}" not found.`);
      body.copy_from_id = source.id;
      if (parts) body.parts = parts;
    }
    const created = await this.call('configSets.create', { appID: info.app.id, body });
    return { name: created.name, active: Boolean(activate) };
  }

  async renameSet(dir, oldName, newName) {
    const { set } = await this.#setByName(dir, oldName);
    const trimmed = String(newName ?? '').trim();
    if (!trimmed) throw new Error('New config set name is required.');
    const updated = await this.call('configSets.update', { id: set.id, body: { name: trimmed } });
    return { name: updated.name };
  }

  async deleteSet(dir, name) {
    const { set } = await this.#setByName(dir, name);
    await this.call('configSets.delete', { id: set.id });
    return { deleted: set.name, activeSet: (await this.sets(dir)).activeSet };
  }

  /** Copies parts (env, templates, dev, ...; true or a list) from one set into another. */
  async copyFrom(dir, targetName, sourceName, parts = { env: true, templates: true }) {
    const info = await this.sets(dir);
    const target = targetName ? info.sets.find((s) => sameName(s.name, targetName)) : info.sets.find((s) => s.active);
    if (!target) throw new Error(`Target config set "${targetName}" not found.`);
    const source = info.sets.find((s) => sameName(s.name, sourceName));
    if (!source) throw new Error(`Source config set "${sourceName}" not found.`);
    await this.call('configSets.copyFrom', { id: target.id, sourceID: source.id, parts });
    return { target: target.name, source: source.name };
  }

  // ---- environment variables ----

  async listEnv(dir, setName) {
    const { set, detail } = await this.detail(dir, setName);
    const env = {};
    for (const v of [...detail.env_vars].sort((a, b) => a.key.localeCompare(b.key, undefined, { sensitivity: 'base' }))) env[v.key] = v.value;
    return { set: set.name, env };
  }

  /** Upserts variables in the active config set (the daemon writes env vars to the active set). */
  async setEnv(dir, entries) {
    if (!entries || typeof entries !== 'object' || Array.isArray(entries)) throw new Error('Environment entries must be an object.');
    const app = await this.requireApp(dir);
    const existing = await this.call('envVars.list', { appID: app.id });
    for (const [rawKey, rawValue] of Object.entries(entries)) {
      const key = String(rawKey).trim();
      if (!key) throw new Error('Environment variable key cannot be empty.');
      const value = String(rawValue ?? '');
      const found = existing.find((v) => v.key === key);
      if (found) await this.call('envVars.update', { id: found.id, body: { value } });
      else await this.call('envVars.create', { appID: app.id, body: { key, value } });
    }
    return this.listEnv(dir);
  }

  /** Renames and/or changes a variable of the active config set. */
  async updateEnv(dir, oldKey, newKey, value) {
    const app = await this.requireApp(dir);
    const found = (await this.call('envVars.list', { appID: app.id })).find((v) => v.key === oldKey);
    if (!found) throw new Error(`Environment variable "${oldKey}" not found.`);
    const body = { value: String(value ?? '') };
    if (newKey && newKey !== oldKey) body.key = String(newKey).trim();
    await this.call('envVars.update', { id: found.id, body });
    return this.listEnv(dir);
  }

  async removeEnv(dir, keys) {
    const app = await this.requireApp(dir);
    const list = Array.isArray(keys) ? keys : [keys];
    const existing = await this.call('envVars.list', { appID: app.id });
    const removed = [];
    for (const raw of list) {
      const found = existing.find((v) => v.key === String(raw).trim());
      if (found) {
        await this.call('envVars.delete', { id: found.id });
        removed.push(found.key);
      }
    }
    return { removed, ...(await this.listEnv(dir)) };
  }

  /** Imports a .env/YAML file inside the project: variables into the active set plus a matching template. */
  async importEnv(dir, file) {
    const app = await this.requireApp(dir);
    const abs = path.isAbsolute(file) ? file : path.resolve(dir, file);
    const res = await this.call('envVars.import', { appID: app.id, path: abs });
    return {
      set: app.active_config_set_name,
      imported: res.imported ?? 0,
      filePath: res.template?.file_path ?? toProjectRelative(dir, abs),
    };
  }

  // ---- templates ----

  async listTemplates(dir, setName) {
    const { set, detail } = await this.detail(dir, setName);
    const templates = {};
    for (const t of [...detail.templates].sort((a, b) => a.file_path.localeCompare(b.file_path))) templates[t.file_path] = t.content;
    return { set: set.name, templates };
  }

  async getTemplate(dir, file, setName) {
    const rel = toProjectRelative(dir, file);
    const { set, detail } = await this.detail(dir, setName);
    const found = detail.templates.find((t) => t.file_path === rel);
    if (!found) throw new Error(`Template "${rel}" not found in config set "${set.name}".`);
    return { set: set.name, filePath: rel, content: found.content, id: found.id, configSetId: set.id };
  }

  /** Creates or updates a template. New templates go to the active config set. */
  async setTemplate(dir, file, content, setName) {
    const rel = toProjectRelative(dir, file);
    const text = String(content ?? '');
    const { app, set, detail } = await this.detail(dir, setName);
    const found = detail.templates.find((t) => t.file_path === rel);
    if (found) {
      await this.call('templates.update', { id: found.id, body: { content: text } });
    } else {
      if (!set.active) throw new Error(`Switch to config set "${set.name}" before adding a new template to it.`);
      await this.call('templates.create', { appID: app.id, body: { file_path: rel, content: text } });
    }
    return { set: set.name, filePath: rel };
  }

  async removeTemplate(dir, file, setName) {
    const rel = toProjectRelative(dir, file);
    const { set, detail } = await this.detail(dir, setName);
    const found = detail.templates.find((t) => t.file_path === rel);
    if (!found) throw new Error(`Template "${rel}" not found in config set "${set.name}".`);
    await this.call('templates.delete', { id: found.id });
    return { set: set.name, removed: rel };
  }

  /** Saves a file's current content from disk as a template (creating or replacing it). */
  async snapshotFile(dir, file, setName) {
    const rel = toProjectRelative(dir, file);
    const abs = path.resolve(dir, rel);
    if (!fs.existsSync(abs) || !fs.statSync(abs).isFile()) throw new Error(`File "${rel}" does not exist on disk.`);
    return this.setTemplate(dir, rel, fs.readFileSync(abs, 'utf8'), setName);
  }

  /** What applying a template would write: the content rendered with a set's env vars. */
  async renderTemplate(configSetId, content) {
    const res = await this.call('templates.render', { config_set_id: configSetId, content });
    return res.content;
  }

  // ---- running ----

  async runConfig(dir, kind) {
    assertKind(kind);
    const app = await this.requireApp(dir);
    return this.call('runConfig.get', { appID: app.id, kind });
  }

  /** Starts the active config set's commands of a kind: setup, dev, build or prod. */
  async run(dir, kind) {
    assertKind(kind);
    const app = await this.requireApp(dir);
    const status = await this.call(RUN_METHOD[kind], { appID: app.id });
    return { kind, running: status.running, sessionId: status.sessionId };
  }

  async stop(dir) {
    const app = await this.requireApp(dir);
    const status = await this.call('runner.stop', { appID: app.id });
    return { running: status.running };
  }

  async reload(dir) {
    const app = await this.requireApp(dir);
    const status = await this.call('runner.reload', { appID: app.id });
    return { kind: status.kind, running: status.running };
  }

  async status(dir) {
    const app = await this.requireApp(dir);
    return summarizeStatus(await this.call('runner.status', { appID: app.id }));
  }

  /** Recent terminal output of one command of the app's run, as text. */
  async output(dir, commandId) {
    const app = await this.requireApp(dir);
    const out = await this.call('runner.output', { appID: app.id, commandID: commandId });
    return Buffer.from(out.data || '', 'base64').toString('utf8');
  }
}

function assertKind(kind) {
  if (!KINDS.includes(kind)) throw new Error(`kind must be one of ${KINDS.join(', ')}.`);
}

/** A runner StatusEvent reduced to what a person or agent reads. */
export function summarizeStatus(st) {
  return {
    running: Boolean(st.running),
    kind: st.kind || null,
    configSetId: st.configSetId ?? null,
    error: st.error || null,
    processes: (st.processes ?? []).map((p) => ({
      commandId: p.commandId,
      label: p.label ?? null,
      command: p.command,
      status: p.status,
      exitCode: p.exitCode ?? null,
      urls: p.urls ?? [],
    })),
  };
}
