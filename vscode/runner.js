import * as vscode from 'vscode';

// Run state and terminal output for the apps the daemon runs. The daemon owns
// the processes; this keeps the latest status per app from the daemon's events
// and shows a command's output in a VS Code terminal (a pseudoterminal fed with
// the daemon's raw terminal bytes, so colors and progress bars work).

/** Tracks the runner status of every app and the output terminals opened for them. */
export class RunTracker {
  /**
   * @param {import('./service.js').Service} service
   * @param {() => void} onChange called when any status changes
   */
  constructor(service, onChange) {
    this.service = service;
    this.onChange = onChange;
    this.statuses = new Map(); // appId -> StatusEvent
    this.terminals = new Map(); // `${appId}:${commandId}` -> OutputTerminal
    this.autoShow = new Set(); // appIds whose next run should open its terminals
  }

  get(appId) {
    return this.statuses.get(appId);
  }

  /** Loads the current status of the given apps (after connecting). */
  async load(apps) {
    await Promise.all(
      apps.map(async (app) => {
        try {
          this.statuses.set(app.id, await this.service.call('runner.status', { appID: app.id }));
        } catch {
          // the app may have been removed meanwhile
        }
      })
    );
    this.onChange();
  }

  /** Handles a pushed runnerEvent: {appId, data: StatusEvent | LogEvent}. */
  handle(ev) {
    const data = ev.data ?? {};
    const appId = ev.appId ?? data.appId;
    if (data.type === 'status') {
      this.statuses.set(appId, data);
      if (data.running && this.autoShow.has(appId) && data.processes?.length) {
        this.autoShow.delete(appId);
        data.processes.forEach((p, i) => this.openOutput(appId, p, i === 0));
      }
      this.onChange();
    } else if (data.type === 'log') {
      this.terminals.get(`${appId}:${data.commandId}`)?.feed(data);
    }
  }

  /** The next run of this app opens its output terminals by itself. */
  showNextRun(appId) {
    this.autoShow.add(appId);
  }

  /** Shows (creating if needed) the terminal of one command of the app's current run. */
  openOutput(appId, process, focus = true) {
    const key = `${appId}:${process.commandId}`;
    let entry = this.terminals.get(key);
    if (!entry || entry.disposed) {
      entry = new OutputTerminal(this.service, appId, process.commandId, () => this.terminals.delete(key));
      entry.terminal = vscode.window.createTerminal({
        name: `wm: ${process.label || process.command}`.slice(0, 60),
        pty: entry,
        isTransient: true,
      });
      this.terminals.set(key, entry);
    }
    entry.terminal.show(!focus);
  }

  dispose() {
    for (const t of this.terminals.values()) t.terminal?.dispose();
    this.terminals.clear();
  }
}

/** A read-only terminal showing one run command's output. */
class OutputTerminal {
  constructor(service, appId, commandId, onClose) {
    this.service = service;
    this.appId = appId;
    this.commandId = commandId;
    this.onClose = onClose;
    this.disposed = false;
    this.writeEmitter = new vscode.EventEmitter();
    this.onDidWrite = this.writeEmitter.event;
    this.decoder = new TextDecoder('utf-8');
    this.received = 0; // bytes already shown
    this.ready = false;
    this.queue = [];
    this.terminal = undefined;
  }

  async open() {
    try {
      const out = await this.service.call('runner.output', { appID: this.appId, commandID: this.commandId });
      const bytes = Buffer.from(out.data || '', 'base64');
      if (bytes.length) this.#write(bytes);
      this.received = out.end ?? bytes.length;
    } catch (err) {
      this.writeEmitter.fire(`\x1b[31mCannot load the output: ${err?.message ?? err}\x1b[0m\r\n`);
    }
    this.ready = true;
    for (const chunk of this.queue.splice(0)) this.feed(chunk);
  }

  close() {
    this.disposed = true;
    this.writeEmitter.dispose();
    this.onClose();
  }

  setDimensions(dims) {
    this.service.call('runner.resize', { appID: this.appId, commandID: this.commandId, cols: dims.columns, rows: dims.rows }).catch(() => {});
  }

  /** A live LogEvent: Data is base64, Offset the bytes the command produced before it. */
  feed(ev) {
    if (!this.ready) {
      this.queue.push(ev);
      return;
    }
    const bytes = Buffer.from(ev.data || '', 'base64');
    const end = ev.offset + bytes.length;
    if (end <= this.received) return; // already shown by the catch-up
    const fresh = ev.offset < this.received ? bytes.subarray(this.received - ev.offset) : bytes;
    this.received = end;
    this.#write(fresh);
  }

  #write(bytes) {
    this.writeEmitter.fire(this.decoder.decode(bytes, { stream: true }));
  }
}
