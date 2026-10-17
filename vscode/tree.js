import * as vscode from 'vscode';
import { KINDS } from './service.js';

const KIND_LABEL = { setup: 'Setup', dev: 'Dev', build: 'Build', prod: 'Prod' };
const KIND_ICON = { setup: 'tools', dev: 'play', build: 'package', prod: 'rocket' };
const cap = (s) => s[0].toUpperCase() + s.slice(1);

const PLACEHOLDER_RE = /\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}/g;
const placeholders = (content) => [...new Set([...String(content ?? '').matchAll(PLACEHOLDER_RE)].map((m) => m[1]))];

/**
 * The sidebar: each workspace folder with its config sets, env vars, templates and
 * run commands, all read from the daemon. Folders the daemon does not track show an
 * "add" action instead.
 */
export class WmTree {
  /**
   * @param {import('./service.js').Service} service
   * @param {import('./runner.js').RunTracker} tracker
   * @param {() => Array<{label: string, dir: string, folder: object}>} getProjects
   */
  constructor(service, tracker, getProjects) {
    this.service = service;
    this.tracker = tracker;
    this.getProjects = getProjects;
    this.emitter = new vscode.EventEmitter();
    this.onDidChangeTreeData = this.emitter.event;
    this.cache = new Map(); // dir -> Promise<{app, sets, detail} | null>
  }

  refresh() {
    this.cache.clear();
    this.emitter.fire();
  }

  /** The folder's app, active set and its details; null when the folder is not tracked. */
  load(dir) {
    if (!this.cache.has(dir)) {
      this.cache.set(
        dir,
        (async () => {
          const app = await this.service.findApp(dir);
          if (!app) return null;
          const { sets, activeSet } = await this.service.sets(dir);
          const { detail, set } = await this.service.detail(dir);
          return { app, sets, activeSet, set, detail };
        })()
      );
    }
    return this.cache.get(dir);
  }

  async getChildren(node) {
    if (!this.service.client.connected) {
      try {
        await this.service.client.connect();
      } catch {
        return [];
      }
    }
    const projects = this.getProjects();
    if (!node) {
      if (projects.length === 0) return [];
      if (projects.length === 1) return this.#projectChildren(projects[0]);
      return projects.map((project) => ({ type: 'project', project }));
    }
    const { project } = node;
    const data = project && (await this.load(project.dir));
    switch (node.type) {
      case 'project':
        return this.#projectChildren(project);
      case 'setsGroup':
        return data.sets.map((set) => ({ type: 'set', project, set }));
      case 'envGroup':
        return data.detail.env_vars.map((v) => ({ type: 'env', project, setName: data.set.name, key: v.key, value: v.value }));
      case 'templatesGroup':
        return data.detail.templates.map((t) => ({
          type: 'template',
          project,
          setName: data.set.name,
          filePath: t.file_path,
          placeholders: placeholders(t.content),
        }));
      case 'runGroup':
        return KINDS.map((kind) => ({ type: 'runKind', project, kind }));
      case 'runKind': {
        const st = this.tracker.get(data.app.id);
        if (!st?.running || st.kind !== node.kind) return [];
        return (st.processes ?? []).map((p) => ({ type: 'process', project, appId: data.app.id, process: p }));
      }
      case 'process':
        return (node.process.urls ?? []).map((url) => ({ type: 'url', project, url }));
      default:
        return [];
    }
  }

  async #projectChildren(project) {
    const data = await this.load(project.dir);
    if (!data) return [{ type: 'untracked', project }];
    return [
      { type: 'setsGroup', project },
      { type: 'envGroup', project },
      { type: 'templatesGroup', project },
      { type: 'runGroup', project },
    ];
  }

  async getTreeItem(node) {
    const { Expanded: expanded, Collapsed: collapsed, None: none } = vscode.TreeItemCollapsibleState;
    const data = node.project ? await this.load(node.project.dir).catch(() => null) : null;
    let item;

    switch (node.type) {
      case 'project': {
        item = new vscode.TreeItem(node.project.label, expanded);
        item.description = data ? `[${data.activeSet}]${this.#runningLabel(data.app.id)}` : 'not tracked';
        item.tooltip = `${node.project.dir}\n${data ? `Active config set: ${data.activeSet}` : 'Not tracked by Workspace Manager'}`;
        item.iconPath = new vscode.ThemeIcon('root-folder');
        break;
      }
      case 'untracked': {
        item = new vscode.TreeItem('Not tracked by Workspace Manager', none);
        item.description = 'click to add';
        item.iconPath = new vscode.ThemeIcon('add');
        item.command = { command: 'workspaceManager.addProject', title: 'Add Folder', arguments: [node] };
        break;
      }
      case 'setsGroup': {
        item = new vscode.TreeItem('Config Sets', expanded);
        item.description = `Active: ${data?.activeSet ?? ''}`;
        item.iconPath = new vscode.ThemeIcon('layers');
        break;
      }
      case 'set': {
        item = new vscode.TreeItem(node.set.name, none);
        item.description = node.set.active ? 'active' : '';
        item.tooltip = `${node.set.name}${node.set.active ? ' (active)' : ''}\nClick to switch to this config set.`;
        item.iconPath = new vscode.ThemeIcon(node.set.active ? 'pass-filled' : 'circle-large-outline');
        item.command = { command: 'workspaceManager.switchSet', title: 'Switch Config Set', arguments: [node] };
        break;
      }
      case 'envGroup': {
        const count = data?.detail.env_vars.length ?? 0;
        item = new vscode.TreeItem('Environment Variables', expanded);
        item.description = `${count}`;
        item.tooltip = `Environment variables in "${data?.set.name}" — injected into integrated terminals and into runs`;
        item.iconPath = new vscode.ThemeIcon('symbol-variable');
        break;
      }
      case 'env': {
        item = new vscode.TreeItem(node.key, none);
        item.description = `= ${node.value}`;
        item.tooltip = `${node.key}=${node.value}\nClick to edit`;
        item.iconPath = new vscode.ThemeIcon('symbol-key');
        item.command = { command: 'workspaceManager.editEnv', title: 'Edit Environment Variable', arguments: [node] };
        break;
      }
      case 'templatesGroup': {
        const count = data?.detail.templates.length ?? 0;
        item = new vscode.TreeItem('Snapshot Templates', expanded);
        item.description = `${count}`;
        item.tooltip = `Snapshot file templates in "${data?.set.name}". They are rendered into the project when a command runs and restored afterwards.`;
        item.iconPath = new vscode.ThemeIcon('files');
        break;
      }
      case 'template': {
        item = new vscode.TreeItem(node.filePath, none);
        item.description = node.placeholders.length ? node.placeholders.map((p) => `{{${p}}}`).join(', ') : 'no placeholders';
        item.tooltip = `Snapshot template: ${node.filePath} (${node.setName})\nClick to edit; Ctrl+S saves it to the daemon.`;
        item.iconPath = new vscode.ThemeIcon('file-code');
        item.command = { command: 'workspaceManager.editTemplate', title: 'Edit Template', arguments: [node] };
        break;
      }
      case 'runGroup': {
        item = new vscode.TreeItem('Run', expanded);
        item.description = this.#runningLabel(data?.app.id).trim();
        item.iconPath = new vscode.ThemeIcon('debug-start');
        break;
      }
      case 'runKind': {
        const cfg = data?.detail[`${node.kind}_config`];
        const count = cfg?.commands?.length ?? 0;
        const st = data && this.tracker.get(data.app.id);
        const active = Boolean(st?.running && st.kind === node.kind);
        item = new vscode.TreeItem(KIND_LABEL[node.kind], active ? expanded : none);
        item.description = active ? `running · ${count} command${count === 1 ? '' : 's'}` : count ? `${count} command${count === 1 ? '' : 's'}` : 'no commands';
        item.tooltip = count ? cfg.commands.map((c) => (c.label ? `${c.label}: ${c.command}` : c.command)).join('\n') : 'No commands configured in this config set. Configure them in the Workspace Manager app.';
        item.iconPath = new vscode.ThemeIcon(active ? 'sync~spin' : KIND_ICON[node.kind]);
        item.contextValue = `run${cap(node.kind)}${active ? 'Active' : ''}`;
        item.id = this.#id(node, data);
        return item;
      }
      case 'process': {
        const p = node.process;
        item = new vscode.TreeItem(p.label || p.command, (p.urls?.length ?? 0) > 0 ? expanded : none);
        item.description = p.status === 'exited' && p.exitCode != null ? `exited (${p.exitCode})` : p.status;
        item.tooltip = p.command;
        item.iconPath = new vscode.ThemeIcon(p.status === 'running' ? 'circle-filled' : p.exitCode ? 'error' : 'circle-outline');
        item.command = { command: 'workspaceManager.showOutput', title: 'Show Output', arguments: [node] };
        break;
      }
      case 'url': {
        item = new vscode.TreeItem(node.url, none);
        item.iconPath = new vscode.ThemeIcon('link');
        item.command = { command: 'workspaceManager.openUrl', title: 'Open URL', arguments: [node] };
        break;
      }
    }

    item.id = this.#id(node, data);
    item.contextValue = node.type;
    return item;
  }

  #runningLabel(appId) {
    const st = appId != null ? this.tracker.get(appId) : undefined;
    return st?.running ? ` · ${st.kind} running` : '';
  }

  #id(node, data) {
    const dir = node.project?.dir ?? '';
    switch (node.type) {
      case 'set':
        return `set:${dir}:${node.set.name}`;
      case 'env':
        return `env:${dir}:${node.setName}:${node.key}`;
      case 'template':
        return `template:${dir}:${node.setName}:${node.filePath}`;
      case 'runKind':
        return `run:${dir}:${node.kind}`;
      case 'process':
        return `proc:${dir}:${node.process.commandId}`;
      case 'url':
        return `url:${dir}:${node.url}`;
      default:
        return `${node.type}:${dir}`;
    }
  }
}
