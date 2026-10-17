// Runs inside the VS Code extension host (see run.mjs). Plain assertions, no framework.
const assert = require('node:assert/strict');
const vscode = require('vscode');

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const agent = (verb, args) => vscode.commands.executeCommand(`workspaceManager.agent.${verb}`, args);

async function until(fn, what, ms = 15000) {
  const end = Date.now() + ms;
  let last;
  while (Date.now() < end) {
    last = await fn();
    if (last) return last;
    await sleep(150);
  }
  throw new Error(`timed out waiting for ${what}`);
}

const templateUri = (dir, set, file, scheme = 'wm-template') =>
  vscode.Uri.from({
    scheme,
    authority: Buffer.from(dir, 'utf8').toString('base64url'),
    path: `/${encodeURIComponent(set)}/${file}`,
    query: `set=${encodeURIComponent(set)}&file=${encodeURIComponent(file)}`,
  });

exports.run = async function run() {
  const dir = process.env.WM_TEST_PROJECT;
  const ext = vscode.extensions.getExtension('workspace-manager.workspace-manager');
  assert.ok(ext, 'the extension is installed in the host');
  await ext.activate();

  // Commands are contributed, and agent commands are registered.
  const commands = await vscode.commands.getCommands(true);
  for (const id of ['workspaceManager.switchSet', 'workspaceManager.runDev', 'workspaceManager.agent.run', 'workspaceManager.agent.setEnv']) {
    assert.ok(commands.includes(id), `${id} is registered`);
  }

  // Sets and env, through the daemon.
  const sets = await until(async () => {
    const r = await agent('listSets', {});
    return r.ok && r;
  }, 'the extension to reach the daemon');
  const first = sets.activeSet;
  assert.equal(sets.sets.length, 1);

  const env = await agent('listEnv', {});
  assert.deepEqual(env.env, { GREETING: 'hello' });

  const set = await agent('setEnv', { entries: { PORT: '3000' } });
  assert.equal(set.ok, true, JSON.stringify(set));
  assert.equal(set.env.PORT, '3000');

  const bad = await agent('setEnv', { project: '/nowhere', entries: { X: '1' } });
  assert.equal(bad.ok, false);
  assert.match(bad.error, /not found/);

  const created = await agent('createSet', { name: 'Staging', copyFrom: first, parts: { env: true } });
  assert.equal(created.ok, true, JSON.stringify(created));
  assert.equal((await agent('listSets', {})).activeSet, 'Staging');
  assert.equal((await agent('switchSet', { name: first })).ok, true);
  assert.equal((await agent('deleteSet', { name: 'Staging' })).ok, true);

  // The template opens through the virtual file system, edits save to the daemon,
  // and the preview renders with the set's env vars.
  const uri = templateUri(dir, first, 'app.conf');
  const text = Buffer.from(await vscode.workspace.fs.readFile(uri)).toString('utf8');
  assert.equal(text, 'port={{GREETING}}\n');

  await vscode.workspace.fs.writeFile(uri, Buffer.from('port={{PORT}}\nhi={{GREETING}}\n'));
  const saved = await agent('getTemplate', { file: 'app.conf' });
  assert.equal(saved.content, 'port={{PORT}}\nhi={{GREETING}}\n');

  const preview = await vscode.workspace.openTextDocument(templateUri(dir, first, 'app.conf', 'wm-template-preview'));
  assert.equal(preview.getText(), 'port=3000\nhi=hello\n');

  // A UI-opened template is a normal editor document.
  const doc = await vscode.workspace.openTextDocument(uri);
  assert.equal(doc.languageId.length > 0, true);

  // Running: the daemon applies the template, the extension opens an output terminal.
  const started = await agent('run', { kind: 'build' });
  assert.equal(started.ok, true, JSON.stringify(started));
  const done = await until(async () => {
    const st = await agent('status', {});
    return st.ok && !st.running && st.processes.length === 1 && st;
  }, 'the build to finish');
  assert.equal(done.processes[0].status, 'exited');
  assert.equal(done.processes[0].exitCode, 0);

  await until(() => vscode.window.terminals.some((t) => t.name.startsWith('wm:')), 'the output terminal');

  const refused = await agent('run', { kind: 'deploy' });
  assert.equal(refused.ok, false);

  // Terminal env injection: the active set's variables reach new terminals of the folder.
  const out = require('node:path').join(require('node:os').tmpdir(), `wm-env-${Date.now()}.txt`);
  const terminal = vscode.window.createTerminal({
    name: 'env probe',
    shellPath: process.env.WM_TEST_NODE,
    shellArgs: ['-e', `require('fs').writeFileSync(${JSON.stringify(out)}, process.env.PORT + ':' + process.env.GREETING)`],
    cwd: dir,
  });
  const fs = require('node:fs');
  const probed = await until(() => fs.existsSync(out) && fs.readFileSync(out, 'utf8'), 'the terminal to print its env');
  terminal.dispose();
  fs.rmSync(out, { force: true });
  assert.equal(probed, '3000:hello');
};
