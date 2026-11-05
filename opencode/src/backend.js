import { DaemonClient } from '../../vscode/daemon.js';
import { Service, toProjectRelative, summarizeStatus } from '../../vscode/service.js';

// Only the desktop main process owns this GUI-privileged connection. No agent tools.
export const actions = [
  'switchSet', 'createSet', 'renameSet', 'deleteSet', 'copyFrom',
  'setEnv', 'updateEnv', 'removeEnv', 'importEnv',
  'getTemplate', 'setTemplate', 'removeTemplate', 'snapshotFile',
  'run', 'stop', 'reload',
];

export function createBackend(options = {}, log) {
  const client = new DaemonClient({
    name: 'opencode-desktop', version: '0.1.0',
    dataDir: options.dataDir || undefined,
    daemonPath: options.daemonPath || undefined,
    autoStart: options.autoStartDaemon !== false,
    log,
  });
  return { client, service: new Service(client) };
}

export async function snapshot(service, dir) {
  const app = await service.findApp(dir);
  if (!app) return { app: null, workspaces: await service.workspaces() };
  const [info, current, status] = await Promise.all([
    service.sets(dir), service.detail(dir), service.call('runner.status', { appID: app.id }),
  ]);
  return { app, sets: info.sets, detail: current.detail, status: { ...summarizeStatus(status), sessionId: status.sessionId } };
}

/** @param {Service} service
 * @param {{dir: string, action: string, args: readonly unknown[], expectedSetId?: number}} input */
export async function act(service, { dir, action, args, expectedSetId }) {
  if (!actions.includes(action)) throw new Error(`Unknown action: ${action}`);
  if (!dir || !Array.isArray(args)) throw new Error('A project directory and arguments are required.');
  if (expectedSetId !== undefined && (await service.requireApp(dir)).active_config_set_id !== expectedSetId) {
    throw new Error('The active config set changed. Close this editor and try again.');
  }
  if (action === 'importEnv') toProjectRelative(dir, args[0]);
  return service[action](dir, ...args);
}
