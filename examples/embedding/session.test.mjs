import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, copyFile, readFile, readdir, rm, symlink } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createStudioSession, relativeImagePath, validateStudioCall } from './session.mjs';

const repository = fileURLToPath(new URL('../..', import.meta.url));
const binary = path.join(repository, 'bin', process.platform === 'win32' ? 'dither-mcp.exe' : 'dither-mcp');

test('embedding grants only two tools for its selected source and relative output', () => {
  for (const value of ['../source.png', '/source.png', 'file:///tmp/source.png', 'C:\\source.png', 'a/../b.png', 'a\\b.png', './source.png', '', 'a//b.png']) assert.throws(() => relativeImagePath(value));
  const policy = { input: 'source.png' };
  const request = { name: 'dither_render', arguments: { input: 'source.png', output: 'art/result.png', expected_source_sha256: 'a'.repeat(64), options: { width: 1600 } }, _meta: { example: 'preserve' } };
  assert.equal(validateStudioCall(request, policy), request);
  for (const call of [
    { name: 'dither_batch', arguments: {} },
    { name: 'dither_studio', arguments: { input: 'other.png' } },
    { name: 'dither_studio', arguments: { input: 'source.png', mask_input: 'mask.png' } },
    { name: 'dither_studio', arguments: { input: 'source.png', recipe: 'recipe.json' } },
    { name: 'dither_render', arguments: { input: 'source.png', output: 'source.png' } },
    { name: 'dither_render', arguments: { input: 'source.png', output: '../result.png' } },
  ]) assert.throws(() => validateStudioCall(call, policy));
});

test('reusable embedding session negotiates Apps and preserves guarded export results over real stdio', { timeout: 30_000 }, async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  const session = await createStudioSession({ binary, root, input: 'source.png', initial: { options: { width: 96, algorithm: 'atkinson' } } });
  t.after(() => session.close());
  const { resource, result } = session.bootstrap;
  assert.equal(resource.mimeType, 'text/html;profile=mcp-app');
  assert.deepEqual(resource._meta.ui.csp.connectDomains, []);
  assert.equal(result._meta.dither.algorithms.length, 41);
  assert.equal(result._meta.dither.palettes.length, 256);
  assert.match(result.structuredContent.source_sha256, /^[a-f0-9]{64}$/);
  assert.deepEqual(await readdir(root), ['source.png']);
  const args = { ...result._meta.dither.request, output: 'saved.png', expected_source_sha256: result.structuredContent.source_sha256 };
  const saved = await session.callTool({ name: 'dither_render', arguments: args });
  assert.ok(!saved.isError);
  assert.deepEqual(await readFile(path.join(root, 'saved.png')), Buffer.from(result.content.find((entry) => entry.type === 'image').data, 'base64'));
  const changed = await session.callTool({ name: 'dither_render', arguments: { ...args, output: 'blocked.png', expected_source_sha256: '0'.repeat(64) } });
  assert.equal(changed.isError, true);
  assert.equal(changed.structuredContent.error.code, 'source_changed');
  assert.deepEqual((await readdir(root)).sort(), ['saved.png', 'source.png']);
  await assert.rejects(session.callTool({ name: 'dither_studio', arguments: { input: 'other.png' } }), /source/);
});

test('embedding session rejects an initial source escaping root through a symlink', async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-path-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await symlink(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  await assert.rejects(createStudioSession({ binary, root, input: 'source.png' }), /inside root/);
});
