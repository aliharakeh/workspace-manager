import type { Context } from '@opencode/plugin/gui';
import { createEffect, createSignal, onCleanup, onMount, Show } from 'solid-js';
import { WorkspaceManager } from './contract';
import type { Terminal } from '@xterm/xterm';

interface Chunk { data: string; offset: number }
const bytes = (data: string) => Uint8Array.from(atob(data), (c) => c.charCodeAt(0));

export function Output(props: { ctx: Context; appID: number; commandID: number }) {
  const daemon = props.ctx.main(WorkspaceManager);
  const input = { appID: props.appID, commandID: props.commandID };
  const [error, setError] = createSignal('');
  let element!: HTMLDivElement;
  let terminal: Terminal | undefined;
  let disposed = false;
  let ready = false;
  let received = 0;
  let queue: Chunk[] = [];
  let generation = 0;
  let observer: ResizeObserver | undefined;

  function feed(chunk: Chunk) {
    if (!ready) { queue.push(chunk); return; }
    const data = bytes(chunk.data);
    const end = chunk.offset + data.length;
    if (end <= received) return;
    terminal?.write(data.subarray(Math.max(0, received - chunk.offset)));
    received = end;
  }

  async function catchUp() {
    const current = ++generation;
    ready = false;
    queue = [];
    try {
      const output = await daemon.output(input);
      if (disposed || current !== generation) return;
      terminal?.reset();
      terminal?.write(bytes(output.data));
      received = output.end;
      setError('');
    } catch (error) {
      if (!disposed && current === generation) setError(String(error));
    }
    if (disposed || current !== generation) return;
    ready = true;
    for (const chunk of queue.splice(0)) feed(chunk);
  }

  const unsubscribe = daemon.on('runner', (value) => {
    const event = value as { appId?: number; data?: Chunk & { type: string; commandId: number } };
    if (event.appId === input.appID && event.data?.type === 'log' && event.data.commandId === input.commandID) feed(event.data);
  });
  createEffect(() => {
    if (daemon.state()?.connected && terminal) void catchUp();
  });
  onMount(() => {
    void (async () => {
      // Construct terminals only in a window; main also evaluates this shared bundle.
      const [{ Terminal }, { FitAddon }] = await Promise.all([import('@xterm/xterm'), import('@xterm/addon-fit')]);
      if (disposed) return;
      terminal = new Terminal({ disableStdin: true, fontSize: 12, scrollback: 5000 });
      const fit = new FitAddon();
      terminal.loadAddon(fit);
      terminal.open(element);
      const resize = () => {
        if (!element.clientWidth || !element.clientHeight) return;
        fit.fit();
        void daemon.resize({ ...input, cols: terminal!.cols, rows: terminal!.rows }).catch((e) => setError(String(e)));
      };
      observer = new ResizeObserver(resize);
      observer.observe(element);
      resize();
      await catchUp();
    })().catch((e) => { if (!disposed) setError(String(e)); });
  });
  onCleanup(() => { disposed = true; unsubscribe(); observer?.disconnect(); terminal?.dispose(); });
  return <><Show when={error()}><p role="alert">{error()}</p></Show><div class="wm-terminal" ref={element} role="region" aria-label="Run command output" /></>;
}
