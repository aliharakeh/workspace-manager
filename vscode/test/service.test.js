import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { DaemonClient } from '../daemon.js';
import { Service } from '../service.js';

// These run the real wmd binary: the daemon is built once, started on demand by
// the client in a fresh (short) data folder, and driven through Service.

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, '..', '..');
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'wmx-'));
const exe = path.join(tmp, process.platform === 'win32' ? 'wmd.exe' : 'wmd');
const data = path.join(tmp, 'd');
const project = path.join(tmp, 'proj');

let client;
let svc;
const events = [];

before(async () => {
  execFileSync('go', ['build', '-o', exe, './cmd/wmd'], { cwd: repo, stdio: 'inherit' });
  fs.mkdirSync(project, { recursive: true });
  client = new DaemonClient({ dataDir: data, daemonPath: exe, version: 'test' });
  client.on('event', (ev) => events.push(ev));
  await client.connect(); // starts the daemon: nothing is running yet
  svc = new Service(client);
});

after(async () => {
  try {
    await client.stopDaemon();
  } catch {
    // already gone
  }
  client.close();
  await new Promise((r) => setTimeout(r, 300));
  fs.rmSync(tmp, { recursive: true, force: true });
});

test('a folder is tracked as an app, matched by its path', async () => {
  assert.equal(await svc.findApp(project), undefined);
  await assert.rejects(svc.sets(project), /not tracked/);
  const app = await svc.addApp(project, { workspace: 'VS Code' });
  assert.equal(app.name, 'proj');
  const found = await svc.findApp(project + path.sep); // trailing slash, same folder
  assert.equal(found.id, app.id);
  assert.equal((await svc.addApp(project)).id, app.id, 'adding twice returns the same app');
});

test('config sets: create with copy, switch, rename, delete', async () => {
  const start = await svc.sets(project);
  assert.equal(start.sets.length, 1);
  const first = start.activeSet;

  await svc.setEnv(project, { A: '1', B: 'two' });
  const created = await svc.createSet(project, 'Staging', { copyFrom: first, parts: { env: true } });
  assert.equal(created.name, 'Staging');
  let info = await svc.sets(project);
  assert.equal(info.activeSet, 'Staging', 'a new set becomes active');
  assert.deepEqual((await svc.listEnv(project)).env, { A: '1', B: 'two' }, 'env was copied');

  await svc.switchSet(project, first.toUpperCase());
  assert.equal((await svc.sets(project)).activeSet, first);

  await svc.renameSet(project, 'Staging', 'Stage');
  assert.ok((await svc.sets(project)).sets.some((s) => s.name === 'Stage'));
  await svc.deleteSet(project, 'Stage');
  assert.equal((await svc.sets(project)).sets.length, 1);
  await assert.rejects(svc.switchSet(project, 'nope'), /not found/);
});

test('env vars: upsert, update key, remove, other set is readable', async () => {
  const res = await svc.setEnv(project, { A: '10', C: 'x y' });
  assert.deepEqual(res.env, { A: '10', B: 'two', C: 'x y' });
  await svc.updateEnv(project, 'C', 'D', 'z');
  assert.deepEqual((await svc.listEnv(project)).env, { A: '10', B: 'two', D: 'z' });
  const removed = await svc.removeEnv(project, ['B', 'missing']);
  assert.deepEqual(removed.removed, ['B']);

  const active = (await svc.sets(project)).activeSet;
  await svc.createSet(project, 'Other', { activate: false });
  assert.deepEqual((await svc.listEnv(project, 'Other')).env, {}, 'other set is empty');
  assert.deepEqual((await svc.listEnv(project, active)).env, { A: '10', D: 'z' });
  await svc.deleteSet(project, 'Other');
});

test('templates: set, get, update, render, snapshot, import, remove', async () => {
  fs.writeFileSync(path.join(project, 'app.conf'), 'port=1\n');
  await svc.setTemplate(project, 'app.conf', 'port={{A}}\n');
  const t = await svc.getTemplate(project, path.join(project, 'app.conf'));
  assert.equal(t.content, 'port={{A}}\n');
  assert.equal(await svc.renderTemplate(t.configSetId, t.content), 'port=10\n');

  await svc.setTemplate(project, 'app.conf', 'port={{A}}!\n');
  assert.equal((await svc.getTemplate(project, 'app.conf')).content, 'port={{A}}!\n');

  fs.writeFileSync(path.join(project, 'disk.txt'), 'from disk');
  await svc.snapshotFile(project, 'disk.txt');
  assert.equal((await svc.getTemplate(project, 'disk.txt')).content, 'from disk');
  await assert.rejects(svc.snapshotFile(project, '../outside.txt'), /inside the project/);

  fs.writeFileSync(path.join(project, '.env'), 'FOO=bar\nBAZ=qux\n');
  const imp = await svc.importEnv(project, '.env');
  assert.equal(imp.imported, 2);
  assert.equal((await svc.listEnv(project)).env.FOO, 'bar');
  assert.match((await svc.getTemplate(project, '.env')).content, /FOO=\{\{FOO\}\}/);

  const removed = await svc.removeTemplate(project, 'disk.txt');
  assert.equal(removed.removed, 'disk.txt');
  await assert.rejects(svc.getTemplate(project, 'disk.txt'), /not found/);
});

test('writes are announced as dataChanged events', async () => {
  events.length = 0;
  await svc.setEnv(project, { EVENT_PROBE: '1' });
  await new Promise((r) => setTimeout(r, 200));
  assert.ok(events.some((e) => e.type === 'dataChanged'), JSON.stringify(events));
});

test('running: a build applies templates, streams output, and ends', async () => {
  const node = process.execPath.replace(/\\/g, '/');
  await svc.call('runConfig.save', {
    appID: (await svc.requireApp(project)).id,
    kind: 'build',
    body: { mode: 'sequential', commands: [{ command: `"${node}" -e "console.log(require('fs').readFileSync('app.conf','utf8').trim())"` }] },
  });
  assert.equal((await svc.runConfig(project, 'build')).commands.length, 1);

  const started = await svc.run(project, 'build');
  assert.equal(started.kind, 'build');
  let status;
  for (let i = 0; i < 100; i++) {
    status = await svc.status(project);
    if (!status.running) break;
    await new Promise((r) => setTimeout(r, 100));
  }
  assert.equal(status.running, false);
  assert.equal(status.processes[0].status, 'exited');
  const out = await svc.output(project, status.processes[0].commandId);
  assert.match(out, /port=10!/, `the template was rendered for the run: ${out}`);
  assert.equal(fs.readFileSync(path.join(project, 'app.conf'), 'utf8'), 'port=1\n', 'the original file is restored');
});

test('an unknown run kind is refused', async () => {
  await assert.rejects(svc.run(project, 'deploy'), /kind must be/);
});
