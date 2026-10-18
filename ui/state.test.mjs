import test from 'node:test';
import assert from 'node:assert/strict';
import { StudioState, clone, decodePreview, editOption, parseColors, studioRequest } from './state.mjs';

const request = { input: 'source.png', palette: 'oat-and-ink', mask_input: 'mask.png', options: { algorithm: 'atkinson', width: 320, height: 160, strength: .9, seed: 17, crop: { x: 2, y: 3, width: 90, height: 80 }, mask: { shape: 'image' }, effects: { noise: .1, pixel_sort: true, pixel_sort_threshold: 0 } } };
function result(value = request) {
  const source = clone(value);
  return { content: [{ type: 'image', mimeType: 'image/png', data: 'iVBORw0KGgo=' }], structuredContent: { path: source.input, mask_input: source.mask_input, width: source.options.width, height: source.options.height, mime_type: 'image/png', source_sha256: 'a'.repeat(64), mask_sha256: 'b'.repeat(64), source_width: 90, source_height: 80, export_limits: { max_width: 16384, max_height: 16384, max_pixels: 16777216 }, recipe: { version: 1, palette: source.palette, colors: source.colors, options: clone(source.options) } }, _meta: { dither: { request: source, algorithms: [{ id: 'atkinson', name: 'Atkinson', family: 'diffusion' }], palettes: [{ id: 'oat-and-ink', name: 'Oat & ink', colors: ['#182F37', '#E8DBC0'] }] } } };
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
