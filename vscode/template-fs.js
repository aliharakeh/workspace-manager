import * as vscode from 'vscode';

// Templates open in a native editor tab through a virtual file system
// (wm-template:), so syntax highlighting, undo and Ctrl+S all work: saving writes
// the template back to the daemon. A second scheme (wm-template-preview:) shows
// the template rendered with the config set's env vars, by the daemon's renderer.

export const TEMPLATE_SCHEME = 'wm-template';
export const PREVIEW_SCHEME = 'wm-template-preview';

// VS Code lowercases URI authorities, so the project dir travels base64url
// encoded and is remembered here to keep its original case.
const authorities = new Map();

function encodeAuthority(dir) {
  const encoded = Buffer.from(String(dir ?? ''), 'utf8').toString('base64url');
  authorities.set(encoded.toLowerCase(), dir);
  return encoded;
}

function decodeAuthority(authority) {
  const raw = String(authority ?? '');
  return authorities.get(raw.toLowerCase()) ?? Buffer.from(raw, 'base64url').toString('utf8');
}

const cleanFile = (file) => String(file ?? '').replace(/\\/g, '/').replace(/^\/+/, '');

function buildUri(scheme, dir, setName, file) {
  const f = cleanFile(file);
  const set = String(setName ?? '');
  return vscode.Uri.from({
    scheme,
    authority: encodeAuthority(dir),
    path: `/${encodeURIComponent(set)}/${f}`,
    query: `set=${encodeURIComponent(set)}&file=${encodeURIComponent(f)}`,
  });
}

export const buildTemplateUri = (dir, setName, file) => buildUri(TEMPLATE_SCHEME, dir, setName, file);
export const buildPreviewUri = (dir, setName, file) => buildUri(PREVIEW_SCHEME, dir, setName, file);

/**
 * The project dir, config set and template file a URI points at. They come from
 * the path (/<set>/<file>), because VS Code probes parent folders (/<set>) with
 * the same query; the query is only a fallback.
 */
export function parseTemplateUri(uri) {
  const trimmed = String(uri?.path ?? '').replace(/^\/+/, '');
  const slash = trimmed.indexOf('/');
  const safeDecode = (s) => {
    try {
      return decodeURIComponent(s);
    } catch {
      return s;
    }
  };
  let setName = safeDecode(slash === -1 ? trimmed : trimmed.slice(0, slash));
  let file = slash === -1 ? '' : trimmed.slice(slash + 1);
  if (!setName) setName = new URLSearchParams(uri?.query ?? '').get('set') ?? '';
  return { dir: decodeAuthority(uri?.authority), setName, file };
}

/**
 * Registers both providers. onSaved({dir, setName, file}) runs after a template
 * is saved from an editor.
 */
export function registerTemplateEditors(context, service, onSaved) {
  const fsEmitter = new vscode.EventEmitter();
  const previewEmitter = new vscode.EventEmitter();

  const read = async (uri) => {
    const { dir, setName, file } = parseTemplateUri(uri);
    try {
      return await service.getTemplate(dir, file, setName);
    } catch {
      throw vscode.FileSystemError.FileNotFound(uri);
    }
  };

  const fsProvider = {
    onDidChangeFile: fsEmitter.event,
    watch: () => new vscode.Disposable(() => {}),
    async stat(uri) {
      const { dir, setName, file } = parseTemplateUri(uri);
      const directory = { type: vscode.FileType.Directory, ctime: 0, mtime: Date.now(), size: 0 };
      if (!file || file === '/') return directory; // the root and a config set's folder
      try {
        const t = await service.getTemplate(dir, file, setName);
        return { type: vscode.FileType.File, ctime: 0, mtime: Date.now(), size: Buffer.byteLength(t.content, 'utf8') };
      } catch {
        // not a template: a folder when some template lives below it
        const { templates } = await service.listTemplates(dir, setName).catch(() => ({ templates: {} }));
        const prefix = `${file.replace(/\/+$/, '')}/`;
        if (Object.keys(templates).some((p) => p.startsWith(prefix))) return directory;
        throw vscode.FileSystemError.FileNotFound(uri);
      }
    },
    async readDirectory(uri) {
      const { dir, setName, file } = parseTemplateUri(uri);
      const prefix = file && file !== '/' ? `${file.replace(/\/+$/, '')}/` : '';
      const { templates } = await service.listTemplates(dir, setName).catch(() => ({ templates: {} }));
      const children = new Map();
      for (const p of Object.keys(templates)) {
        if (!p.startsWith(prefix)) continue;
        const rest = p.slice(prefix.length);
        const slash = rest.indexOf('/');
        children.set(slash === -1 ? rest : rest.slice(0, slash), slash === -1 ? vscode.FileType.File : vscode.FileType.Directory);
      }
      return [...children];
    },
    createDirectory() {},
    async readFile(uri) {
      return Buffer.from((await read(uri)).content, 'utf8');
    },
    async writeFile(uri, content) {
      const { dir, setName, file } = parseTemplateUri(uri);
      await service.setTemplate(dir, file, Buffer.from(content).toString('utf8'), setName);
      fsEmitter.fire([{ type: vscode.FileChangeType.Changed, uri }]);
      previewEmitter.fire(buildPreviewUri(dir, setName, file));
      await onSaved?.({ dir, setName, file });
    },
    async delete(uri) {
      const { dir, setName, file } = parseTemplateUri(uri);
      await service.removeTemplate(dir, file, setName);
      fsEmitter.fire([{ type: vscode.FileChangeType.Deleted, uri }]);
      previewEmitter.fire(buildPreviewUri(dir, setName, file));
    },
    async rename(oldUri, newUri) {
      const from = parseTemplateUri(oldUri);
      const to = parseTemplateUri(newUri);
      const t = await read(oldUri);
      await service.setTemplate(to.dir, to.file, t.content, to.setName);
      await service.removeTemplate(from.dir, from.file, from.setName);
      fsEmitter.fire([
        { type: vscode.FileChangeType.Deleted, uri: oldUri },
        { type: vscode.FileChangeType.Created, uri: newUri },
      ]);
    },
  };

  const previewProvider = {
    onDidChange: previewEmitter.event,
    async provideTextDocumentContent(uri) {
      try {
        const t = await read(uri);
        return await service.renderTemplate(t.configSetId, t.content);
      } catch (err) {
        return `Cannot render this template: ${err?.message ?? err}`;
      }
    },
  };

  context.subscriptions.push(
    vscode.workspace.registerFileSystemProvider(TEMPLATE_SCHEME, fsProvider, { isCaseSensitive: true }),
    vscode.workspace.registerTextDocumentContentProvider(PREVIEW_SCHEME, previewProvider),
    fsEmitter,
    previewEmitter
  );

  return {
    async openTemplate(dir, setName, file) {
      const doc = await vscode.workspace.openTextDocument(buildTemplateUri(dir, setName, file));
      return vscode.window.showTextDocument(doc, { preview: false });
    },
    async openPreview(dir, setName, file) {
      const doc = await vscode.workspace.openTextDocument(buildPreviewUri(dir, setName, file));
      return vscode.window.showTextDocument(doc, { viewColumn: vscode.ViewColumn.Beside, preview: true });
    },
    /** Re-reads open template and preview documents after the daemon's data changed. */
    refreshOpen() {
      for (const doc of vscode.workspace.textDocuments) {
        if (doc.uri.scheme === PREVIEW_SCHEME) previewEmitter.fire(doc.uri);
        else if (doc.uri.scheme === TEMPLATE_SCHEME && !doc.isDirty) fsEmitter.fire([{ type: vscode.FileChangeType.Changed, uri: doc.uri }]);
      }
    },
  };
}
