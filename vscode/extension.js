import * as vscode from 'vscode';
import fs from 'node:fs';
import path from 'node:path';
import { DaemonClient, dataDir } from './daemon.js';
import { Service, KINDS, normalizeDir, toProjectRelative } from './service.js';
import { RunTracker } from './runner.js';
import { WmTree } from './tree.js';
import { registerTemplateEditors, TEMPLATE_SCHEME, parseTemplateUri } from './template-fs.js';
import { registerAgentCommands } from './agent.js';

const cfg = () => vscode.workspace.getConfiguration('workspaceManager');

function getProjects() {
  return (vscode.workspace.workspaceFolders ?? []).map((f) => ({ label: f.name, dir: f.uri.fsPath, folder: f }));
}

export function activate(context) {
  const output = vscode.window.createOutputChannel('Workspace Manager');
  context.subscriptions.push(output);

  const client = new DaemonClient({
    dataDir: cfg().get('dataDir') || undefined,
    daemonPath: cfg().get('daemonPath') || undefined,
    autoStart: cfg().get('autoStartDaemon') !== false,
    version: context.extension.packageJSON.version,
    log: (line) => output.appendLine(line),
  });
  const service = new Service(client);

  let selectedDir = getProjects()[0]?.dir ?? '';

  // ---- state shared by the views ----

  const tracker = new RunTracker(service, () => {
    tree.refresh();
    updateStatusBar();
  });
  const tree = new WmTree(service, tracker, getProjects);
  const view = vscode.window.createTreeView('workspaceManager.view', { treeDataProvider: tree });
  context.subscriptions.push(view, { dispose: () => tracker.dispose() });

  const setItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 50);
  setItem.command = 'workspaceManager.switchSet';
  const runItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 49);
  runItem.command = 'workspaceManager.stop';
  context.subscriptions.push(setItem, runItem);

  const templates = registerTemplateEditors(context, service, async () => {
    refreshAll();
  });

  // ---- project resolution ----

  function matchProject(projects, arg) {
    if (!arg || typeof arg !== 'string') return undefined;
    const want = normalizeDir(arg);
    return projects.find((p) => p.dir === arg || p.label === arg || normalizeDir(p.dir) === want);
  }

  function selectedProject() {
    const projects = getProjects();
    if (projects.length === 0) return undefined;
    const matched = matchProject(projects, selectedDir);
    if (matched) return matched;
    const uri = vscode.window.activeTextEditor?.document?.uri;
    if (uri?.scheme === TEMPLATE_SCHEME) {
      const m = matchProject(projects, parseTemplateUri(uri).dir);
      if (m) return (selectedDir = m.dir), m;
    } else if (uri?.scheme === 'file') {
      const wf = vscode.workspace.getWorkspaceFolder(uri);
      const m = wf && matchProject(projects, wf.uri.fsPath);
      if (m) return (selectedDir = m.dir), m;
    }
    selectedDir = projects[0].dir;
    return projects[0];
  }

  /** The project a UI command acts on: from the tree node, a file, or a pick. */
  async function resolveProject(arg) {
    const projects = getProjects();
    if (projects.length === 0) {
      vscode.window.showErrorMessage('Workspace Manager: open a folder first.');
      return undefined;
    }
    let found;
    if (arg?.project?.dir) found = arg.project;
    else if (arg instanceof vscode.Uri && arg.scheme === 'file') {
      const wf = vscode.workspace.getWorkspaceFolder(arg);
      found = wf && matchProject(projects, wf.uri.fsPath);
    } else if (typeof arg === 'string') found = matchProject(projects, arg);
    if (!found) {
      found = projects.length === 1 ? projects[0] : selectedProject();
    }
    if (!found) {
      const picked = await vscode.window.showQuickPick(
        projects.map((p) => ({ label: p.label, description: p.dir, project: p })),
        { placeHolder: 'Select project' }
      );
      found = picked?.project;
    }
    if (found) selectedDir = found.dir;
    return found;
  }

  /** The project, which must also be tracked by the daemon (offers to add it). */
  async function trackedProject(arg) {
    const project = await resolveProject(arg);
    if (!project) return undefined;
    if (await service.findApp(project.dir)) return project;
    const choice = await vscode.window.showInformationMessage(
      `"${project.label}" is not tracked by Workspace Manager yet.`,
      'Add It'
    );
    if (choice !== 'Add It') return undefined;
    return (await addProject(project)) ? project : undefined;
  }

  async function addProject(arg) {
    const project = await resolveProject(arg);
    if (!project) return undefined;
    const workspaces = await service.workspaces();
    let workspace;
    if (workspaces.length > 0) {
      const picked = await vscode.window.showQuickPick(
        [...workspaces.map((w) => ({ label: w.name, description: w.pinned ? '$(pin) pinned' : undefined, workspace: w.name })), { label: '$(add) New workspace…', workspace: null }],
        { placeHolder: 'Add this folder to which workspace?' }
      );
      if (!picked) return undefined;
      workspace = picked.workspace;
      if (workspace === null) {
        workspace = await vscode.window.showInputBox({ prompt: 'New workspace name', value: 'VS Code' });
        if (!workspace) return undefined;
      }
    }
    const app = await service.addApp(project.dir, { workspace });
    refreshAll();
    vscode.window.showInformationMessage(`Added "${app.name}" to Workspace Manager.`);
    return app;
  }

  // ---- views, status bar and terminal env ----

  async function syncTerminalEnv() {
    const col = context.environmentVariableCollection;
    col.clear();
    for (const p of getProjects()) {
      let env;
      try {
        if (!(await service.findApp(p.dir))) continue;
        env = (await service.listEnv(p.dir)).env;
      } catch {
        continue;
      }
      const scoped = col.getScoped({ workspaceFolder: p.folder });
      scoped.clear();
      for (const [k, v] of Object.entries(env)) scoped.replace(k, String(v));
    }
  }

  async function updateStatusBar() {
    if (!client.connected) {
      setItem.text = '$(debug-disconnect) Workspace Manager';
      setItem.tooltip = 'Not connected to the Workspace Manager daemon. Click to retry.';
      setItem.command = 'workspaceManager.startDaemon';
      setItem.show();
      runItem.hide();
      return;
    }
    const project = selectedProject();
    if (!project) {
      setItem.hide();
      runItem.hide();
      return;
    }
    setItem.command = 'workspaceManager.switchSet';
    try {
      const app = await service.findApp(project.dir);
      if (!app) {
        setItem.text = '$(layers) Workspace Manager: not tracked';
        setItem.tooltip = 'This folder is not tracked by Workspace Manager.';
        setItem.command = 'workspaceManager.addProject';
        setItem.show();
        runItem.hide();
        return;
      }
      const prefix = getProjects().length > 1 ? `${project.label}: ` : '';
      setItem.text = `$(layers) ${prefix}${app.active_config_set_name ?? ''}`;
      setItem.tooltip = `Workspace Manager — active config set: ${app.active_config_set_name}\nClick to switch config set`;
      setItem.show();
      const st = tracker.get(app.id);
      if (st?.running) {
        runItem.text = `$(sync~spin) ${st.kind}`;
        runItem.tooltip = `${project.label} is running its ${st.kind} commands. Click to stop.`;
        runItem.show();
      } else {
        runItem.hide();
      }
    } catch {
      setItem.hide();
    }
  }

  let refreshTimer;
  function refreshAll() {
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(async () => {
      tree.refresh();
      templates.refreshOpen();
      await updateStatusBar();
      await syncTerminalEnv();
    }, 150);
  }

  async function reloadStatuses() {
    try {
      const apps = await service.apps();
      const wanted = new Set(getProjects().map((p) => normalizeDir(p.dir)));
      await tracker.load(apps.filter((a) => wanted.has(normalizeDir(a.project_path))));
    } catch (err) {
      output.appendLine(`cannot load run status: ${err?.message ?? err}`);
    }
  }

  client.on('event', (ev) => {
    if (ev.type === 'dataChanged') refreshAll();
    else if (ev.type === 'runnerEvent') tracker.handle(ev);
  });
  client.on('connected', () => {
    refreshAll();
    reloadStatuses();
  });
  client.on('disconnected', () => {
    updateStatusBar();
    tree.refresh();
  });

  // ---- command plumbing ----

  const ui = (id, fn) =>
    context.subscriptions.push(
      vscode.commands.registerCommand(`workspaceManager.${id}`, async (...args) => {
        try {
          return await fn(...args);
        } catch (err) {
          vscode.window.showErrorMessage(`Workspace Manager: ${err?.message ?? err}`);
        }
      })
    );

  async function pickSet(project, { placeHolder, includeActive = true, node } = {}) {
    if (node?.set) return node.set.name;
    const { sets } = await service.sets(project.dir);
    const items = sets.filter((s) => includeActive || !s.active).map((s) => ({ label: s.name, description: s.active ? 'active' : '' }));
    const picked = await vscode.window.showQuickPick(items, { placeHolder });
    return picked?.label;
  }

  async function pickParts(placeHolder) {
    const picked = await vscode.window.showQuickPick(
      [
        { label: 'Environment variables', part: 'env', picked: true },
        { label: 'Templates', part: 'templates', picked: true },
        { label: 'Setup commands', part: 'setup' },
        { label: 'Dev commands', part: 'dev' },
        { label: 'Build commands', part: 'build' },
        { label: 'Prod commands', part: 'prod' },
      ],
      { canPickMany: true, placeHolder }
    );
    if (!picked || picked.length === 0) return undefined;
    return Object.fromEntries(picked.map((p) => [p.part, true]));
  }

  // ---- general ----

  ui('refresh', async () => {
    tree.refresh();
    await reloadStatuses();
    refreshAll();
  });
  ui('addProject', (arg) => addProject(arg));
  ui('startDaemon', async () => {
    await client.connect();
    vscode.window.showInformationMessage('Connected to the Workspace Manager daemon.');
    refreshAll();
  });
  ui('showLog', async () => {
    const log = path.join(dataDir(cfg().get('dataDir') || undefined), 'wmd.log');
    if (fs.existsSync(log)) await vscode.window.showTextDocument(vscode.Uri.file(log));
    else output.show();
  });

  // ---- config sets ----

  ui('switchSet', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const name = await pickSet(project, { placeHolder: 'Switch active config set', node });
    if (!name) return;
    await service.switchSet(project.dir, name);
    refreshAll();
  });

  ui('createSet', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const name = await vscode.window.showInputBox({ prompt: 'New config set name', validateInput: (v) => (v.trim() ? undefined : 'Name is required') });
    if (!name) return;
    const { sets } = await service.sets(project.dir);
    let copyFrom;
    let parts;
    if (sets.length > 0) {
      const source = await vscode.window.showQuickPick(
        [{ label: 'Empty set', source: undefined }, ...sets.map((s) => ({ label: `Copy from ${s.name}`, source: s.name }))],
        { placeHolder: 'Start from' }
      );
      if (!source) return;
      copyFrom = source.source;
      if (copyFrom) {
        parts = await pickParts('What to copy');
        if (!parts) return;
      }
    }
    await service.createSet(project.dir, name, { copyFrom, parts });
    refreshAll();
  });

  ui('copyFromSet', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const source = await pickSet(project, { placeHolder: 'Copy from which config set?', includeActive: false });
    if (!source) return;
    const parts = await pickParts('What to copy into the active set');
    if (!parts) return;
    await service.copyFrom(project.dir, undefined, source, parts);
    refreshAll();
  });

  ui('renameSet', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const oldName = await pickSet(project, { placeHolder: 'Rename which config set?', node });
    if (!oldName) return;
    const newName = await vscode.window.showInputBox({ prompt: `Rename "${oldName}" to`, value: oldName });
    if (!newName || newName === oldName) return;
    await service.renameSet(project.dir, oldName, newName);
    refreshAll();
  });

  ui('deleteSet', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const name = await pickSet(project, { placeHolder: 'Delete which config set?', node });
    if (!name) return;
    const sure = await vscode.window.showWarningMessage(`Delete config set "${name}" and everything in it?`, { modal: true }, 'Delete');
    if (sure !== 'Delete') return;
    await service.deleteSet(project.dir, name);
    refreshAll();
  });

  // ---- environment variables ----

  ui('addEnv', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const key = await vscode.window.showInputBox({ prompt: 'Variable name', validateInput: (v) => (/^[A-Za-z_][A-Za-z0-9_.-]*$/.test(v) ? undefined : 'Letters, digits, _ . - ; not starting with a digit') });
    if (!key) return;
    const value = await vscode.window.showInputBox({ prompt: `Value of ${key}` });
    if (value === undefined) return;
    await service.setEnv(project.dir, { [key]: value });
    refreshAll();
  });

  ui('editEnv', async (node) => {
    const project = await trackedProject(node);
    if (!project || !node?.key) return;
    const value = await vscode.window.showInputBox({ prompt: `Value of ${node.key}`, value: node.value });
    if (value === undefined) return;
    await service.updateEnv(project.dir, node.key, node.key, value);
    refreshAll();
  });

  ui('deleteEnv', async (node) => {
    const project = await trackedProject(node);
    if (!project || !node?.key) return;
    await service.removeEnv(project.dir, node.key);
    refreshAll();
  });

  ui('importEnv', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const picked = await vscode.window.showOpenDialog({
      canSelectMany: false,
      openLabel: 'Import .env File',
      defaultUri: vscode.Uri.file(project.dir),
      filters: { 'Env Files': ['env', 'local', 'development', 'production', 'staging', 'example', 'template', 'yml', 'yaml'], 'All Files': ['*'] },
    });
    if (!picked?.length) return;
    const res = await service.importEnv(project.dir, picked[0].fsPath);
    refreshAll();
    vscode.window.showInformationMessage(`Imported ${res.imported} variable(s) and saved template "${res.filePath}".`);
    await templates.openTemplate(project.dir, res.set, res.filePath);
  });

  // ---- templates ----

  async function activeSetName(dir) {
    return (await service.sets(dir)).activeSet;
  }

  ui('snapshotFile', async (arg) => {
    const project = await trackedProject(arg);
    if (!project) return;
    let target;
    if (arg instanceof vscode.Uri && arg.scheme === 'file') target = arg.fsPath;
    else {
      const picked = await vscode.window.showOpenDialog({ canSelectMany: false, openLabel: 'Snapshot File as Template', defaultUri: vscode.Uri.file(project.dir) });
      if (!picked?.length) return;
      target = picked[0].fsPath;
    }
    const res = await service.snapshotFile(project.dir, target);
    refreshAll();
    await templates.openTemplate(project.dir, res.set, res.filePath);
  });

  ui('snapshotActiveEditor', async (node) => {
    const project = await trackedProject(node);
    if (!project) return;
    const doc = vscode.window.activeTextEditor?.document;
    if (!doc) throw new Error('No active text editor to snapshot.');
    if (doc.uri.scheme === TEMPLATE_SCHEME) {
      await doc.save();
      return;
    }
    if (doc.uri.scheme !== 'file') throw new Error('The active editor is not a file on disk.');
    const rel = toProjectRelative(project.dir, doc.uri.fsPath);
    const res = await service.setTemplate(project.dir, rel, doc.getText());
    refreshAll();
    vscode.window.showInformationMessage(`Saved "${res.filePath}" as a snapshot template in "${res.set}".`);
  });

  ui('editTemplate', async (node) => {
    if (!node?.filePath) return;
    await templates.openTemplate(node.project.dir, node.setName, node.filePath);
  });
  ui('previewTemplate', async (node) => {
    if (!node?.filePath) return;
    await templates.openPreview(node.project.dir, node.setName, node.filePath);
  });
  ui('reloadTemplate', async (node) => {
    if (!node?.filePath) return;
    await service.snapshotFile(node.project.dir, node.filePath, node.setName);
    refreshAll();
    vscode.window.showInformationMessage(`Updated template "${node.filePath}" from disk.`);
  });
  ui('deleteTemplate', async (node) => {
    if (!node?.filePath) return;
    await service.removeTemplate(node.project.dir, node.filePath, node.setName);
    refreshAll();
  });

  // ---- running ----

  async function startRun(kind, node) {
    const project = await trackedProject(node);
    if (!project) return;
    const app = await service.requireApp(project.dir);
    const config = await service.runConfig(project.dir, kind);
    if (!config?.commands?.length) {
      throw new Error(`No ${kind} commands are configured for "${await activeSetName(project.dir)}". Configure them in the Workspace Manager app.`);
    }
    tracker.showNextRun(app.id);
    try {
      await service.run(project.dir, kind);
    } catch (err) {
      tracker.autoShow.delete(app.id);
      throw err;
    }
  }
  for (const kind of KINDS) ui(`run${kind[0].toUpperCase()}${kind.slice(1)}`, (node) => startRun(kind, node));

  ui('stop', async (node) => {
    const project = await resolveProject(node);
    if (!project) return;
    await service.stop(project.dir);
  });
  ui('reload', async (node) => {
    const project = await resolveProject(node);
    if (!project) return;
    await service.reload(project.dir);
  });
  ui('showOutput', (node) => {
    if (node?.process) tracker.openOutput(node.appId, node.process, true);
  });
  ui('openUrl', (node) => {
    if (node?.url) vscode.env.openExternal(vscode.Uri.parse(node.url));
  });

  registerAgentCommands(context, service, getProjects, {
    afterRun: async (dir) => {
      const app = await service.findApp(dir);
      if (app) tracker.showNextRun(app.id);
    },
  });

  // ---- lifecycle ----

  context.subscriptions.push(
    vscode.workspace.onDidChangeWorkspaceFolders(() => {
      selectedDir = getProjects()[0]?.dir ?? '';
      refreshAll();
      reloadStatuses();
    }),
    vscode.window.onDidChangeActiveTextEditor(() => updateStatusBar()),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('workspaceManager')) {
        client.opts.dataDir = cfg().get('dataDir') || undefined;
        client.opts.daemonPath = cfg().get('daemonPath') || undefined;
        client.opts.autoStart = cfg().get('autoStartDaemon') !== false;
      }
    }),
    { dispose: () => client.close() }
  );

  client
    .connect()
    .then(() => refreshAll())
    .catch((err) => {
      output.appendLine(String(err?.message ?? err));
      updateStatusBar();
      vscode.window.showWarningMessage(`Workspace Manager: ${err?.message ?? err}`);
    });
}

export function deactivate() {}
