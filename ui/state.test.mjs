import test from 'node:test';
import assert from 'node:assert/strict';
import { StudioState, clone, decodePreview, editOption, favoriteName, parseColors, studioRequest } from './state.mjs';

const request = { input: 'source.png', palette: 'oat-and-ink', mask_input: 'mask.png', options: { algorithm: 'atkinson', width: 320, height: 160, strength: .9, seed: 17, crop: { x: 2, y: 3, width: 90, height: 80 }, mask: { shape: 'image' }, effects: { noise: .1, pixel_sort: true, pixel_sort_threshold: 0 } } };
function result(value = request) {
  const source = clone(value);
  return { content: [{ type: 'image', mimeType: 'image/png', data: 'iVBORw0KGgo=' }], structuredContent: { path: source.input, mask_input: source.mask_input, width: source.options.width, height: source.options.height, mime_type: 'image/png', source_sha256: 'a'.repeat(64), mask_sha256: source.mask_input ? 'b'.repeat(64) : undefined, source_width: 90, source_height: 80, export_limits: { max_width: 16384, max_height: 16384, max_pixels: 16777216 }, recipe: { version: 1, palette: source.palette, colors: source.colors, options: clone(source.options) } }, _meta: { dither: { request: source, algorithms: [{ id: 'atkinson', name: 'Atkinson', family: 'diffusion' }], palettes: [{ id: 'oat-and-ink', name: 'Oat & ink', colors: ['#182F37', '#E8DBC0'] }] } } };
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
  assert.deepEqual(operation.request, { ...request, output: 'output/final.png', expected_source_sha256: 'a'.repeat(64), expected_mask_sha256: 'b'.repeat(64) }); assert.equal(state.canSave, false);
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
  assert.equal(state.canSave, true); assert.match(state.status, /Output exists/); assert.deepEqual(state.beginSave('new.png').request, { ...request, output: 'new.png', expected_source_sha256: 'a'.repeat(64), expected_mask_sha256: 'b'.repeat(64) });
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

test('source and custom exports change only dimensions and retain the preview fingerprints', () => {
  for (const [selection, size] of [[{ mode: 'source' }, { width: 90, height: 80 }], [{ mode: 'custom', width: 1920, height: 1080 }, { width: 1920, height: 1080 }]]) {
    const state = ready();
    const original = clone(state.rendered);
    const save = state.beginSave('large.png', selection);
    assert.deepEqual(save.request, { ...request, options: { ...request.options, ...size }, output: 'large.png', expected_source_sha256: 'a'.repeat(64), expected_mask_sha256: 'b'.repeat(64) });
    assert.deepEqual(state.rendered, original);
    assert.equal(state.dirty, false);
  }
});
test('invalid or excessive export dimensions never begin a save', () => {
  const state = ready();
  for (const selection of [{ mode: 'unknown' }, ...[0, -1, 1.5, NaN, Infinity, 16385, undefined, '100'].map(width => ({ mode: 'custom', width, height: 1 })), { mode: 'custom', width: 1 }, { mode: 'custom', width: 4097, height: 4096 }]) {
    assert.throws(() => state.beginSave('invalid.png', selection));
    assert.equal(state.pending, null);
    assert.equal(state.canSave, true);
  }
  state.rendered.sourceWidth = 16385;
  assert.throws(() => state.beginSave('source.png', { mode: 'source' }), /Export must fit/);
  assert.equal(state.beginSave('limit.png', { mode: 'custom', width: 4096, height: 4096 }).request.options.width, 4096);
});
test('source and mask changes invalidate saving until a fresh preview is accepted', () => {
  for (const code of ['source_changed', 'mask_changed']) {
    for (const structured of [true, false]) {
      const state = ready(); const save = state.beginSave('changed.png');
      const failure = { isError: true, content: [{ type: 'text', text: `${code}: File changed.` }] };
      if (structured) failure.structuredContent = { error: { code } };
      assert.equal(state.finishSave(save, failure), false);
      assert.equal(state.canSave, false);
      assert.equal(state.rendered.request.input, request.input);
      assert.match(state.status, /Apply preview again/);
      assert.throws(() => state.beginSave('retry.png'), /Apply/);
      const preview = state.beginPreview(); const fresh = result(); fresh.structuredContent.source_sha256 = 'c'.repeat(64);
      assert.equal(state.finishPreview(preview, fresh), true);
      assert.equal(state.beginSave('fresh.png').request.expected_source_sha256, 'c'.repeat(64));
    }
  }
});
test('missing or malformed fingerprints and export metadata cannot authorize a save', () => {
  for (const [key, value] of [['source_sha256', undefined], ['source_sha256', 'x'.repeat(64)], ['source_sha256', ['a'.repeat(64)]], ['mask_sha256', undefined], ['source_width', 0], ['source_height', 1.5], ['export_limits', undefined], ['export_limits', { max_width: 16384, max_height: 16384, max_pixels: 0 }]]) {
    const invalid = result(); invalid.structuredContent[key] = value;
    assert.throws(() => decodePreview(invalid));
  }
});

test('source colors become a custom draft without altering options, mask, or displayed preview', () => {
  const state = ready(); const displayed = clone(state.rendered);
  const extract = state.beginExtraction();
  assert.deepEqual(extract.request, { input: request.input, count: 8 });
  assert.equal(state.canPreview, false); assert.equal(state.canSave, false);
  assert.throws(() => state.beginExtraction(), /running/);
  const colors = ['#123', '#abcdef'];
  assert.equal(state.finishExtraction(extract, { structuredContent: { colors } }), true);
  assert.deepEqual(state.draft, { input: request.input, mask_input: request.mask_input, options: request.options, colors: ['#112233', '#ABCDEF'] });
  assert.deepEqual(state.rendered, displayed); assert.equal(state.canSave, false);
  const preview = state.beginPreview();
  assert.equal(state.finishPreview(preview, result(state.draft)), true);
  assert.deepEqual(state.beginSave('extracted.png').request.colors, ['#112233', '#ABCDEF']);
});
test('even identical extracted colors need an explicit fresh preview', () => {
  const state = ready(); const custom = clone(request); delete custom.palette; custom.colors = ['#000000', '#FFFFFF'];
  state.setInput(custom); assert.equal(state.hostResult(result(custom)), true);
  const extract = state.beginExtraction(2);
  assert.equal(state.finishExtraction(extract, { structuredContent: { colors: custom.colors } }), true);
  assert.equal(state.dirty, false); assert.equal(state.canSave, false);
});
test('invalid extraction counts never call a tool or invalidate a preview', () => {
  for (const count of [0, 1, 257, 1.5, NaN, Infinity, '8', null]) {
    const state = ready(); assert.throws(() => state.beginExtraction(count), /whole number/);
    assert.equal(state.pending, null); assert.equal(state.canSave, true);
  }
  const state = ready(); state.canCallTools = false; assert.throws(() => state.beginExtraction(), /unavailable/);
});
test('cancelled, edited, or superseded extraction cannot replace current colors or a newer operation', () => {
  const extracted = { structuredContent: { colors: ['#000000', '#FFFFFF'] } };
  for (const change of ['cancel', 'edit', 'input', 'raw-controls']) {
    const state = ready(); const extract = state.beginExtraction(2);
    if (change === 'cancel') state.cancel();
    if (change === 'edit') state.edit({ ...state.draft, palette: 'mono' });
    if (change === 'input') state.setInput({ ...request, input: 'other.png' });
    if (change === 'raw-controls') state.edit(clone(state.draft));
    const draft = clone(state.draft);
    assert.equal(state.finishExtraction(extract, extracted), false);
    assert.deepEqual(state.draft, draft); assert.equal(state.canSave, false);
  }
  const state = ready(); const old = state.beginExtraction(); state.cancel(); const current = state.beginExtraction();
  assert.equal(state.finishExtraction(old, extracted), false); assert.equal(state.pending.id, current.id);
  assert.equal(state.finishExtraction(current, extracted), true);
});
test('monochrome, malformed, duplicate, oversized, and failed extraction preserve the existing settings', () => {
  for (const extraction of [
    {}, { colors: '#000000 #FFFFFF' }, { colors: [] }, { colors: ['#000000'] },
    { colors: ['#000000', 'invalid'] }, { colors: ['#000000', '#000'] },
    { colors: [123, 456] }, { colors: ['#000000', '#FFFFFF', '#FF0000'] },
  ]) {
    const state = ready(); const extract = state.beginExtraction(2);
    assert.equal(state.finishExtraction(extract, { structuredContent: extraction }), false);
    assert.deepEqual(state.draft, request); assert.deepEqual(state.rendered.request, request);
    assert.equal(state.error, true); assert.equal(state.canSave, false); assert.equal(state.canPreview, true);
  }
  for (const transportError of [false, true]) {
    const state = ready(); const extract = state.beginExtraction();
    if (transportError) state.reject(extract, new Error('Disconnected'));
    else state.finishExtraction(extract, { isError: true, content: [{ type: 'text', text: 'Source missing' }] });
    assert.deepEqual(state.draft, request); assert.equal(state.error, true); assert.equal(state.canSave, false);
  }
});

test('a later control edit merges pending advanced JSON and keeps hidden options', () => {
  const previous = { brightness: 0, algorithm: 'atkinson', width: 320 };
  const advanced = { ...previous, brightness: -0.1, algorithm: 'bayer-8', serpentine: true, crop: { x: 2, y: 3, width: 90, height: 80 }, mask: { shape: 'ellipse', invert: true }, effects: { noise: 0.2 } };
  const text = JSON.stringify(advanced);
  for (const [key, value] of [['brightness', 0.35], ['algorithm', 'bayer-4'], ['serpentine', false]]) {
    const edited = editOption(previous, key, value, text);
    assert.deepEqual(edited, { ...advanced, [key]: value });
    edited.crop.x = 100;
    assert.equal(advanced.crop.x, 2);
    assert.deepEqual(previous, { brightness: 0, algorithm: 'atkinson', width: 320 });
  }
  const empty = editOption(previous, 'brightness', 0, '{}');
  assert.deepEqual(empty, { brightness: 0 });
  assert.deepEqual(editOption(advanced, 'serpentine', false), { ...advanced, serpentine: false });
});
test('malformed or nonobject pending options cannot silently replace a control edit', () => {
  for (const text of ['{', '', '[1, 2]', 'null', 'true', '42', '"options"']) {
    const previous = clone(request.options);
    assert.throws(() => editOption(previous, 'brightness', 0.35, text), /Correct the advanced options JSON/);
    assert.deepEqual(previous, request.options);
  }
});

test('omitted options default to an object but explicit null cannot reset the draft', () => {
  assert.deepEqual(studioRequest({ input: 'source.png' }), { input: 'source.png', options: {} });
  for (const options of [null, [], 42, false, 'options']) assert.throws(() => studioRequest({ input: 'source.png', options }), /Options must be a JSON object/);
  const state = ready(); const original = clone(state.draft);
  assert.throws(() => studioRequest({ ...state.draft, options: JSON.parse('null') }), /Options must be a JSON object/);
  assert.deepEqual(state.draft, original);
});

function recipeReceipt(operation) {
  return { structuredContent: { path: operation.request.output, format: 'json', bytes: 128, sha256: 'a'.repeat(64), recipe: clone(operation.request.recipe) } };
}
test('recipe saving preserves the exact accepted custom recipe and excludes source, mask, and output paths', () => {
  const state = ready(); const custom = clone(request); delete custom.palette; custom.colors = ['#182F37', '#E8DBC0'];
  state.edit(custom); const preview = state.beginPreview(); assert.equal(state.finishPreview(preview, result(custom)), true);
  const operation = state.beginRecipeSave(' recipes/extracted.json ', 'art.png');
  assert.deepEqual(operation.request, { output: 'recipes/extracted.json', recipe: state.rendered.recipe });
  assert.deepEqual(Object.keys(operation.request.recipe).sort(), ['colors', 'options', 'version']);
  assert.equal(operation.request.recipe.options.mask.shape, 'image'); assert.equal(operation.request.recipe.options.effects.pixel_sort_threshold, 0);
  assert.equal(state.saving, true); assert.equal(state.canSave, false); assert.equal(state.canPreview, false);
  assert.equal(state.finishRecipeSave(operation, recipeReceipt(operation)), true);
  assert.equal(state.canSave, true); assert.match(state.status, /^Saved recipe recipes\/extracted.json$/);
  operation.request.recipe.options.width = 12; assert.equal(state.rendered.recipe.options.width, 320);
});
test('recipe saving requires current preview, capability, and a distinct relative JSON destination', () => {
  for (const output of ['', ' ', 'recipe.png', '../recipe.json', '/recipe.json', 'file:recipe.json', 'a\\recipe.json', './recipe.json', 'a//recipe.json', 'a/../recipe.json']) {
    assert.throws(() => ready().beginRecipeSave(output));
  }
  const state = ready(); assert.throws(() => state.beginRecipeSave('export.json', 'export.json'), /different/);
  for (const key of ['input', 'mask_input']) {
    const protectedState = ready(); protectedState.rendered.request[key] = 'protected.json'; protectedState.draft[key] = 'protected.json'; assert.throws(() => protectedState.beginRecipeSave('protected.json'), /different/);
  }
  const dirty = ready(); dirty.edit({ ...request, palette: 'mono' }); assert.throws(() => dirty.beginRecipeSave('recipe.json'), /Apply/);
  const denied = ready(); denied.canCallTools = false; assert.throws(() => denied.beginRecipeSave('recipe.json'), /Apply/);
  const extracting = ready(); extracting.beginExtraction(); assert.throws(() => extracting.beginRecipeSave('recipe.json'), /Apply/);
  extracting.cancel(); assert.throws(() => extracting.beginRecipeSave('recipe.json'), /Apply/);
});
test('recipe receipt rejects missing, malformed, mismatched and error results without losing retry settings', () => {
  const mutations = [value => delete value.structuredContent, value => value.structuredContent.path = 'other.json', value => value.structuredContent.format = 'png', value => value.structuredContent.bytes = 0, value => value.structuredContent.bytes = 1.5, value => value.structuredContent.sha256 = 'wrong', value => delete value.structuredContent.recipe, value => value.structuredContent.recipe.options.seed = 18];
  for (const mutate of mutations) {
    const state = ready(); const operation = state.beginRecipeSave('recipe.json'); const receipt = recipeReceipt(operation); mutate(receipt);
    assert.equal(state.finishRecipeSave(operation, receipt), false); assert.equal(state.error, true); assert.doesNotMatch(state.status, /^Saved/);
    assert.equal(state.canSave, true); assert.deepEqual(state.draft, request); assert.equal(state.beginRecipeSave('retry.json').request.recipe.options.seed, 17);
  }
  const state = ready(); const operation = state.beginRecipeSave('recipe.json');
  assert.equal(state.finishRecipeSave(operation, { isError: true, content: [{ type: 'text', text: 'Output already exists.' }] }), false);
  assert.match(state.status, /already exists/); assert.equal(state.canSave, true);
});
test('stale recipe completions cannot announce success or clear a newer operation', () => {
  const state = ready(); const operation = state.beginRecipeSave('recipe.json');
  state.setInput(request); assert.equal(state.hostResult(result()), true); const newer = state.beginPreview();
  assert.equal(state.finishRecipeSave(operation, recipeReceipt(operation)), false); assert.equal(state.pending.id, newer.id);
  const edited = ready(); const saving = edited.beginRecipeSave('recipe.json'); edited.edit({ ...request, palette: 'mono' });
  assert.equal(edited.finishRecipeSave(saving, recipeReceipt(saving)), false); assert.equal(edited.canSave, false); assert.match(edited.status, /changed while/);
});

const randomResult = (seed = 42) => ({ structuredContent: { seed, recipe: { version: 1, palette: 'oat-and-ink', options: { algorithm: 'atkinson', seed, strength: .8, effects: { scanlines: .1 } } } } });
test('random style previews complete appearance replacement while retaining geometry and mask', () => {
  const state = ready(); const base = clone(state.draft); base.options.pixel_scale = 3; base.options.resize_filter = 'nearest';
  const random = state.beginRandom(base); assert.equal(state.canSave, false); assert.equal(state.canRandom, true); assert.equal(state.pending.kind, 'random');
  const preview = state.finishRandom(random, randomResult());
  assert.equal(preview.kind, 'preview'); assert.equal(state.canRandom, true); assert.deepEqual(state.draft, request);
  assert.equal(preview.request.input, request.input); assert.equal(preview.request.mask_input, request.mask_input);
  assert.deepEqual(preview.request.options, { algorithm: 'atkinson', seed: 42, strength: .8, effects: { scanlines: .1 }, width: 320, height: 160, crop: request.options.crop, mask: request.options.mask, pixel_scale: 3, resize_filter: 'nearest' });
  assert.equal(state.finishPreview(preview, result(preview.request)), true); assert.equal(state.canSave, true);
  assert.deepEqual(state.beginRecipeSave('random.json').request.recipe, clone(result(preview.request).structuredContent.recipe));
  assert.equal(state.canRandom, false);
});
test('repeated random clicks supersede either stage and ignore late results and failures', () => {
  const state = ready(); const first = state.beginRandom(); const second = state.beginRandom();
  assert.equal(state.finishRandom(first, randomResult(1)), null); assert.equal(state.pending.id, second.id);
  assert.equal(state.reject(first, new Error('late error')), false);
  const oldPreview = state.finishRandom(second, randomResult(2)); const third = state.beginRandom();
  assert.equal(state.finishPreview(oldPreview, result(oldPreview.request)), false); assert.equal(state.pending.id, third.id);
  const latestPreview = state.finishRandom(third, randomResult(3));
  assert.equal(state.finishPreview(latestPreview, result(latestPreview.request)), true); assert.equal(state.draft.options.seed, 3);
});
test('edits, source replacement and cancellation supersede random generation without auto preview', () => {
  for (const change of [state => state.edit({ ...request, palette: 'mono' }), state => state.setInput({ ...request, input: 'other.png' }), state => state.cancel()]) {
    const state = ready(); const operation = state.beginRandom(); change(state);
    const draft = clone(state.draft); assert.equal(state.finishRandom(operation, randomResult()), null); assert.deepEqual(state.draft, draft); assert.equal(state.canSave, false);
  }
});
test('malformed or failed random styles preserve the draft and cannot authorize saving', () => {
  const mutations = [value => delete value.structuredContent, value => value.structuredContent.seed = 1.5, value => value.structuredContent.seed = 9007199254740992, value => value.structuredContent.recipe.options.seed = 2, value => value.structuredContent.recipe.version = 2, value => value.structuredContent.recipe.palette = 'unknown', value => value.structuredContent.recipe.options.algorithm = 'unknown', value => value.structuredContent.recipe.options = null, value => value.structuredContent.recipe.input = 'wrong.png', ...['width', 'height', 'pixel_scale', 'crop', 'mask', 'resize_filter'].map(key => value => value.structuredContent.recipe.options[key] = 1)];
  for (const mutate of mutations) {
    const state = ready(); const random = state.beginRandom(); const value = randomResult(); mutate(value);
    assert.equal(state.finishRandom(random, value), null); assert.deepEqual(state.draft, request); assert.equal(state.canSave, false); assert.equal(state.error, true);
  }
  const state = ready(); const operation = state.beginRandom();
  assert.equal(state.finishRandom(operation, { isError: true, content: [{ type: 'text', text: 'Random generation failed.' }] }), null);
  assert.match(state.status, /generation failed/); assert.deepEqual(state.draft, request);
  const noTools = ready(); noTools.canCallTools = false; assert.throws(() => noTools.beginRandom(), /unavailable/);
  const saving = ready(); saving.beginSave('art.png'); assert.throws(() => saving.beginRandom(), /saving/);
});

test('zero-seeded random style allows canonical omitted engine seed', () => {
  const state = ready(); const random = state.beginRandom(); const value = randomResult(0); delete value.structuredContent.recipe.options.seed;
  const preview = state.finishRandom(random, value); assert.equal(preview.kind, 'preview'); assert.equal(preview.request.options.seed, undefined);
});

function acceptSeed(state, seed) {
  const draft = clone(state.draft); draft.options.seed = seed; state.edit(draft);
  const operation = state.beginPreview(); assert.equal(state.finishPreview(operation, result(draft)), true);
  return state.historyActive;
}
function restoreLook(state, target) {
  const operation = state.beginHistory(target);
  assert.equal(state.finishHistory(operation, result(operation.request)), true);
  return state.historyActive;
}
test('history remembers canonical initial, manual, and random requests without previews or receipts', () => {
  const state = ready(); const initial = state.historyActive;
  assert.deepEqual(state.history, [{ id: initial, request }]); assert.equal(state.canUndo, false); assert.equal(state.canRedo, false);
  const manual = acceptSeed(state, 11); assert.notEqual(manual, initial);
  const random = state.beginRandom(); const preview = state.finishRandom(random, randomResult(22));
  assert.equal(state.finishPreview(preview, result(preview.request)), true);
  assert.equal(state.history.length, 3); assert.deepEqual(state.undoIDs, [initial, manual]);
  for (const entry of state.history) assert.deepEqual(Object.keys(entry).sort(), ['id', 'request']);
  const requestCopy = clone(state.history.at(-1).request);
  state.draft.options.seed = 99; assert.deepEqual(state.history.at(-1).request, requestCopy);
  state.rendered.request.options.seed = 101; assert.deepEqual(state.history.at(-1).request, requestCopy);
});
test('history undo, redo and picks commit only on successful exact fresh previews', () => {
  const state = ready(); const first = state.historyActive; const second = acceptSeed(state, 1); const third = acceptSeed(state, 2);
  const operation = state.beginHistory('undo');
  assert.equal(operation.targetId, second); assert.equal(state.historyActive, third); assert.deepEqual(state.undoIDs, [first, second]); assert.deepEqual(state.redoIDs, []);
  assert.equal(state.canHistory, false); assert.equal(state.canRandom, true); assert.equal(state.canSave, false);
  assert.equal(state.finishHistory(operation, result(operation.request)), true);
  assert.equal(state.historyActive, second); assert.deepEqual(state.undoIDs, [first]); assert.deepEqual(state.redoIDs, [third]);
  const repeat = state.beginPreview(); assert.equal(state.finishPreview(repeat, result(state.draft)), true);
  assert.deepEqual(state.redoIDs, [third]); assert.equal(state.history.length, 3);
  assert.equal(restoreLook(state, 'redo'), third); assert.deepEqual(state.undoIDs, [first, second]);
  assert.equal(restoreLook(state, first), first); assert.deepEqual(state.undoIDs, [first, second, third]);
  const fourth = acceptSeed(state, 3); assert.equal(state.redoIDs.length, 0);
  assert.deepEqual(new Set(state.history.map(entry => entry.id)), new Set([first, second, third, fourth]));
  assert.equal(restoreLook(state, 'undo'), first); assert.equal(restoreLook(state, 'redo'), fourth);
});
test('restoring current look recovers a dirty draft without loops or clearing redo', () => {
  const state = ready(); const first = state.historyActive; const second = acceptSeed(state, 1);
  restoreLook(state, 'undo'); const undo = clone(state.undoIDs), redo = clone(state.redoIDs);
  state.edit({ ...request, palette: 'mono' });
  assert.equal(restoreLook(state, first), first); assert.deepEqual(state.draft, request);
  assert.deepEqual(state.undoIDs, undo); assert.deepEqual(state.redoIDs, redo); assert.equal(state.history.length, 2);
  assert.equal(restoreLook(state, 'redo'), second);
});
test('failed, canceled, edited and superseded restoration cannot change history or draft', () => {
  for (const mode of ['error', 'changed', 'extra', 'cancel', 'edit', 'random']) {
    const state = ready(); const first = state.historyActive; acceptSeed(state, 1);
    const before = clone({ history: state.history, active: state.historyActive, undo: state.undoIDs, redo: state.redoIDs });
    const operation = state.beginHistory(first); const response = result(operation.request);
    if (mode === 'changed') { response._meta.dither.request.options.seed = 99; response.structuredContent.recipe.options.seed = 99; }
    if (mode === 'extra') { response._meta.dither.request.options.brightness = .3; response.structuredContent.recipe.options.brightness = .3; }
    if (mode === 'cancel') state.cancel();
    if (mode === 'edit') state.edit({ ...state.draft, palette: 'mono' });
    let replacement; if (mode === 'random') replacement = state.beginRandom();
    const draft = clone(state.draft);
    assert.equal(state.finishHistory(operation, mode === 'error' ? { isError: true, content: [{ type: 'text', text: 'Source unavailable.' }] } : response), false);
    assert.deepEqual({ history: state.history, active: state.historyActive, undo: state.undoIDs, redo: state.redoIDs }, before);
    assert.deepEqual(state.draft, draft); assert.equal(state.canSave, false);
    if (replacement) assert.equal(state.pending.id, replacement.id);
  }
});
test('history caps snapshots and navigation, touches restored old entries, and prunes no-op navigation', () => {
  const state = ready(); const oldest = state.historyActive;
  for (let seed = 1; seed < 20; seed++) acceptSeed(state, 1000 + seed);
  assert.equal(state.history.length, 20); restoreLook(state, oldest);
  acceptSeed(state, 20); assert.equal(state.history.length, 20);
  assert.ok(state.history.some(entry => entry.id === oldest)); assert.equal(restoreLook(state, 'undo'), oldest);
  for (let i = 0; i < 80; i++) {
    if (i % 3 === 0) restoreLook(state, state.history[0].id);
    else acceptSeed(state, 100 + i);
    assert.ok(state.history.length <= 20 && state.undoIDs.length <= 20 && state.redoIDs.length <= 20);
    assert.notEqual(state.undoIDs.at(-1), state.historyActive); assert.notEqual(state.redoIDs.at(-1), state.historyActive);
    for (const id of [...state.undoIDs, ...state.redoIDs]) assert.ok(state.history.some(entry => entry.id === id));
  }
});
test('new host input resets history and stale restoration cannot cross sources', () => {
  const state = ready(); acceptSeed(state, 1); const old = state.beginHistory('undo');
  const other = { ...request, input: 'other.png' }; state.setInput(other);
  assert.deepEqual(state.history, []); assert.deepEqual(state.undoIDs, []); assert.deepEqual(state.redoIDs, []); assert.equal(state.historyActive, null);
  assert.equal(state.finishHistory(old, result(old.request)), false); assert.equal(state.canHistory, false);
  assert.equal(state.hostResult(result(other)), true); assert.equal(state.history.length, 1); assert.equal(state.history[0].request.input, 'other.png');
  state.canCallTools = false; assert.throws(() => state.beginHistory(state.historyActive), /unavailable/);
});

const favorite = (recipe = clone(result().structuredContent.recipe), name = 'Saved look', id = 'a'.repeat(64)) => ({ id, name, recipe });
function withFavorites(state = ready(), items = [favorite()]) {
  const operation = state.beginFavoritesList();
  assert.equal(state.finishFavoritesList(operation, { structuredContent: { favorites: items.map(({ id, name }) => ({ id, name })), unavailable: 0 } }), true);
  return state;
}
test('favorite names match Go Unicode whitespace, limits, and case-sensitive display', () => {
  assert.equal(favoriteName(' \u0085Look\u2003'), 'Look'); assert.equal(favoriteName('\uFEFFLook'), '\uFEFFLook');
  assert.equal(favoriteName('Look'), 'Look'); assert.equal(favoriteName('\uFFFD'), '\uFFFD'); assert.equal(favoriteName('look'), 'look');
  assert.equal(favoriteName('💫'.repeat(80)), '💫'.repeat(80));
  for (const name of ['', '  ', '💫'.repeat(81), 'a\nb', 'a\u0085b', '\ud800', '\udc00']) assert.throws(() => favoriteName(name));
});
test('favorites metadata refresh, failure, and cancellation preserve preview authorization', () => {
  const state = ready(); const operation = state.beginFavoritesList(); assert.equal(state.canSave, false); assert.equal(state.validPreview, true);
  assert.equal(state.finishFavoritesList(operation, { structuredContent: { favorites: [{ id: 'a'.repeat(64), name: '\uFEFFLook' }], unavailable: 2 } }), true);
  assert.equal(state.canSave, true); assert.equal(state.favoritesUnavailable, 2); assert.equal(state.favorites[0].name, '\uFEFFLook');
  const refresh = state.beginFavoritesList(); state.cancel('Favorite refresh cancelled.');
  assert.equal(state.canSave, true); assert.equal(state.finishFavoritesList(refresh, { structuredContent: { favorites: [], unavailable: 0 } }), false);
  const failing = state.beginFavoritesList(); assert.equal(state.finishFavoritesList(failing, { isError: true, content: [{ type: 'text', text: 'Unable to read favorites.' }] }), false);
  assert.equal(state.canSave, true); assert.equal(state.favorites.length, 1); assert.match(state.status, /Unable/);
  state.beginFavoritesList(); const random = state.beginRandom(); assert.equal(state.pending.id, random.id); assert.equal(state.canSave, false);
});
test('malformed favorites metadata cannot replace the current list', () => {
  const state = withFavorites();
  for (const data of [{}, { favorites: [], unavailable: -1 }, { favorites: [{ id: 'wrong', name: 'Name' }], unavailable: 0 }, { favorites: [favorite(), favorite()], unavailable: 0 }, { favorites: Array(513).fill(favorite()), unavailable: 0 }]) {
    const operation = state.beginFavoritesList(); assert.equal(state.finishFavoritesList(operation, { structuredContent: data }), false); assert.equal(state.favorites.length, 1); assert.equal(state.canSave, true);
  }
});
test('saving a favorite requires the current accepted recipe and verifies its receipt', () => {
  const state = ready(); const operation = state.beginFavoriteSave('  Warm print  ');
  assert.deepEqual(operation.request, { name: 'Warm print', recipe: state.rendered.recipe }); assert.equal(state.saving, true); assert.equal(state.canRandom, false);
  assert.equal(state.finishFavoriteSave(operation, { structuredContent: favorite(operation.request.recipe, 'Warm print') }), true);
  assert.equal(state.favorites[0].name, 'Warm print'); assert.equal(state.canSave, true); assert.match(state.status, /^Saved favorite/);
  for (const mutate of [item => item.id = 'bad', item => item.name = 'Other', item => item.recipe.options.seed = 99]) {
    const saving = state.beginFavoriteSave('Warm print'); const receipt = favorite(clone(saving.request.recipe), 'Warm print'); mutate(receipt);
    assert.equal(state.finishFavoriteSave(saving, { structuredContent: receipt }), false); assert.equal(state.canSave, true); assert.equal(state.favorites.length, 1);
  }
  const collision = state.beginFavoriteSave('Warm print'); assert.equal(state.finishFavoriteSave(collision, { isError: true, content: [{ type: 'text', text: 'Favorite already exists.' }] }), false);
  assert.equal(state.canSave, true); assert.match(state.status, /already exists/);
  state.edit({ ...request, palette: 'mono' }); assert.throws(() => state.beginFavoriteSave('Dirty'), /Apply/);
});
test('favorite load applies its complete recipe to current source and mask only after a fresh exact preview', () => {
  const state = ready(); const current = { ...request, input: 'new.png', mask_input: 'new-mask.png' };
  state.setInput(current); assert.equal(state.hostResult(result(current)), true); withFavorites(state);
  state.edit({ ...state.draft, palette: 'mono' }); const original = clone(state.draft);
  const loading = state.beginFavoriteLoad(favorite().id); assert.equal(state.canSave, false);
  const preview = state.finishFavoriteLoad(loading, { structuredContent: favorite() });
  assert.equal(preview.request.input, 'new.png'); assert.equal(preview.request.mask_input, 'new-mask.png');
  assert.deepEqual(preview.request.options, request.options); assert.deepEqual(state.draft, original);
  assert.equal(state.finishFavoritePreview(preview, result(preview.request)), true); assert.equal(state.canSave, true);
  assert.equal(state.draft.palette, request.palette); assert.equal(state.history.at(-1).request.input, 'new.png');
});
test('favorite defaults admit only canonical mono, algorithm, effects, and inferred dimensions', () => {
  for (const options of [{}, { width: 45 }, { height: 40 }]) {
    const plain = clone(request); delete plain.mask_input; delete plain.options.mask;
    const state = ready(); state.setInput(plain); assert.equal(state.hostResult(result(plain)), true);
    const entry = favorite({ version: 1, options }); withFavorites(state, [entry]);
    const load = state.beginFavoriteLoad(entry.id); const preview = state.finishFavoriteLoad(load, { structuredContent: entry });
    const canonical = { input: plain.input, palette: 'mono', options: { algorithm: 'floyd-steinberg', effects: {}, width: options.width || (options.height ? 45 : 90), height: options.height || (options.width ? 40 : 80) } };
    assert.equal(state.finishFavoritePreview(preview, result(canonical)), true);
    const again = state.beginFavoriteLoad(entry.id); const pending = state.finishFavoriteLoad(again, { structuredContent: entry });
    canonical.options.brightness = .2;
    assert.equal(state.finishFavoritePreview(pending, result(canonical)), false); assert.match(state.status, /different settings/);
  }
});
test('favorite mask mismatches, large dimensions and malformed responses retain existing settings', () => {
  for (const mode of ['no-mask', 'procedural', 'large', 'wrong-id', 'path']) {
    const state = withFavorites(); const before = clone(state.draft); const entry = favorite();
    if (mode === 'no-mask') delete entry.recipe.options.mask;
    if (mode === 'procedural') entry.recipe.options.mask = { shape: 'rectangle', width: 4, height: 4 };
    if (mode === 'large') entry.recipe.options.width = 2048;
    if (mode === 'wrong-id') entry.id = 'b'.repeat(64);
    if (mode === 'path') entry.recipe.input = 'elsewhere.png';
    const operation = state.beginFavoriteLoad(favorite().id); assert.equal(state.finishFavoriteLoad(operation, { structuredContent: entry }), null);
    assert.deepEqual(state.draft, before); assert.equal(state.canSave, false);
  }
  const plain = clone(request); delete plain.mask_input; delete plain.options.mask;
  const state = ready(); state.setInput(plain); assert.equal(state.hostResult(result(plain)), true); withFavorites(state);
  const operation = state.beginFavoriteLoad(favorite().id); assert.equal(state.finishFavoriteLoad(operation, { structuredContent: favorite() }), null); assert.match(state.status, /needs an image mask/);
});
test('edits, cancellation and newer operations discard both favorite loading stages and stale saves', () => {
  const state = withFavorites(); const first = state.beginFavoriteLoad(favorite().id); state.cancel();
  assert.equal(state.finishFavoriteLoad(first, { structuredContent: favorite() }), null);
  const second = state.beginFavoriteLoad(favorite().id); const preview = state.finishFavoriteLoad(second, { structuredContent: favorite() });
  state.edit({ ...state.draft, palette: 'mono' }); assert.equal(state.finishFavoritePreview(preview, result(preview.request)), false); assert.equal(state.draft.palette, 'mono');
  const third = state.beginFavoriteLoad(favorite().id); const random = state.beginRandom();
  assert.equal(state.finishFavoriteLoad(third, { structuredContent: favorite() }), null); assert.equal(state.pending.id, random.id);
  const saved = withFavorites(); const save = saved.beginFavoriteSave('Old'); saved.setInput(request);
  assert.equal(saved.finishFavoriteSave(save, { structuredContent: favorite(save.request.recipe, 'Old') }), false); assert.deepEqual(saved.favorites, []);
});
