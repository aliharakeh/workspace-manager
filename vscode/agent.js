import * as vscode from 'vscode';
import { normalizeDir, KINDS } from './service.js';

// Non-interactive commands for AI agents: workspaceManager.agent.<verb>. They never
// open a form, quick pick or dialog; everything comes in as arguments, and they
// always resolve to {ok: true, ...result} or {ok: false, error} and never throw.
// They call the same Service as the UI.

/** Finds the workspace folder an agent means: by path or name, or the only one open. */
function resolveProject(projects, arg) {
  if (projects.length === 0) throw new Error('No workspace folder is open.');
  if (arg !== undefined && arg !== null && arg !== '') {
    const text = String(arg).trim();
    const want = normalizeDir(text);
    const found = projects.find((p) => p.label.toLowerCase() === text.toLowerCase() || normalizeDir(p.dir) === want);
    if (!found) throw new Error(`Project "${arg}" not found. Open projects: ${projects.map((p) => `{ project: "${p.dir}" }`).join('; ')}`);
    return found;
  }
  if (projects.length === 1) return projects[0];
  throw new Error(`Project is ambiguous, specify one of: ${projects.map((p) => `{ project: "${p.dir}" }`).join('; ')}`);
}

/**
 * @param {vscode.ExtensionContext} context
 * @param {import('./service.js').Service} service
 * @param {() => Array<{label: string, dir: string}>} getProjects
 * @param {{ afterRun?: (dir: string) => void }} hooks
 */
export function registerAgentCommands(context, service, getProjects, hooks = {}) {
  const define = (verb, fn) => {
    context.subscriptions.push(
      vscode.commands.registerCommand(`workspaceManager.agent.${verb}`, async (args = {}) => {
        try {
          const a = args && typeof args === 'object' ? args : {};
          const project = resolveProject(getProjects(), a.project);
          return { ok: true, ...(await fn(project.dir, a)) };
        } catch (err) {
          return { ok: false, error: err?.message || String(err) };
        }
      })
    );
  };

  define('listSets', async (dir) => {
    const { activeSet, sets } = await service.sets(dir);
    return { activeSet, sets: sets.map((s) => ({ name: s.name, active: s.active })) };
  });
  define('switchSet', (dir, a) => service.switchSet(dir, a.name));
  define('createSet', (dir, a) => service.createSet(dir, a.name, { copyFrom: a.copyFrom, parts: a.parts, activate: a.activate !== false }));
  define('renameSet', (dir, a) => service.renameSet(dir, a.oldName ?? a.name, a.newName));
  define('copyFrom', (dir, a) => service.copyFrom(dir, a.targetSet, a.sourceSet, a.parts));
  define('deleteSet', (dir, a) => service.deleteSet(dir, a.name));

  define('listEnv', (dir, a) => service.listEnv(dir, a.set));
  define('setEnv', (dir, a) => service.setEnv(dir, a.entries ?? (a.key ? { [a.key]: a.value } : undefined)));
  define('removeEnv', (dir, a) => service.removeEnv(dir, a.keys ?? a.key));
  define('importEnv', (dir, a) => service.importEnv(dir, a.file));

  define('listTemplates', async (dir, a) => {
    const { set, templates } = await service.listTemplates(dir, a.set);
    return { set, templates: Object.keys(templates) };
  });
  define('getTemplate', async (dir, a) => {
    const { set, filePath, content } = await service.getTemplate(dir, a.file, a.set);
    return { set, filePath, content };
  });
  define('setTemplate', (dir, a) => service.setTemplate(dir, a.file, a.content, a.set));
  define('snapshotFile', (dir, a) => service.snapshotFile(dir, a.file, a.set));
  define('removeTemplate', (dir, a) => service.removeTemplate(dir, a.file, a.set));

  define('run', async (dir, a) => {
    if (!KINDS.includes(a.kind)) throw new Error(`kind must be one of ${KINDS.join(', ')}.`);
    const res = await service.run(dir, a.kind);
    hooks.afterRun?.(dir);
    return res;
  });
  define('stop', (dir) => service.stop(dir));
  define('status', (dir) => service.status(dir));
}
