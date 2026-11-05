import { Command, MenuItem, Panel, Plugin, Style } from '@opencode/plugin/gui';
import { WorkspaceManager } from './contract';
import { main } from './main';
import { View } from './view';
import { Show } from 'solid-js';
import css from './panel.css?inline';
import terminalCss from '@xterm/xterm/css/xterm.css?inline';

export default Plugin.define({
  setup(ctx) {
    ctx.add(Style, css + '\n@scope (.wm-panel) {\n' + terminalCss + '\n}');
    const tab = { id: 'main', title: 'Workspace Manager', closable: 'compact' as const, fallback: true };
    const open = () => {
      const session = ctx.sessions.current();
      if (session) ctx.layout.open(`${ctx.id}:main`, session);
    };
    ctx.add(Command, () => ({
      id: 'open', title: 'Open Workspace Manager', group: 'Workspace Manager',
      enabled: !!ctx.sessions.current() && ctx.app.platform === 'desktop', run: open,
    }));
    ctx.add(MenuItem, (): MenuItem => ({
      menu: 'session.panel', id: 'open', title: 'Workspace Manager', icon: 'server',
      keybind: `${ctx.id}.open`, run: open,
    }));
    ctx.add(Panel, {
      id: 'main', region: 'side',
      list: ({ open }) => open.includes('main') ? [tab] : [],
      render: (props) => <Show when={props.session.directory} keyed>{(dir) => <View dir={dir} ctx={ctx} />}</Show>,
    });
    const daemon = ctx.main(WorkspaceManager);
    ctx.add(Command, () => ({
      id: 'connect', title: 'Connect / Start Workspace Manager Daemon', group: 'Workspace Manager',
      run: async () => {
        try { await daemon.connect(); }
        catch (error) { ctx.toast.show({ title: String(error) }); }
      },
    }));
  },
  main,
});
