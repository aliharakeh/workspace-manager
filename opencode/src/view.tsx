import type { Context } from '@opencode/plugin/gui';
import { createResource, createSignal, For, onCleanup, Show } from 'solid-js';
import { WorkspaceManager } from './contract';
import type { Snapshot, Template } from './types';
import { Output } from './output';

const kinds = ['setup', 'dev', 'build', 'prod'];
const copyParts = ['env', 'templates', ...kinds];
type Field = { key: string; label: string; value?: string; multiline?: boolean; options?: string[] };

export function View(props: { dir: string; ctx: Context }) {
  const ctx = props.ctx;
  const ui = ctx.ui;
  const daemon = ctx.main(WorkspaceManager);
  const [error, setError] = createSignal('');
  const [busy, setBusy] = createSignal(false);
  const [command, setCommand] = createSignal<number>();
  const [data, { refetch }] = createResource(async () => {
    try { return await daemon.snapshot(props.dir) as Snapshot; }
    catch (error) { setError(String(error)); return undefined; }
  });
  let timer: ReturnType<typeof setTimeout>;
  const refresh = () => { clearTimeout(timer); timer = setTimeout(() => void refetch(), 120); };
  const unsubscribe = daemon.on('changed', refresh);
  onCleanup(() => { unsubscribe(); clearTimeout(timer); });

  async function perform(action: string, args: unknown[], expectedSetId = data()?.detail?.id) {
    const result = await daemon.act({ dir: props.dir, action, args, expectedSetId });
    refresh();
    return result;
  }

  async function run(fn: () => Promise<unknown>) {
    if (busy()) return;
    setBusy(true); setError('');
    try { await fn(); }
    catch (error) { setError(String(error)); }
    finally { setBusy(false); }
  }

  // Dialog drafts are independent of live snapshots, so another client's refresh cannot erase typing.
  function edit(title: string, fields: Field[], save: (values: Record<string, string>) => Promise<unknown>, parts = false) {
    const [values, setValues] = createSignal(Object.fromEntries(fields.map((f) => [f.key, f.value ?? ''])));
    const [failure, setFailure] = createSignal('');
    const [saving, setSaving] = createSignal(false);
    ctx.dialogs.open((dialog) => <ui.Dialog title={title} footer={<>
      <ui.Button disabled={saving()} onClick={dialog.close}>Cancel</ui.Button>
      <ui.Button variant="contrast" busy={saving()} onClick={() => {
        setSaving(true); setFailure('');
        void save(values()).then(() => dialog.close()).catch((e) => setFailure(String(e))).finally(() => setSaving(false));
      }}>Save</ui.Button>
    </>}>
      <div class="wm-panel wm-form">
        <For each={fields}>{(field) => <Show when={field.options} fallback={
          <label>{field.label}<Show when={field.multiline} fallback={
            <input value={values()[field.key]} onInput={(e) => setValues({ ...values(), [field.key]: e.currentTarget.value })} />
          }><textarea rows={12} spellcheck={false} value={values()[field.key]} onInput={(e) => setValues({ ...values(), [field.key]: e.currentTarget.value })} /></Show></label>
        }>
          <label>{field.label}<select value={values()[field.key]} onChange={(e) => setValues({ ...values(), [field.key]: e.currentTarget.value })}>
            <For each={field.options}>{(name) => <option value={name}>{name || 'Empty set'}</option>}</For>
          </select></label>
        </Show>}</For>
        <Show when={parts}><fieldset><legend>Parts to copy</legend><For each={copyParts}>{(part) =>
          <label class="wm-check"><input type="checkbox" checked={values()[part] === 'true'} onChange={(e) => setValues({ ...values(), [part]: String(e.currentTarget.checked) })} />{part}</label>
        }</For></fieldset></Show>
        <Show when={failure()}><p role="alert">{failure()}</p></Show>
      </div>
    </ui.Dialog>);
  }

  function confirm(title: string, save: () => Promise<unknown>) { edit(title, [], save); }
  const parts = (v: Record<string, string>) => Object.fromEntries(copyParts.filter((p) => v[p] === 'true').map((p) => [p, true]));

  function templateEditor(template?: Template) {
    const set = data()?.detail;
    if (!set) return;
    edit(template ? `Edit ${template.file_path}` : 'New template', [
      { key: 'file', label: 'Project-relative file path', value: template?.file_path },
      { key: 'content', label: 'Content (supports {{KEY}})', value: template?.content, multiline: true },
    ], async (v) => {
      if (template && v.file !== template.file_path) throw new Error('Create a new template to use a different path.');
      await perform('setTemplate', [v.file, v.content, set.name], set.id);
    });
  }

  function preview(template: Template) {
    const setId = data()!.detail!.id;
    void run(async () => {
      const rendered = await daemon.preview({ configSetId: setId, content: template.content });
      ctx.dialogs.open((dialog) => <ui.Dialog title={`Preview ${template.file_path}`} footer={<ui.Button onClick={dialog.close}>Close</ui.Button>}>
        <pre class="wm-preview">{rendered}</pre>
      </ui.Dialog>);
    });
  }

  return <div class="wm-panel">
    <header><div><strong>Workspace Manager</strong><small title={props.dir}>{props.dir}</small></div>
      <ui.Button size="small" busy={busy()} onClick={() => void run(async () => { await daemon.connect(); await refetch(); })}>Refresh</ui.Button>
    </header>
    <p role="status">{daemon.state()?.connected ? 'Connected to wmd' : 'Disconnected — Refresh to reconnect'}</p>
    <Show when={error() || data.error}><p role="alert">{error() || String(data.error)}</p></Show>
    <Show when={data.loading && !data()}><ui.Loader /></Show>
    <Show when={data()}>{(snapshot) => <Show when={snapshot().app} fallback={<section>
      <p>This folder is not tracked by Workspace Manager.</p>
      <ui.Button onClick={() => edit('Add folder to Workspace Manager', [
        { key: 'workspace', label: 'Existing or new workspace name', value: snapshot().workspaces?.[0]?.name || 'OpenCode' },
      ], async (v) => { await daemon.add({ dir: props.dir, workspace: v.workspace }); refresh(); })}>Add folder</ui.Button>
    </section>}>
      <section><h3>Config sets</h3>
        <label>Active config set <select value={snapshot().app!.active_config_set_name} disabled={busy()}
          onChange={(event) => void run(() => perform('switchSet', [event.currentTarget.value]))}>
          <For each={snapshot().sets}>{(set) => <option value={set.name}>{set.name}</option>}</For>
        </select></label>
        <div class="wm-actions">
          <ui.Button size="small" onClick={() => {
            const id = data()?.detail?.id;
            edit('New config set', [
              { key: 'name', label: 'Name' },
              { key: 'source', label: 'Copy from', options: ['', ...(snapshot().sets ?? []).map((s) => s.name)] },
            ], (v) => perform('createSet', [v.name, { copyFrom: v.source || undefined, parts: parts(v) }], id), true);
          }}>New</ui.Button>
          <ui.Button size="small" onClick={() => {
            const set = data()!.detail!;
            edit('Rename config set', [{ key: 'name', label: 'Name', value: set.name }], (v) => perform('renameSet', [set.name, v.name], set.id));
          }}>Rename</ui.Button>
          <ui.Button size="small" onClick={() => {
            const set = data()!.detail!;
            const sources = (snapshot().sets ?? []).filter((s) => s.id !== set.id).map((s) => s.name);
            edit('Copy parts into active set (replaces selected parts)', [{ key: 'source', label: 'Source set', value: sources[0], options: sources }],
              (v) => perform('copyFrom', [set.name, v.source, parts(v)], set.id), true);
          }}>Copy parts</ui.Button>
          <ui.Button size="small" variant="danger" onClick={() => {
            const set = data()!.detail!;
            confirm(`Delete config set "${set.name}"?`, () => perform('deleteSet', [set.name], set.id));
          }}>Delete</ui.Button>
        </div>
      </section>
      <section><h3>Environment</h3>
        <For each={snapshot().detail?.env_vars}>{(env) => <div class="wm-row">
          <code>{env.key}</code><span aria-label="Value hidden">••••••</span>
          <Show when={env.shared_var_id == null} fallback={
            <span title="Shared from the workspace's config: change it in Workspace Manager" aria-label="Shared from the workspace, read-only">shared · read-only</span>
          }>
            <ui.Button size="small" onClick={() => {
              const id = data()?.detail?.id;
              edit(`Edit ${env.key}`, [{ key: 'key', label: 'Key', value: env.key }, { key: 'value', label: 'Value', value: env.value }],
                (v) => perform('updateEnv', [env.key, v.key, v.value], id));
            }}>Edit</ui.Button>
            <ui.Button size="small" variant="danger" onClick={() => {
              const id = data()?.detail?.id;
              confirm(`Delete ${env.key}?`, () => perform('removeEnv', [[env.key]], id));
            }}>Delete</ui.Button>
          </Show>
        </div>}</For>
        <div class="wm-actions">
          <ui.Button size="small" onClick={() => {
            const id = data()?.detail?.id;
            edit('Add environment variable', [{ key: 'key', label: 'Key' }, { key: 'value', label: 'Value' }], (v) => perform('setEnv', [{ [v.key]: v.value }], id));
          }}>Add variable</ui.Button>
          <ui.Button size="small" onClick={() => {
            const id = data()?.detail?.id;
            edit('Import .env / YAML (also creates a template)', [{ key: 'file', label: 'Project-relative path', value: '.env' }], (v) => perform('importEnv', [v.file], id));
          }}>Import file</ui.Button>
        </div>
      </section>
      <section><h3>Snapshot templates</h3>
        <For each={snapshot().detail?.templates}>{(template) => <div class="wm-row">
          <code>{template.file_path}</code>
          <ui.Button size="small" onClick={() => templateEditor(template)}>Edit</ui.Button>
          <ui.Button size="small" onClick={() => preview(template)}>Preview</ui.Button>
          <ui.Button size="small" variant="danger" onClick={() => {
            const set = data()!.detail!;
            confirm(`Delete template "${template.file_path}"?`, () => perform('removeTemplate', [template.file_path, set.name], set.id));
          }}>Delete</ui.Button>
        </div>}</For>
        <div class="wm-actions">
          <ui.Button size="small" onClick={() => templateEditor()}>New template</ui.Button>
          <ui.Button size="small" onClick={() => {
            const set = data()!.detail!;
            edit('Snapshot file (replaces an existing template)', [{ key: 'file', label: 'Project-relative file path' }], (v) => perform('snapshotFile', [v.file, set.name], set.id));
          }}>Snapshot file</ui.Button>
        </div>
      </section>
      <section><h3>Run <small>{snapshot().status?.running ? snapshot().status?.kind : 'Stopped'}</small></h3>
        <div class="wm-actions"><For each={kinds}>{(kind) => <ui.Button size="small" disabled={busy() || snapshot().status?.running}
          onClick={() => void run(() => perform('run', [kind]))}>{kind}</ui.Button>}</For>
          <ui.Button size="small" disabled={busy() || !snapshot().status?.running} onClick={() => void run(() => perform('stop', []))}>Stop</ui.Button>
          <ui.Button size="small" disabled={busy() || !snapshot().status?.running} onClick={() => void run(() => perform('reload', []))}>Reload</ui.Button>
        </div>
        <Show when={snapshot().status?.error}><p role="alert">{snapshot().status?.error}</p></Show>
        <For each={snapshot().status?.processes}>{(process) => <div class="wm-process">
          <ui.Button size="small" onClick={() => setCommand(process.commandId)}>{process.label || process.command}</ui.Button>
          <small>{process.status}{process.exitCode !== null ? ` (${process.exitCode})` : ''}</small>
          <For each={process.urls}>{(url) => <Show when={/^https?:\/\//i.test(url)}><a href={url} onClick={(e) => { e.preventDefault(); void ctx.system.openExternal(url); }}>{url}</a></Show>}</For>
        </div>}</For>
        <Show when={snapshot().status?.processes.find((p) => p.commandId === command()) ?? snapshot().status?.processes[0]}>{(process) =>
          <Show when={`${snapshot().status?.sessionId}:${process().commandId}`} keyed>{(_key) =>
            <Output ctx={ctx} appID={snapshot().app!.id} commandID={process().commandId} />
          }</Show>
        }</Show>
      </section>
      <p class="wm-note">Configure run commands in the Workspace Manager desktop app. Runs continue when OpenCode closes.</p>
    </Show>}</Show>
  </div>;
}
