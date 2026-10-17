// Runs the extension inside a real VS Code (the installed one, or VSCODE_PATH)
// against a real daemon: npm run test:host. It prepares a project in the daemon,
// opens it as the workspace folder, and runs test/host/suite.cjs in the extension
// host, which drives the extension through its agent commands and file systems.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { runTests } from '@vscode/test-electron';
import { DaemonClient } from '../../daemon.js';
import { Service } from '../../service.js';

const here = path.dirname(fileURLToPath(import.meta.url));
const extensionDir = path.resolve(here, '..', '..');
const repo = path.resolve(extensionDir, '..');

const candidates = [
  process.env.VSCODE_PATH,
  path.join(process.env.LOCALAPPDATA ?? '', 'Programs', 'Microsoft VS Code', 'Code.exe'),
  '/usr/share/code/code',
  '/Applications/Visual Studio Code.app/Contents/MacOS/Electron',
].filter(Boolean);
const vscodeExecutablePath = candidates.find((p) => fs.existsSync(p));
if (!vscodeExecutablePath) {
  console.log('No VS Code found (set VSCODE_PATH); skipping the host test.');
  process.exit(0);
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'wmh-'));
const exe = path.join(tmp, process.platform === 'win32' ? 'wmd.exe' : 'wmd');
const data = path.join(tmp, 'd');
const project = path.join(tmp, 'proj');
fs.mkdirSync(project, { recursive: true });
fs.writeFileSync(path.join(project, 'app.conf'), 'port=1\n');

execFileSync('go', ['build', '-o', exe, './cmd/wmd'], { cwd: repo, stdio: 'inherit' });
execFileSync('npm', ['run', 'bundle'], { cwd: extensionDir, stdio: 'inherit', shell: true });

// The daemon already has the project, as if it had been added from the desktop app.
const client = new DaemonClient({ dataDir: data, daemonPath: exe, version: 'host-test' });
await client.connect();
const svc = new Service(client);
const app = await svc.addApp(project, { workspace: 'Host Test' });
await svc.setEnv(project, { GREETING: 'hello' });
await svc.setTemplate(project, 'app.conf', 'port={{GREETING}}\n');
await svc.call('runConfig.save', {
  appID: app.id,
  kind: 'build',
  body: { mode: 'sequential', commands: [{ label: 'show', command: `"${process.execPath}" -e "console.log('built ok')"` }] },
});

let failed = false;
try {
  await runTests({
    vscodeExecutablePath,
    extensionDevelopmentPath: extensionDir,
    extensionTestsPath: path.join(here, 'suite.cjs'),
    launchArgs: [project, '--disable-extensions', '--disable-workspace-trust', '--user-data-dir', path.join(tmp, 'user'), '--extensions-dir', path.join(tmp, 'ext')],
    extensionTestsEnv: { WM_DATA_DIR: data, WM_DAEMON: exe, WM_TEST_PROJECT: project, WM_TEST_NODE: process.execPath },
  });
} catch (err) {
  console.error('Host test failed:', err?.message ?? err);
  failed = true;
}

try {
  await client.stopDaemon();
} catch {
  // already gone
}
client.close();
await new Promise((r) => setTimeout(r, 500));
fs.rmSync(tmp, { recursive: true, force: true });
process.exit(failed ? 1 : 0);
