import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { Script } from 'node:vm';

test('embedded HTML preserves executable bundled JavaScript without template expansion', async () => {
  const html = await readFile(new URL('../internal/mcpapp/studio.html', import.meta.url), 'utf8');
  assert.equal((html.match(/<!doctype html>/gi) ?? []).length, 1);
  assert.doesNotMatch(html, /<!-- UI_(?:JS|CSS) -->/);
  const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)];
  assert.equal(scripts.length, 1);
  // Compile the delivered script without executing its browser operations.
  assert.doesNotThrow(() => new Script(scripts[0][1], { filename: 'embedded-studio.js' }));
});
