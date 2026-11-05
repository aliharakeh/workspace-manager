import type { Plugin } from '@opencode/plugin/gui';
import { WorkspaceManager } from './contract';
import { act, createBackend, snapshot } from './backend.js';

export const main: Plugin.MainSetup = (ctx) => {
  const { client, service } = createBackend(ctx.options, (line: string) => ctx.log.write('info', line));
  const provider = ctx.provide(WorkspaceManager, {
    state: () => ({ connected: client.connected }),
    connect: () => client.connect(),
    snapshot: (dir) => snapshot(service, dir),
    act: (input) => act(service, input),
    add: ({ dir, workspace }) => service.addApp(dir, { workspace: workspace.trim() || 'OpenCode' }),
    preview: ({ configSetId, content }) => service.renderTemplate(configSetId, content),
    output: (input) => client.call('runner.output', input),
    resize: async (input) => { await client.call('runner.resize', input); },
  });
  client.on('connected', () => { provider.changed(); provider.emit('changed', undefined); });
  client.on('disconnected', () => provider.changed());
  client.on('event', (event) => {
    if (event.type === 'dataChanged') provider.emit('changed', undefined);
    if (event.type === 'runnerEvent') {
      provider.emit('runner', event);
      if (event.data?.type === 'status') provider.emit('changed', undefined);
    }
  });
  return () => { client.close(); client.removeAllListeners(); };
};
