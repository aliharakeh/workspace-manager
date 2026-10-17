import net from 'node:net';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { EventEmitter } from 'node:events';

// A client of the wmd daemon. Node cannot open the daemon's unix socket on
// Windows, so it uses the loopback TCP endpoint the daemon publishes in its data
// folder (wmd.endpoint: address + token). The protocol is the same: one JSON
// message per line, {id, method, params} -> {id, result | error}, plus pushed
// {event, data} notifications. Nothing here imports vscode, so it is tested
// against a real daemon.

/** The data folder the daemon uses (same rules as api.DataDir in Go). */
export function dataDir(override) {
  if (override) return override;
  if (process.env.WM_DATA_DIR) return process.env.WM_DATA_DIR;
  const home = os.homedir();
  switch (process.platform) {
    case 'win32':
      return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'workspace-manager');
    case 'darwin':
      return path.join(home, 'Library', 'Application Support', 'workspace-manager');
    default:
      return path.join(process.env.XDG_DATA_HOME || path.join(home, '.local', 'share'), 'workspace-manager');
  }
}

/** An error reported by the daemon; kind is one of invalid, not_found, conflict, unavailable, failed. */
export class DaemonError extends Error {
  constructor(message, kind = 'failed') {
    super(message);
    this.name = 'DaemonError';
    this.kind = kind;
  }
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export class DaemonClient extends EventEmitter {
  /**
   * @param {object} opts
   * @param {string} [opts.dataDir] data folder override
   * @param {string} [opts.daemonPath] path to wmd; empty: WM_DAEMON, then PATH
   * @param {boolean} [opts.autoStart] start the daemon when it is not running (default true)
   * @param {string} [opts.version] client version sent in hello
   * @param {(line: string) => void} [opts.log]
   */
  constructor(opts = {}) {
    super();
    this.opts = { autoStart: true, version: 'dev', ...opts };
    this.socket = null;
    this.nextId = 0;
    this.pending = new Map();
    this.connecting = null;
    this.closed = false;
    this.buffer = '';
    this.reconnectTimer = null;
  }

  get dir() {
    return dataDir(this.opts.dataDir);
  }

  get connected() {
    return Boolean(this.socket) && !this.socket.destroyed;
  }

  log(line) {
    this.opts.log?.(line);
  }

  /** Connects (starting the daemon when needed) unless already connected. */
  connect() {
    if (this.connected) return Promise.resolve();
    if (!this.connecting) {
      this.closed = false;
      this.connecting = this.#connect().finally(() => {
        this.connecting = null;
      });
    }
    return this.connecting;
  }

  async #connect() {
    let started = false;
    let lastError;
    for (let attempt = 0; attempt < 40; attempt++) {
      try {
        await this.#dial();
        return;
      } catch (err) {
        lastError = err;
        if (err instanceof DaemonError && err.kind === 'unavailable') throw err; // refused by the daemon, not a missing one
      }
      if (!started && this.opts.autoStart) {
        started = true;
        this.log('daemon not reachable, starting it');
        await this.#startDaemon();
      } else if (!this.opts.autoStart && attempt >= 1) {
        break;
      }
      await sleep(250);
    }
    throw new DaemonError(
      `cannot reach the Workspace Manager daemon (${lastError?.message ?? 'unknown error'}). Is wmd installed? Set workspaceManager.daemonPath.`,
      'unavailable'
    );
  }

  #readEndpoint() {
    const raw = fs.readFileSync(path.join(this.dir, 'wmd.endpoint'), 'utf8');
    const ep = JSON.parse(raw);
    if (!ep.addr || !ep.token) throw new Error('invalid endpoint file');
    return ep;
  }

  async #dial() {
    const ep = this.#readEndpoint();
    const [host, port] = [ep.addr.slice(0, ep.addr.lastIndexOf(':')), Number(ep.addr.slice(ep.addr.lastIndexOf(':') + 1))];
    const socket = await new Promise((resolve, reject) => {
      const s = net.connect({ host, port });
      s.once('connect', () => resolve(s));
      s.once('error', reject);
    });
    socket.setEncoding('utf8');
    this.buffer = '';
    socket.on('data', (chunk) => this.#onData(chunk));
    socket.on('error', () => {});
    socket.on('close', () => this.#onClose(socket));
    this.socket = socket;
    try {
      await this.call('hello', { kind: 'gui', name: 'vscode', version: this.opts.version, token: ep.token });
      await this.call('events.subscribe', {});
    } catch (err) {
      socket.destroy();
      this.socket = null;
      throw err;
    }
    this.log('connected to the daemon');
    this.emit('connected');
  }

  async #startDaemon() {
    const exe = this.opts.daemonPath || process.env.WM_DAEMON || (process.platform === 'win32' ? 'wmd.exe' : 'wmd');
    const env = { ...process.env };
    if (this.opts.dataDir) env.WM_DATA_DIR = this.opts.dataDir;
    await new Promise((resolve, reject) => {
      const child = spawn(exe, [], { detached: true, stdio: 'ignore', env, windowsHide: true });
      child.once('error', (err) => reject(new DaemonError(`cannot start ${exe}: ${err.message}`, 'unavailable')));
      child.once('spawn', () => {
        child.unref();
        resolve();
      });
    });
  }

  #onData(chunk) {
    this.buffer += chunk;
    let nl;
    while ((nl = this.buffer.indexOf('\n')) >= 0) {
      const line = this.buffer.slice(0, nl).trim();
      this.buffer = this.buffer.slice(nl + 1);
      if (!line) continue;
      let msg;
      try {
        msg = JSON.parse(line);
      } catch {
        continue;
      }
      if (msg.event) {
        this.emit('event', msg.data ?? {});
      } else if (msg.id !== undefined) {
        const waiter = this.pending.get(msg.id);
        if (!waiter) continue;
        this.pending.delete(msg.id);
        if (msg.error) waiter.reject(new DaemonError(msg.error.message, msg.error.kind || 'failed'));
        else waiter.resolve(msg.result);
      }
    }
  }

  #onClose(socket) {
    if (this.socket !== socket) return;
    this.socket = null;
    for (const waiter of this.pending.values()) waiter.reject(new DaemonError('connection to the daemon closed', 'unavailable'));
    this.pending.clear();
    this.emit('disconnected');
    if (!this.closed) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = setTimeout(() => {
        this.connect().catch(() => {});
      }, 1000);
    }
  }

  /** Runs a daemon method and resolves with its result. */
  async call(method, params) {
    if (method !== 'hello' && !this.connected) await this.connect();
    const socket = this.socket;
    if (!socket) throw new DaemonError('not connected to the daemon', 'unavailable');
    const id = ++this.nextId;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      socket.write(JSON.stringify({ id, method, ...(params === undefined ? {} : { params }) }) + '\n', (err) => {
        if (err) {
          this.pending.delete(id);
          reject(new DaemonError(err.message, 'unavailable'));
        }
      });
    });
  }

  /** Stops the running daemon (used by tests). */
  stopDaemon() {
    return this.call('server.stop');
  }

  close() {
    this.closed = true;
    clearTimeout(this.reconnectTimer);
    this.socket?.destroy();
    this.socket = null;
  }
}
