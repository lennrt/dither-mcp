#!/usr/bin/env node
// Each CLI call opens a new service: the favorite survives between processes.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const [input, name, output] = process.argv.slice(2);
if (!input || !name || !output) {
  console.error('Usage: node examples/favorites.mjs <relative image> <new favorite name> <new PNG path>');
  process.exit(1);
}
const binary = fileURLToPath(new URL(`../bin/dither-mcp${process.platform === 'win32' ? '.exe' : ''}`, import.meta.url));
function call(tool, args) {
  const result = spawnSync(binary, ['call', '--root', process.cwd(), tool, JSON.stringify(args)], {
    encoding: 'utf8', maxBuffer: 4 * 1024 * 1024,
  });
  if (result.error || result.status !== 0) {
    throw new Error(`${tool}: ${result.error?.message ?? result.stderr.trim()}`);
  }
  return JSON.parse(result.stdout);
}
function preview(recipe) {
  return call('dither_studio', { input, palette: recipe.palette, colors: recipe.colors, options: recipe.options });
}
try {
  const generated = call('dither_random_style', { seed: 42 });
  const accepted = preview(generated.recipe);
  const saved = call('dither_favorite_save', { name, recipe: accepted.recipe });
  const library = call('dither_favorites_list', {});
  assert(library.favorites.some(item => item.id === saved.id && item.name === saved.name));
  const loaded = call('dither_favorite_load', { id: saved.id });
  assert.deepEqual(loaded.recipe, accepted.recipe);
  const restored = preview(loaded.recipe);
  assert.equal(restored.data, accepted.data, 'The source changed or the restored look differs.');
  const rendered = call('dither_render', {
    input, output, format: 'png', palette: restored.recipe.palette, colors: restored.recipe.colors,
    options: restored.recipe.options, expected_source_sha256: restored.source_sha256,
  });
  const expectedHash = createHash('sha256').update(Buffer.from(restored.data, 'base64')).digest('hex');
  assert.equal(rendered.sha256, expectedHash, 'Saved PNG differs from the restored preview.');
  console.log(JSON.stringify({ favorite: { id: saved.id, name: saved.name }, image: rendered, exact_replay: true }, null, 2));
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
