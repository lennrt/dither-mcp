import test from 'node:test';
import assert from 'node:assert/strict';
import { StudioState, clone, decodePreview, parseColors, studioRequest } from './state.mjs';

const request = { input: 'source.png', palette: 'oat-and-ink', mask_input: 'mask.png', options: { algorithm: 'atkinson', width: 320, height: 160, strength: .9, seed: 17, crop: { x: 2, y: 3, width: 90, height: 80 }, mask: { shape: 'image' }, effects: { noise: .1, pixel_sort: true, pixel_sort_threshold: 0 } } };
function result(value = request) {
  const source = clone(value);
  return { content: [{ type: 'image', mimeType: 'image/png', data: 'iVBORw0KGgo=' }], structuredContent: { path: source.input, mask_input: source.mask_input, width: source.options.width, height: source.options.height, mime_type: 'image/png', recipe: { version: 1, palette: source.palette, colors: source.colors, options: clone(source.options) } }, _meta: { dither: { request: source, algorithms: [{ id: 'atkinson', name: 'Atkinson', family: 'diffusion' }], palettes: [{ id: 'oat-and-ink', name: 'Oat & ink', colors: ['#182F37', '#E8DBC0'] }] } } };
}
function ready() { const state = new StudioState(); state.canCallTools = true; state.setInput(request); assert.equal(state.hostResult(result()), true); return state; }

test('initial preview retains the exact recipe, source, mask, and catalogs', () => {
  const state = ready(); assert.deepEqual(state.rendered.request, request); assert.equal(state.canSave, true);
  assert.equal(state.algorithms.length, 1); assert.equal(state.palettes.length, 1); assert.match(state.rendered.src, /^data:image\/png;base64,/);
});
test('editing controls preserves the displayed recipe and disables saving', () => {
  const state = ready(); const changed = clone(state.draft); changed.options.algorithm = 'bayer-4'; state.edit(changed);
  assert.equal(state.canSave, false); assert.deepEqual(state.rendered.request, request); assert.equal(state.dirty, true);
});
test('save uses the exact accepted preview, including hidden crop, mask, effects, and zero values', () => {
  const state = ready(); const operation = state.beginSave(' output/final.png ');
  assert.deepEqual(operation.request, { ...request, output: 'output/final.png' }); assert.equal(state.canSave, false);
  operation.request.options.effects.noise = 99; assert.equal(state.rendered.request.options.effects.noise, .1);
});
test('save requires an explicit output destination and a successful current preview', () => {
  const state = ready(); assert.throws(() => state.beginSave(' '), /output path/);
  state.edit({ ...request, palette: 'mono' }); assert.throws(() => state.beginSave('final.png'), /Apply/);
});
test('preview requests cannot carry output, format, or recipe file fields', () => {
  assert.deepEqual(studioRequest({ ...request, output: 'hidden.png', recipe: 'hidden.json', format: 'svg' }), request);
});
test('old app response cannot replace a newer preview after cancellation', () => {
  const state = ready(); const first = state.beginPreview(); state.cancel();
  const changed = clone(request); changed.options.algorithm = 'bayer-4'; state.edit(changed); const second = state.beginPreview();
  assert.equal(state.finishPreview(first, result()), false); assert.equal(state.pending.id, second.id);
  assert.equal(state.finishPreview(second, result(changed)), true); assert.equal(state.rendered.request.options.algorithm, 'bayer-4');
});
test('editing during rendering rejects that response and keeps the old displayed image', () => {
  const state = ready(); const operation = state.beginPreview(); const changed = clone(request); changed.options.strength = 1.2; state.edit(changed);
  assert.equal(state.finishPreview(operation, result()), false); assert.equal(state.rendered.request.options.strength, .9);
  assert.equal(state.canSave, false); assert.equal(state.busy, false); assert.match(state.status, /changed while/);
});
test('late host result cannot overwrite edits or clear a newer app operation', () => {
  const state = new StudioState(); state.canCallTools = true; state.setInput(request);
  const changed = clone(request); changed.options.algorithm = 'bayer-4'; state.edit(changed); const operation = state.beginPreview();
  assert.equal(state.hostResult(result()), false); assert.equal(state.pending.id, operation.id); assert.equal(state.draft.options.algorithm, 'bayer-4');
  assert.equal(state.finishPreview(operation, result(changed)), true); assert.equal(state.hostResult(result()), false);
});
test('late host result for a different source or same-source older method is rejected', () => {
  const state = new StudioState(); state.canCallTools = true;
  state.setInput({ ...request, input: 'other.png' }); assert.equal(state.hostResult(result()), false);
  state.setInput({ ...request, options: { ...request.options, algorithm: 'bayer-4' } }); assert.equal(state.hostResult(result()), false);
  assert.equal(state.rendered, null);
});
test('failed replacement preview invalidates save even when settings did not change', () => {
  const state = ready(); const operation = state.beginPreview(); assert.equal(state.canSave, false);
  assert.equal(state.finishPreview(operation, { isError: true, content: [{ type: 'text', text: 'Source no longer exists.' }] }), false);
  assert.equal(state.canSave, false); assert.equal(state.rendered.request.input, request.input); assert.match(state.status, /no longer exists/);
});
test('canceled replacement invalidates save and a late success cannot restore it', () => {
  const state = ready(); const operation = state.beginPreview(); state.cancel();
  assert.equal(state.finishPreview(operation, result()), false); assert.equal(state.canSave, false); assert.ok(state.rendered);
});
test('transport errors invalidate a pending preview without changing its displayed recipe', () => {
  const state = ready(); const operation = state.beginPreview(); state.reject(operation, new Error('Disconnected'));
  assert.equal(state.canSave, false); assert.equal(state.busy, false); assert.equal(state.error, true);
});
test('a save error is visible and an explicit retry uses the same accepted preview', () => {
  const state = ready(); const operation = state.beginSave('final.png');
  assert.equal(state.finishSave(operation, { isError: true, content: [{ type: 'text', text: 'Output exists.' }] }), false);
  assert.equal(state.canSave, true); assert.match(state.status, /Output exists/); assert.deepEqual(state.beginSave('new.png').request, { ...request, output: 'new.png' });
});
test('unsafe integer seeds and excessive dimensions cannot be previewed or reconstructed', () => {
  for (const seed of [9007199254740992, -9007199254740992, 1.5]) assert.throws(() => studioRequest({ ...request, options: { ...request.options, seed } }), /Seed/);
  for (const width of [1025, -1, 1.5]) assert.throws(() => studioRequest({ ...request, options: { ...request.options, width } }), /Width/);
  assert.equal(studioRequest({ ...request, options: { ...request.options, seed: 9007199254740991 } }).options.seed, 9007199254740991);
});
test('lowercase and short hex initial palettes preserve their exact strings and save recipe', () => {
  for (const colors of [['#152a39', '#efddc2'], ['#000', '#fff']]) {
    const custom = clone(request); delete custom.palette; custom.colors = colors;
    const state = new StudioState(); state.canCallTools = true; state.setInput(custom);
    assert.equal(state.hostResult(result(custom)), true); assert.deepEqual(state.beginSave('custom.png').request.colors, colors);
  }
});
test('custom palette validation rejects duplicates, invalid colors, and invalid counts', () => {
  assert.deepEqual(parseColors('#fa6, 172a35'), ['#FFAA66', '#172A35']);
  for (const colors of ['#FFF #ffffff', '#fff', '#fff nope', Array(257).fill('#fff')]) assert.throws(() => parseColors(colors));
  assert.throws(() => studioRequest({ ...request, colors: ['#fff', '#000'] }), /built-in palette or custom/);
});
test('malformed image data, mismatched path, mask, dimensions, and recipe never enable save', () => {
  const changes = [(value) => value.content[0].data = 'https://example.com/image.png', (value) => value.structuredContent.path = 'wrong.png', (value) => value.structuredContent.mask_input = 'wrong.png', (value) => value.structuredContent.width = 1025, (value) => value.structuredContent.recipe.options.strength = 1.8];
  for (const change of changes) { const state = new StudioState(); state.canCallTools = true; state.setInput(request); const invalid = result(); change(invalid); assert.equal(state.hostResult(invalid), false); assert.equal(state.canSave, false); }
});
test('omitted dimensions can become exact canonical dimensions but explicit dimensions remain checked', () => {
  const state = new StudioState(); state.canCallTools = true; const initial = clone(request); delete initial.options.width; delete initial.options.height; state.setInput(initial);
  assert.equal(state.hostResult(result()), true); const saved = state.beginSave('exact.png'); assert.deepEqual(saved.request.options, request.options);
  state.finishSave(saved, { content: [], structuredContent: { path: 'exact.png' } });
  const operation = state.beginPreview(); const changed = clone(request); changed.options.width = 321;
  assert.equal(state.finishPreview(operation, result(changed)), false); assert.equal(state.canSave, false);
});
test('host without tools capability can display the preview but cannot call preview or save', () => {
  const state = ready(); state.canCallTools = false;
  assert.equal(state.canPreview, false); assert.equal(state.canSave, false); assert.throws(() => state.beginPreview(), /unavailable/); assert.throws(() => state.beginSave('final.png'), /Apply/);
});
test('invalid new host input invalidates the previous save permission', () => {
  const state = ready(); assert.throws(() => state.setInput({ input: '', options: {} })); assert.equal(state.canSave, false); assert.equal(state.canPreview, false);
});
