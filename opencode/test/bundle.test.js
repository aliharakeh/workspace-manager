import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { createRequire } from 'node:module';
import * as gui from '@opencode/plugin/gui';
import * as effect from 'effect';

test('GUI bundle evaluates and starts main without browser globals', async () => {
  const source = fs.readFileSync(new URL('../dist/gui-experimental.cjs', import.meta.url), 'utf8');
  const require = createRequire(import.meta.url);
  // Like the host, main supplies inert placeholders for window-only modules.
  const placeholder = new Proxy(function () {}, { get: () => placeholder, apply: () => placeholder, construct: () => placeholder });
  const exports = {};
  vm.runInNewContext(source, {
    exports, process, Buffer, setTimeout, clearTimeout, queueMicrotask, console,
    require: (name) => name === '@opencode/plugin/gui' ? gui : name === 'effect' ? effect : name.startsWith('solid-js') ? placeholder : require(name),
  });
  assert.equal(typeof exports.default.setup, 'function');
  let implementation;
  const dispose = await exports.default.main({
    options: { autoStartDaemon: false }, log: { write() {} },
    provide(token, impl) {
      assert.equal(token.id, 'workspace-manager-opencode.daemon');
      implementation = impl;
      return { changed() {}, emit() {} };
    },
  });
  assert.equal(implementation.state().connected, false);
  dispose();
});

test('GUI bundle evaluates in a window without Node globals or native modules', () => {
  const source = fs.readFileSync(new URL('../dist/gui-experimental.cjs', import.meta.url), 'utf8');
  const placeholder = new Proxy(function () {}, { get: () => placeholder, apply: () => placeholder, construct: () => placeholder });
  const exports = {};
  vm.runInNewContext(source, {
    exports, setTimeout, clearTimeout, queueMicrotask, console, document: placeholder,
    window: {}, navigator: { userAgent: 'Chrome', platform: 'Win32', language: 'en-US' },
    require: (name) => name === '@opencode/plugin/gui' ? gui : name === 'effect' ? effect : placeholder,
  });
  const contributions = [];
  exports.default.setup({
    id: 'workspace-manager-opencode',
    add(token, item) { contributions.push(item); },
    main() { return {}; },
  });
  assert.ok(contributions.some((item) => item.region === 'side'));
});
