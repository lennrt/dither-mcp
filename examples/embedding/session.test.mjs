import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, copyFile, readFile, readdir, rm, symlink } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Client } from '@modelcontextprotocol/client';
import { StudioState } from '../../ui/state.mjs';
import { createStudioSession, relativeImagePath, validateStudioCall } from './session.mjs';

const repository = fileURLToPath(new URL('../..', import.meta.url));
const binary = path.join(repository, 'bin', process.platform === 'win32' ? 'dither-mcp.exe' : 'dither-mcp');

test('embedding grants only five tools for its selected source and relative output', () => {
  for (const value of ['../source.png', '/source.png', 'file:///tmp/source.png', 'C:\\source.png', 'a/../b.png', 'a\\b.png', './source.png', '', 'a//b.png']) assert.throws(() => relativeImagePath(value));
  const policy = { input: 'source.png' };
  const request = { name: 'dither_render', arguments: { input: 'source.png', output: 'art/result.png', expected_source_sha256: 'a'.repeat(64), options: { width: 1600 } }, _meta: { example: 'preserve' } };
  assert.equal(validateStudioCall(request, policy), request);
  const extract = { name: 'dither_palette_extract', arguments: { input: 'source.png', count: 8 }, _meta: { example: 'preserve' } };
  assert.equal(validateStudioCall(extract, { ...policy, maskInput: 'mask.png' }), extract);
  for (const args of [{}, { seed: 0 }, { seed: -9007199254740991 }, { seed: 9007199254740991 }]) assert.equal(validateStudioCall({ name: 'dither_random_style', arguments: args }, policy).arguments, args);
  for (const args of [null, [], { seed: null }, { seed: 1.5 }, { seed: '1' }, { seed: 9007199254740992 }, { input: 'source.png' }, { output: 'art.png' }, { seed: 1, options: {} }]) assert.throws(() => validateStudioCall({ name: 'dither_random_style', arguments: args }, policy));
  const recipe = { version: 1, palette: 'mono', options: { width: 64, height: 32 } };
  const recipeCall = { name: 'dither_recipe_save', arguments: { output: 'recipes/art.json', recipe }, _meta: { keep: true } };
  assert.equal(validateStudioCall(recipeCall, { ...policy, acceptedRecipe: recipe }), recipeCall);
  assert.throws(() => validateStudioCall(recipeCall, policy), /successful preview/);
  for (const args of [
    { ...recipeCall.arguments, output: '../art.json' }, { ...recipeCall.arguments, output: 'art.png' },
    { ...recipeCall.arguments, output: 'source.json' }, { ...recipeCall.arguments, output: 'mask.json' },
    { ...recipeCall.arguments, input: 'source.png' }, { ...recipeCall.arguments, recipe: { ...recipe, palette: 'oat-and-ink' } },
    { ...recipeCall.arguments, recipe: { ...recipe, options: { width: 32, height: 32 } } },
  ]) assert.throws(() => validateStudioCall({ name: 'dither_recipe_save', arguments: args }, { input: 'source.json', maskInput: 'mask.json', acceptedRecipe: recipe }));
  for (const call of [
    { name: 'dither_batch', arguments: {} },
    { name: 'dither_palette_extract', arguments: { input: 'other.png', count: 8 } },
    ...[0, 1, 257, 2.5, '8', null, undefined].map(count => ({ name: 'dither_palette_extract', arguments: { input: 'source.png', count } })),
    ...['mask_input', 'options', 'output', 'palette', 'recipe', 'unknown'].map(key => ({ name: 'dither_palette_extract', arguments: { input: 'source.png', count: 8, [key]: 'blocked' } })),
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


test('embedding extracts a draft palette, previews it, and saves the exact preview', { timeout: 30_000 }, async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-palette-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  const session = await createStudioSession({ binary, root, input: 'source.png', initial: { options: { width: 64, algorithm: 'atkinson', seed: 17 } } });
  t.after(() => session.close());
  const state = new StudioState(); state.canCallTools = true; state.setInput(session.bootstrap.arguments); state.hostResult(session.bootstrap.result);
  const extract = state.beginExtraction(5);
  const extracted = await session.callTool({ name: 'dither_palette_extract', arguments: extract.request });
  assert.equal(state.finishExtraction(extract, extracted), true); assert.equal(state.canSave, false);
  assert.equal(state.draft.colors.length, 5); assert.equal(state.draft.palette, undefined); assert.equal(state.draft.options.seed, 17);
  assert.deepEqual(await readdir(root), ['source.png']);
  const preview = state.beginPreview();
  assert.equal(state.finishPreview(preview, await session.callTool({ name: 'dither_studio', arguments: preview.request })), true);
  const save = state.beginSave('extracted.png');
  assert.equal(state.finishSave(save, await session.callTool({ name: 'dither_render', arguments: save.request })), true);
  assert.deepEqual(await readFile(path.join(root, 'extracted.png')), Buffer.from(state.rendered.src.split(',')[1], 'base64'));
});

test('embedding defaults fit tall sources and allow an initial custom palette', { timeout: 30_000 }, async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-defaults-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  const setup = await createStudioSession({ binary, root, input: 'source.png' });
  const tall = await setup.callTool({ name: 'dither_render', arguments: { input: 'source.png', output: 'tall.png', palette: 'mono', options: { width: 8, height: 64 } } });
  assert.ok(!tall.isError); await setup.close();
  for (const initial of [{}, { colors: ['#172A35', '#FAE8D2'] }]) {
    const session = await createStudioSession({ binary, root, input: 'tall.png', initial });
    try {
      const { result, arguments: args } = session.bootstrap;
      assert.equal(args.options.width, undefined); assert.equal(args.options.height, undefined);
      assert.equal(result.structuredContent.width, 8); assert.equal(result.structuredContent.height, 64);
      if (initial.colors) { assert.equal(args.palette, undefined); assert.deepEqual(result.structuredContent.recipe.colors, initial.colors); }
      else assert.equal(args.palette, 'oat-and-ink');
    } finally { await session.close(); }
  }
});

test('embedding recipe authority follows successful previews and survives image saves and recipe retries', { timeout: 30_000 }, async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-recipes-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  const session = await createStudioSession({ binary, root, input: 'source.png', maskInput: 'source.png', initial: { colors: ['#182F37', '#E8DBC0'], options: { width: 64, algorithm: 'atkinson', seed: 17, effects: { noise: .1 } } } });
  t.after(() => session.close());
  const initial = session.bootstrap.result;
  const recipe = initial.structuredContent.recipe;
  assert.equal(recipe.options.mask.shape, 'image');
  const call = output => ({ name: 'dither_recipe_save', arguments: { output, recipe } });
  assert.ok(!(await session.callTool(call('initial.json'))).isError);
  assert.deepEqual(JSON.parse(await readFile(path.join(root, 'initial.json'), 'utf8')), recipe);
  const bytes = await readFile(path.join(root, 'initial.json'));
  assert.equal((await session.callTool(call('initial.json'))).isError, true);
  assert.deepEqual(await readFile(path.join(root, 'initial.json')), bytes);
  assert.ok(!(await session.callTool(call('retry.json'))).isError);
  const image = await session.callTool({ name: 'dither_render', arguments: { ...initial._meta.dither.request, output: 'art.png' } });
  assert.ok(!image.isError); assert.ok(!(await session.callTool(call('after-image.json'))).isError);
  await assert.rejects(session.callTool({ name: 'dither_recipe_save', arguments: { output: 'forged.json', recipe: { ...recipe, options: { ...recipe.options, seed: 18 } } } }), /exact recipe/);
  await session.callTool({ name: 'dither_palette_extract', arguments: { input: 'source.png', count: 5 } });
  await assert.rejects(session.callTool(call('after-extract.json')), /successful preview/);
  const failed = await session.callTool({ name: 'dither_studio', arguments: { ...initial._meta.dither.request, options: { width: 2048 } } });
  assert.equal(failed.isError, true); await assert.rejects(session.callTool(call('after-failure.json')), /successful preview/);
  const controller = new AbortController(); controller.abort();
  await assert.rejects(session.callTool({ name: 'dither_studio', arguments: initial._meta.dither.request }, { signal: controller.signal }));
  await assert.rejects(session.callTool(call('after-cancel.json')), /successful preview/);
  const refreshed = await session.callTool({ name: 'dither_studio', arguments: initial._meta.dither.request });
  assert.ok(!refreshed.isError); assert.ok(!(await session.callTool(call('refreshed.json'))).isError);
  assert.deepEqual((await readdir(root)).sort(), ['after-image.json', 'art.png', 'initial.json', 'refreshed.json', 'retry.json', 'source.png']);
});

test('random host supersession keeps newer authority and pending state despite delayed old completion', { timeout: 30_000 }, async (t) => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'dither-embedding-random-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
  const session = await createStudioSession({ binary, root, input: 'source.png', initial: { options: { width: 64 } } });
  t.after(() => session.close());
  const initial = session.bootstrap.result;
  const original = Client.prototype.callTool;
  const held = new Map(); let holdNext;
  t.mock.method(Client.prototype, 'callTool', async function(params, options) {
    const hold = holdNext === params.name; if (hold) holdNext = undefined;
    const result = await original.call(this, params, options);
    if (hold) await new Promise(resolve => held.set(params.name, { resolve, signal: options.signal }));
    return result;
  });
  const waitHeld = async name => { for (let i = 0; i < 100 && !held.has(name); i++) await new Promise(resolve => setTimeout(resolve, 10)); assert.ok(held.has(name)); };
  holdNext = 'dither_studio';
  const oldPreview = session.callTool({ name: 'dither_studio', arguments: initial._meta.dither.request });
  await waitHeld('dither_studio');
  holdNext = 'dither_random_style';
  const newStyle = session.callTool({ name: 'dither_random_style', arguments: { seed: 0 } });
  await waitHeld('dither_random_style');
  const alreadyAborted = new AbortController(); alreadyAborted.abort();
  await assert.rejects(session.callTool({ name: 'dither_random_style', arguments: {} }, { signal: alreadyAborted.signal }), /aborted/);
  assert.equal(held.get('dither_random_style').signal.aborted, false);
  assert.equal(held.get('dither_studio').signal.aborted, true);
  held.get('dither_studio').resolve(); await oldPreview;
  await assert.rejects(session.callTool({ name: 'dither_studio', arguments: initial._meta.dither.request }), /already running/);
  await assert.rejects(session.callTool({ name: 'dither_recipe_save', arguments: { output: 'old.json', recipe: initial.structuredContent.recipe } }), /successful preview/);
  held.get('dither_random_style').resolve(); const random = await newStyle;
  assert.equal(random.structuredContent.seed, 0);
  const state = new StudioState(); state.canCallTools = true; state.setInput(session.bootstrap.arguments); state.hostResult(initial);
  const operation = state.beginRandom(); const preview = state.finishRandom(operation, random); assert.ok(preview);
  assert.equal(state.finishPreview(preview, await session.callTool({ name: 'dither_studio', arguments: preview.request })), true);
  assert.deepEqual(await readdir(root), ['source.png']);
  const save = state.beginRecipeSave('latest.json');
  assert.equal(state.finishRecipeSave(save, await session.callTool({ name: 'dither_recipe_save', arguments: save.request })), true);
  assert.deepEqual(JSON.parse(await readFile(path.join(root, 'latest.json'), 'utf8')), state.rendered.recipe);
});
