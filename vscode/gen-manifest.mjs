// Generates the contributes section of package.json from one table, so every
// command, menu entry and palette rule stays consistent. Run: npm run manifest
import fs from 'node:fs';

const CAT = 'Workspace Manager';
const cmd = (id, title, icon) => ({ command: `workspaceManager.${id}`, title, category: CAT, ...(icon ? { icon } : {}) });

const ui = [
  ['refresh', 'Refresh', '$(refresh)'],
  ['addProject', 'Add Folder to Workspace Manager', '$(add)'],
  ['startDaemon', 'Connect / Start Daemon', '$(plug)'],
  ['showLog', 'Show Daemon Log', '$(output)'],
  ['switchSet', 'Switch Config Set', '$(layers)'],
  ['createSet', 'New Config Set', '$(add)'],
  ['copyFromSet', 'Copy From Another Config Set', '$(copy)'],
  ['renameSet', 'Rename Config Set', '$(edit)'],
  ['deleteSet', 'Delete Config Set', '$(trash)'],
  ['addEnv', 'Add Environment Variable', '$(add)'],
  ['editEnv', 'Edit Environment Variable', '$(edit)'],
  ['deleteEnv', 'Delete Environment Variable', '$(trash)'],
  ['importEnv', 'Import .env File', '$(file-symlink-file)'],
  ['snapshotFile', 'Snapshot File as Template', '$(add)'],
  ['snapshotActiveEditor', 'Save Active File as Snapshot Template', '$(camera)'],
  ['editTemplate', 'Edit Template in Editor', '$(go-to-file)'],
  ['previewTemplate', 'Preview Rendered Template', '$(open-preview)'],
  ['reloadTemplate', 'Update Snapshot from Disk File', '$(sync)'],
  ['deleteTemplate', 'Delete Template', '$(trash)'],
  ['runSetup', 'Run Setup Commands', '$(tools)'],
  ['runDev', 'Run Dev Commands', '$(play)'],
  ['runBuild', 'Run Build Commands', '$(package)'],
  ['runProd', 'Run Prod Commands', '$(rocket)'],
  ['stop', 'Stop', '$(debug-stop)'],
  ['reload', 'Reload (Restart)', '$(debug-restart)'],
  ['showOutput', 'Show Command Output', '$(terminal)'],
  ['openUrl', 'Open URL', '$(link-external)'],
];

const agent = [
  ['listSets', 'List Sets'], ['switchSet', 'Switch Set'], ['createSet', 'Create Set'],
  ['renameSet', 'Rename Set'], ['copyFrom', 'Copy From Set'], ['deleteSet', 'Delete Set'],
  ['listEnv', 'List Env'], ['setEnv', 'Set Env'], ['removeEnv', 'Remove Env'], ['importEnv', 'Import Env'],
  ['listTemplates', 'List Templates'], ['getTemplate', 'Get Template'], ['setTemplate', 'Set Template'],
  ['snapshotFile', 'Snapshot File'], ['removeTemplate', 'Remove Template'],
  ['run', 'Run'], ['stop', 'Stop'], ['status', 'Status'],
];

const V = 'view == workspaceManager.view';
const item = (id, when, group) => ({ command: `workspaceManager.${id}`, when: `${V}${when ? ` && ${when}` : ''}`, group });
const vi = (kind) => `viewItem == ${kind}`;

const contributes = {
  commands: [
    ...ui.map(([id, title, icon]) => cmd(id, title, icon)),
    ...agent.map(([id, title]) => cmd(`agent.${id}`, `Agent: ${title}`)),
  ],
  viewsContainers: { activitybar: [{ id: 'workspaceManager', title: 'Workspace Manager', icon: 'media/icon.svg' }] },
  views: { workspaceManager: [{ id: 'workspaceManager.view', name: 'Workspace Manager', icon: 'media/icon.svg' }] },
  viewsWelcome: [{
    view: 'workspaceManager.view',
    contents: 'Config sets, environment variables, templates and run commands for this folder, stored by the Workspace Manager daemon.\n[Add Folder to Workspace Manager](command:workspaceManager.addProject)\n[Connect / Start Daemon](command:workspaceManager.startDaemon)',
  }],
  configuration: {
    title: 'Workspace Manager',
    properties: {
      'workspaceManager.daemonPath': { type: 'string', default: '', description: 'Path to wmd (the Workspace Manager daemon). Empty: the WM_DAEMON environment variable, then wmd on PATH.' },
      'workspaceManager.dataDir': { type: 'string', default: '', description: 'Workspace Manager data folder. Empty: WM_DATA_DIR, then the default for your OS.' },
      'workspaceManager.autoStartDaemon': { type: 'boolean', default: true, description: 'Start the daemon when it is not running.' },
    },
  },
  menus: {
    'view/title': [
      item('refresh', '', 'navigation@1'),
      item('addProject', '', 'navigation@2'),
      item('startDaemon', '', '1_daemon@1'),
      item('showLog', '', '1_daemon@2'),
      item('createSet', '', '2_sets@1'),
      item('switchSet', '', '2_sets@2'),
      item('importEnv', '', '3_templates@1'),
      item('snapshotFile', '', '3_templates@2'),
      item('snapshotActiveEditor', '', '3_templates@3'),
    ],
    'view/item/context': [
      item('addProject', vi('untracked'), 'inline@1'),
      item('switchSet', vi('project'), 'inline@1'),
      item('runDev', vi('project'), 'inline@2'),
      item('createSet', vi('setsGroup'), 'inline@1'),
      item('copyFromSet', vi('setsGroup'), '1_actions@1'),
      item('switchSet', vi('set'), 'inline@1'),
      item('renameSet', vi('set'), 'inline@2'),
      item('deleteSet', vi('set'), 'inline@3'),
      item('addEnv', vi('envGroup'), 'inline@1'),
      item('importEnv', vi('envGroup'), 'inline@2'),
      item('editEnv', vi('env'), 'inline@1'),
      item('deleteEnv', vi('env'), 'inline@2'),
      item('snapshotFile', vi('templatesGroup'), 'inline@1'),
      item('snapshotActiveEditor', vi('templatesGroup'), 'inline@2'),
      item('editTemplate', vi('template'), 'inline@1'),
      item('previewTemplate', vi('template'), 'inline@2'),
      item('reloadTemplate', vi('template'), 'inline@3'),
      item('deleteTemplate', vi('template'), 'inline@4'),
      item('runSetup', vi('runSetup'), 'inline@1'),
      item('runDev', vi('runDev'), 'inline@1'),
      item('runBuild', vi('runBuild'), 'inline@1'),
      item('runProd', vi('runProd'), 'inline@1'),
      item('stop', 'viewItem =~ /^run.*Active$/', 'inline@1'),
      item('reload', 'viewItem == runDevActive || viewItem == runProdActive', 'inline@2'),
      item('showOutput', vi('process'), 'inline@1'),
      item('openUrl', vi('url'), 'inline@1'),
    ],
    'explorer/context': [{ command: 'workspaceManager.snapshotFile', when: '!explorerResourceIsFolder', group: 'navigation' }],
    'editor/title/context': [{ command: 'workspaceManager.snapshotFile', when: 'resourceScheme == file', group: '1_modification' }],
    commandPalette: [
      ...['editEnv', 'deleteEnv', 'editTemplate', 'previewTemplate', 'reloadTemplate', 'deleteTemplate', 'showOutput', 'openUrl']
        .map((id) => ({ command: `workspaceManager.${id}`, when: 'false' })),
      ...agent.map(([id]) => ({ command: `workspaceManager.agent.${id}`, when: 'false' })),
    ],
  },
};

const pkgPath = new URL('./package.json', import.meta.url);
const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
pkg.contributes = contributes;
fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
console.log(`wrote ${contributes.commands.length} commands`);
