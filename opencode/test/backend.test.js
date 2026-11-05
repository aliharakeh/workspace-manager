import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { act, createBackend, snapshot } from '../src/backend.js';

const repo = fileURLToPath(new URL('../../', import.meta.url));
const root = process.env.TEMP ? path.join(process.env.TEMP, 'opencode') : os.tmpdir();
fs.mkdirSync(root, { recursive: true });
const tmp = fs.mkdtempSync(path.join(root, 'wm-gui-'));
const dir = path.join(tmp, 'project');
const daemonPath = path.join(tmp, process.platform === 'win32' ? 'wmd.exe' : 'wmd');
const { client, service } = createBackend({ dataDir: path.join(tmp, 'd'), daemonPath });
const events = [];
client.on('event', (e) => events.push(e));

before(async () => {
  execFileSync('go', ['build', '-o', daemonPath, './cmd/wmd'], { cwd: repo, stdio: 'inherit' });
  fs.mkdirSync(dir);
  await client.connect();
});
after(async () => {
  try { await client.stopDaemon(); } finally { client.close(); }
  await new Promise((resolve) => setTimeout(resolve, 300));
  fs.rmSync(tmp, { recursive: true, force: true });
});

test('untracked folder, registration, shared config/env/templates and safe dispatch', async () => {
  assert.equal((await snapshot(service, dir)).app, null);
  await service.addApp(dir, { workspace: 'OpenCode' });
  let state = await snapshot(service, dir);
  const original = state.detail.name;
  const invoke = (action, args, expectedSetId) => act(service, { dir, action, args, expectedSetId });
  await invoke('setEnv', [{ PORT: '4321' }], state.detail.id);
  await invoke('setTemplate', ['app.conf', 'port={{PORT}}']);
  const template = await invoke('getTemplate', ['app.conf']);
  assert.equal(await service.renderTemplate(template.configSetId, template.content), 'port=4321');
  await invoke('createSet', ['Stage', { copyFrom: original, parts: { env: true, templates: true } }]);
  await assert.rejects(invoke('setEnv', [{ WRONG_SET: 'no' }], state.detail.id), /active config set changed/);
  state = await snapshot(service, dir);
  assert.equal(state.app.active_config_set_name, 'Stage');
  assert.equal(state.detail.env_vars.find((v) => v.key === 'PORT').value, '4321');
  assert.equal(state.detail.templates[0].content, 'port={{PORT}}');
  await assert.rejects(invoke('call', ['server.stop']), /Unknown action/);
  await assert.rejects(invoke('constructor', []), /Unknown action/);
  await assert.rejects(invoke('importEnv', ['../outside.env']), /inside the project/);
  await assert.rejects(invoke('setTemplate', ['../outside', 'no']), /inside the project/);
  await new Promise((resolve) => setTimeout(resolve, 100));
  assert.ok(events.some((e) => e.type === 'dataChanged'));
});

test('runner status, raw output, catch-up and template restoration', async () => {
  fs.writeFileSync(path.join(dir, 'app.conf'), 'original');
  const app = await service.requireApp(dir);
  const node = process.execPath.replace(/\\/g, '/');
  await service.call('runConfig.save', {
    appID: app.id, kind: 'build',
    body: { mode: 'sequential', commands: [{ command: `"${node}" -e "console.log(require('fs').readFileSync('app.conf','utf8'))"` }] },
  });
  await act(service, { dir, action: 'run', args: ['build'] });
  let state;
  for (let attempt = 0; attempt < 100; attempt++) {
    state = await snapshot(service, dir);
    if (!state.status.running) break;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  assert.equal(state.status.running, false);
  assert.ok(state.status.sessionId);
  assert.equal(state.status.processes[0].exitCode, 0);
  const output = await client.call('runner.output', { appID: app.id, commandID: state.status.processes[0].commandId });
  assert.match(Buffer.from(output.data, 'base64').toString('utf8'), /port=4321/);
  assert.ok(output.end > 0);
  assert.ok(events.some((e) => e.type === 'runnerEvent' && e.data?.type === 'log'));
  assert.equal(fs.readFileSync(path.join(dir, 'app.conf'), 'utf8'), 'original');
});
