import { App, PostMessageTransport, applyDocumentTheme, applyHostStyleVariables } from '@modelcontextprotocol/ext-apps';
import { StudioState, clone, editOption, exportDimensions, parseColors, studioRequest } from './state.mjs';

const app = new App({ name: 'Dither studio', version: '0.1.0' }, {}, { autoResize: true });
const state = new StudioState();
const $ = (id) => document.getElementById(id);
let abortController;
let zoom = 'fit';
let jsonDirty = false;
let customDirty = false;
let lastCatalog = '';
let historyChoice = '';
let lastHistoryActive = null;
let favoritesAutoPending = true;
let favoriteChoice = '';

const ranges = [
  ['strength', 'Dither', 0, 2, .05, 1], ['brightness', 'Brightness', -1, 1, .05, 0],
  ['contrast', 'Contrast', 0, 4, .05, 1], ['saturation', 'Saturation', 0, 4, .05, 1],
  ['gamma', 'Gamma', .1, 8, .05, 1], ['threshold', 'Threshold', 0, 1, .05, .5],
];
const defaults = { algorithm: 'floyd-steinberg', pixel_scale: 1, seed: 0, resize_filter: 'bilinear', color_space: 'srgb', width: 0, height: 0, ...Object.fromEntries(ranges.map(([key, , , , , value]) => [key, value])) };
for (const [key, name, min, max, step, value] of ranges) {
  const row = document.createElement('div');
  row.className = 'range-row';
  const label = document.createElement('label'); label.htmlFor = key; label.textContent = name;
  const input = document.createElement('input'); Object.assign(input, { id: key, type: 'range', min, max, step, value }); input.dataset.option = key;
  const output = document.createElement('output'); output.htmlFor = key; output.id = `${key}-value`; output.textContent = String(value);
  row.append(label, input, output); $('tone-controls').append(row);
}

function paletteMode() { return state.draft?.colors ? 'custom' : 'builtin'; }
function renderCatalog() {
  const version = JSON.stringify([state.algorithms, state.palettes]);
  if (version !== lastCatalog) {
    lastCatalog = version;
    if (state.algorithms.length) {
      const groups = new Map();
      for (const algorithm of state.algorithms) {
        const family = algorithm.family || algorithm.category || 'Methods';
        if (!groups.has(family)) { const group = document.createElement('optgroup'); group.label = family; groups.set(family, group); }
        const option = document.createElement('option'); option.value = algorithm.id; option.textContent = algorithm.name; groups.get(family).append(option);
      }
      $('algorithm').replaceChildren(...groups.values());
    }
    $('palette-count').textContent = state.palettes.length || '';
    $('catalog-count').textContent = state.algorithms.length ? `${state.algorithms.length} methods · ${state.palettes.length} palettes` : 'A quiet place for pixels.';
  }
  renderPalettes();
}
function renderPalettes() {
  const selected = state.draft?.palette || 'mono';
  const terms = $('palette-search').value.trim().toLowerCase().split(/\s+/).filter(Boolean);
  const visible = state.palettes.filter((palette) => terms.every((term) => JSON.stringify(palette).toLowerCase().includes(term)));
  const display = [...visible];
  const current = state.palettes.find((palette) => palette.id === selected);
  if (current && !display.some((palette) => palette.id === selected)) display.unshift(current);
  if (!display.length && !state.palettes.length) display.push({ id: selected, name: selected });
  const groups = new Map();
  for (const palette of display) {
    const category = palette.category || 'Palettes';
    if (!groups.has(category)) { const group = document.createElement('optgroup'); group.label = category; groups.set(category, group); }
    const option = document.createElement('option'); option.value = palette.id; option.textContent = `${palette.name}${palette.colors ? ` · ${palette.colors.length}` : ''}`; groups.get(category).append(option);
  }
  $('palette').replaceChildren(...groups.values()); $('palette').value = selected;
  $('palette-matches').textContent = terms.length ? `${visible.length} matching palettes${current && !visible.includes(current) ? ' · current selection kept' : ''}` : '';
}
function renderSwatches() {
  const palette = state.palettes.find((palette) => palette.id === state.draft?.palette);
  let colors = palette?.colors ?? [];
  if (paletteMode() === 'custom') { try { colors = parseColors($('custom-colors').value); } catch { colors = []; } }
  const swatches = colors.slice(0, 48).filter((color) => /^#[0-9a-f]{6}$/i.test(color)).map((color) => {
    const swatch = document.createElement('span'); swatch.className = 'swatch'; swatch.style.backgroundColor = color; swatch.title = color; swatch.setAttribute('aria-label', color); return swatch;
  });
  if (colors.length > 48) { const more = document.createElement('span'); more.className = 'swatch-more'; more.textContent = `+${colors.length - 48}`; swatches.push(more); }
  $('swatches').replaceChildren(...swatches);
  $('palette-info').textContent = paletteMode() === 'custom' ? `${colors.length || '2–256'} custom colors` : palette?.description ?? '';
}
function syncControls(preserveInput) {
  const options = state.draft?.options ?? {};
  document.querySelectorAll('[data-option]').forEach((element) => {
    // Number inputs retain intermediate typing such as '-' outside .value.
    if (element === preserveInput) return;
    const value = options[element.dataset.option] ?? defaults[element.dataset.option];
    if (element.type === 'checkbox') element.checked = !!value;
    else element.value = value ?? '';
    if (element.type === 'range') $(`${element.id}-value`).textContent = String(Number(value));
  });
  if (!jsonDirty) $('options-json').value = JSON.stringify(options, null, 2);
  if (!customDirty) $('custom-colors').value = (state.draft?.colors ?? []).join(' ');
  $('palette').value = state.draft?.palette || 'mono';
  const custom = paletteMode() === 'custom';
  $('library-controls').hidden = custom; $('custom-controls').hidden = !custom;
  document.querySelectorAll('[data-mode]').forEach((button) => { const selected = button.dataset.mode === paletteMode(); button.classList.toggle('selected', selected); button.setAttribute('aria-pressed', String(selected)); });
  $('algorithm-info').textContent = state.algorithms.find((algorithm) => algorithm.id === options.algorithm)?.description || 'Choose the texture that suits your image.';
  renderSwatches();
}
function renderHistory() {
  if (state.historyActive !== lastHistoryActive || !state.history.some(entry => String(entry.id) === historyChoice)) historyChoice = state.historyActive === null ? '' : String(state.historyActive);
  lastHistoryActive = state.historyActive;
  const entries = [...state.history].reverse().map(entry => {
    const request = entry.request;
    const algorithm = state.algorithms.find(item => item.id === request.options.algorithm)?.name || request.options.algorithm;
    const palette = request.colors ? `${request.colors.length} custom colors` : state.palettes.find(item => item.id === request.palette)?.name || request.palette;
    const option = document.createElement('option'); option.value = String(entry.id);
    option.textContent = `Look ${entry.id} · ${algorithm} · ${palette} · seed ${request.options.seed ?? 0}`;
    return option;
  });
  if (!entries.length) { const empty = document.createElement('option'); empty.value = ''; empty.textContent = 'Your accepted previews appear here'; entries.push(empty); }
  $('history-look').replaceChildren(...entries); $('history-look').value = historyChoice;
  $('history-count').textContent = `${state.history.length} / 20`;
  $('history-look').disabled = !state.canHistory;
  $('history-restore').disabled = !state.canHistory || !historyChoice;
  $('history-undo').disabled = !state.canUndo;
  $('history-redo').disabled = !state.canRedo;
}
function renderFavorites() {
  if (!state.favorites.some(item => item.id === favoriteChoice)) favoriteChoice = state.favorites[0]?.id || '';
  const options = state.favorites.map(item => { const option = document.createElement('option'); option.value = item.id; option.textContent = item.name; return option; });
  if (!options.length) { const empty = document.createElement('option'); empty.value = ''; empty.textContent = 'No saved favorites'; options.push(empty); }
  $('favorite-list').replaceChildren(...options); $('favorite-list').value = favoriteChoice;
  $('favorite-list').disabled = !state.canFavorites || !state.favorites.length;
  $('favorite-refresh').disabled = !state.canFavorites;
  $('favorite-load').disabled = !state.canFavorites || !favoriteChoice;
  $('favorite-name').disabled = state.saving;
  $('favorite-save').disabled = !state.canSave || jsonDirty || customDirty || !$('favorite-name').value;
  $('favorite-note').textContent = `${state.favoritesUnavailable ? `${state.favoritesUnavailable} unavailable favorites were skipped. ` : ''}Favorites keep the complete preview settings across sessions. Loading uses the current source and mask. Names are unique.`;
}
function render(sync = false) {
  if (sync) { renderCatalog(); syncControls(); }
  renderHistory(); renderFavorites();
  $('settings').disabled = !state.draft || state.saving;
  $('apply').disabled = !state.canPreview;
  $('extract').disabled = !state.canPreview;
  $('random-style').disabled = !state.canRandom;
  const exportValid = renderExport();
  $('save').disabled = !state.canSave || jsonDirty || customDirty || !exportValid || !$('output').value.trim();
  $('output').disabled = state.saving;
  $('recipe-output').disabled = state.saving;
  $('save-recipe').disabled = !state.canSave || jsonDirty || customDirty || !$('recipe-output').value.trim();
  $('recipe-note').textContent = state.rendered?.request.mask_input
    ? `Keeps the preview dimensions and settings. Image and mask paths stay separate. Reuse with mask_input: ${state.rendered.request.mask_input}`
    : 'Keeps the preview dimensions and settings, including custom colors. Source paths and export size choices stay separate.';
  $('cancel').hidden = !['preview', 'extract', 'random', 'history', 'favorites-list', 'favorite-load', 'favorite-preview'].includes(state.pending?.kind);
  $('status').textContent = state.status;
  $('status-dot').className = `status-dot${state.busy ? ' busy' : ''}${state.error ? ' error' : ''}`;
  const dirty = state.dirty || jsonDirty || customDirty;
  $('preview-badge').textContent = state.busy ? ({ 'favorites-list': 'FAVORITES', 'favorite-load': 'LOADING FAVORITE', 'favorite-preview': 'LOADING FAVORITE', 'favorite-save': 'SAVING FAVORITE', history: 'RESTORING', random: 'NEW STYLE', save: 'SAVING', recipe: 'SAVING RECIPE', extract: 'EXTRACTING', preview: 'RENDERING' }[state.pending.kind]) : (dirty || !state.validPreview) && state.rendered ? 'UNAPPLIED' : state.rendered ? 'PREVIEW' : 'WAITING';
  $('preview-badge').classList.toggle('dirty', dirty && !!state.rendered);
  $('source').textContent = state.draft?.input?.split(/[\\/]/).pop() || 'Waiting for a source image';
  if (state.rendered) {
    const image = $('preview');
    if (image.getAttribute('src') !== state.rendered.src) image.src = state.rendered.src;
    image.hidden = false; $('empty').hidden = true;
    image.alt = `Dithered preview of ${state.rendered.request.input.split(/[\\/]/).pop()}, ${state.rendered.width} by ${state.rendered.height} pixels`;
    $('dimensions').textContent = `${state.rendered.width} × ${state.rendered.height} px`;
    $('preview-hint').textContent = dirty ? 'The image shows the last applied settings. Apply preview to see your changes.' : 'Choose an export size below. Save checks that the source and mask still match this preview.';
    setZoom(zoom);
  }
  if (favoritesAutoPending && state.canFavorites && state.validPreview) { favoritesAutoPending = false; queueMicrotask(() => { void refreshFavorites(true); }); }
}
function exportSelection() {
  return { mode: $('export-size').value, width: Number($('export-width').value), height: Number($('export-height').value) };
}
function renderExport() {
  $('export-settings').disabled = !state.rendered || state.busy;
  $('custom-size').hidden = $('export-size').value !== 'custom';
  if (state.rendered) {
    const value = state.rendered;
    $('size-preview').textContent = `Preview · ${value.width} × ${value.height} px`;
    $('size-source').textContent = `Source · ${value.sourceWidth} × ${value.sourceHeight} px`;
    for (const axis of ['width', 'height']) $('export-' + axis).max = value.exportLimits['max_' + axis];
  }
  try {
    const size = exportDimensions(state.rendered, exportSelection());
    $('export-note').textContent = $('export-size').value === 'preview'
      ? `${size.width} × ${size.height} px · same dimensions and settings as the preview.`
      : `${size.width} × ${size.height} px · source size includes orientation and crop. Changing resolution can change the dither pattern. All other settings stay the same.`;
    $('export-note').classList.remove('invalid');
    return true;
  } catch (error) {
    $('export-note').textContent = error.message;
    $('export-note').classList.toggle('invalid', !!state.rendered);
    return false;
  }
}
function setZoom(value) {
  zoom = value;
  $('canvas').classList.toggle('fit', value === 'fit');
  $('preview').style.width = value === 'fit' ? '' : `${state.rendered?.width * Number(value)}px`;
  $('preview').style.height = value === 'fit' ? '' : `${state.rendered?.height * Number(value)}px`;
  document.querySelectorAll('[data-zoom]').forEach((button) => { const selected = button.dataset.zoom === value; button.classList.toggle('selected', selected); button.setAttribute('aria-pressed', String(selected)); });
}
function updateDraft(update) {
  if (!state.draft) return;
  const draft = clone(state.draft); update(draft); state.edit(draft); render();
}
document.querySelectorAll('[data-option]').forEach((element) => element.addEventListener('input', () => {
  if (!state.draft) return;
  try {
    const mergeJSON = jsonDirty;
    const value = element.type === 'checkbox' ? element.checked : element.tagName === 'SELECT' ? element.value : element.value === '' ? 0 : Number(element.value);
    const draft = clone(state.draft);
    draft.options = editOption(draft.options, element.dataset.option, value, jsonDirty ? $('options-json').value : undefined);
    state.edit(draft);
    jsonDirty = false;
    if (mergeJSON) syncControls(element.type === 'number' ? element : undefined);
    else {
      if (element.type === 'range') $(`${element.id}-value`).textContent = String(Number(element.value));
      $('options-json').value = JSON.stringify(state.draft.options, null, 2);
      if (element.dataset.option === 'algorithm') $('algorithm-info').textContent = state.algorithms.find((algorithm) => algorithm.id === element.value)?.description ?? '';
    }
  } catch (error) { state.fail(error); syncControls(); }
  render();
}));
$('palette-search').addEventListener('input', renderPalettes);
$('palette').addEventListener('change', () => { updateDraft((draft) => { draft.palette = $('palette').value; delete draft.colors; }); renderSwatches(); });
document.querySelectorAll('[data-mode]').forEach((button) => button.addEventListener('click', () => {
  customDirty = false;
  updateDraft((draft) => {
    if (button.dataset.mode === 'custom') { draft.colors = state.palettes.find((palette) => palette.id === draft.palette)?.colors?.slice() ?? ['#000000', '#FFFFFF']; delete draft.palette; }
    else { draft.palette = $('palette').value || 'mono'; delete draft.colors; }
  }); syncControls();
}));
$('custom-colors').addEventListener('input', () => { customDirty = true; state.edit(clone(state.draft)); renderSwatches(); render(); });
$('options-json').addEventListener('input', () => { jsonDirty = true; state.edit(clone(state.draft)); render(); });
document.querySelectorAll('[data-zoom]').forEach((button) => button.addEventListener('click', () => setZoom(button.dataset.zoom)));
$('preview').addEventListener('error', () => { state.validPreview = false; state.fail(new Error('The PNG preview could not be displayed. Apply preview to try again.')); render(); });
$('apply').addEventListener('click', async (event) => {
  event.preventDefault();
  if (!state.canPreview) return;
  let operation;
  try {
    const draft = clone(state.draft);
    if (jsonDirty) draft.options = JSON.parse($('options-json').value);
    if (paletteMode() === 'custom') draft.colors = parseColors($('custom-colors').value);
    const request = studioRequest(draft);
    if (jsonDirty || customDirty) state.edit(request);
    jsonDirty = false; customDirty = false;
    operation = state.beginPreview(); abortController = new AbortController(); render(true);
    const result = await app.callServerTool({ name: 'dither_studio', arguments: operation.request }, { signal: abortController.signal });
    const accepted = state.finishPreview(operation, result); render(accepted);
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
async function refreshFavorites(automatic = false) {
  if (!state.canFavorites) { if (automatic) favoritesAutoPending = true; return; }
  let operation;
  try {
    operation = state.beginFavoritesList(); const controller = new AbortController(); abortController = controller; render();
    const result = await app.callServerTool({ name: 'dither_favorites_list', arguments: {} }, { signal: controller.signal });
    state.finishFavoritesList(operation, result); render();
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
}
$('favorite-refresh').addEventListener('click', () => { void refreshFavorites(); });
$('favorite-list').addEventListener('change', () => { favoriteChoice = $('favorite-list').value; });
$('favorite-name').addEventListener('input', () => render());
$('favorite-save').addEventListener('click', async () => {
  if (!state.canSave || jsonDirty || customDirty) return;
  let operation;
  try {
    operation = state.beginFavoriteSave($('favorite-name').value); render();
    const result = await app.callServerTool({ name: 'dither_favorite_save', arguments: operation.request });
    if (state.finishFavoriteSave(operation, result)) favoriteChoice = result.structuredContent.id;
    render();
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
$('favorite-load').addEventListener('click', async () => {
  if (!state.canFavorites) return;
  let operation;
  try {
    operation = state.beginFavoriteLoad(favoriteChoice); const controller = new AbortController(); abortController = controller; render();
    const result = await app.callServerTool({ name: 'dither_favorite_load', arguments: operation.request }, { signal: controller.signal });
    const preview = state.finishFavoriteLoad(operation, result);
    if (!preview) { render(); return; }
    operation = preview; render();
    const rendered = await app.callServerTool({ name: 'dither_studio', arguments: operation.request }, { signal: controller.signal });
    const accepted = state.finishFavoritePreview(operation, rendered);
    if (accepted) { jsonDirty = false; customDirty = false; }
    render(accepted);
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
$('history-look').addEventListener('change', () => { historyChoice = $('history-look').value; });
async function restoreHistory(target) {
  if (!state.canHistory) return;
  let operation;
  try {
    operation = state.beginHistory(target);
    historyChoice = String(operation.targetId);
    const controller = new AbortController(); abortController = controller; render();
    const result = await app.callServerTool({ name: 'dither_studio', arguments: operation.request }, { signal: controller.signal });
    const current = state.current(operation);
    const accepted = state.finishHistory(operation, result);
    if (current) historyChoice = state.historyActive === null ? '' : String(state.historyActive);
    if (accepted) { jsonDirty = false; customDirty = false; }
    render(accepted);
  } catch (error) {
    if (operation) {
      if (state.reject(operation, error)) historyChoice = state.historyActive === null ? '' : String(state.historyActive);
    } else state.fail(error);
    render();
  }
}
$('history-restore').addEventListener('click', () => { void restoreHistory(Number(historyChoice)); });
$('history-undo').addEventListener('click', () => { void restoreHistory('undo'); });
$('history-redo').addEventListener('click', () => { void restoreHistory('redo'); });
$('random-style').addEventListener('click', async () => {
  if (!state.canRandom) return;
  let operation;
  try {
    const draft = clone(state.draft);
    if (jsonDirty) {
      try { draft.options = JSON.parse($('options-json').value); }
      catch { throw new Error('Correct the advanced options JSON before choosing a random style.'); }
    }
    // Validate geometry before canceling work or replacing any pending text.
    const base = studioRequest(draft);
    historyChoice = state.historyActive === null ? '' : String(state.historyActive);
    abortController?.abort();
    operation = state.beginRandom(base);
    const controller = new AbortController(); abortController = controller; render();
    const style = await app.callServerTool({ name: 'dither_random_style', arguments: operation.request }, { signal: controller.signal });
    const preview = state.finishRandom(operation, style);
    if (!preview) { render(); return; }
    operation = preview; render();
    const result = await app.callServerTool({ name: 'dither_studio', arguments: operation.request }, { signal: controller.signal });
    const accepted = state.finishPreview(operation, result);
    if (accepted) { jsonDirty = false; customDirty = false; }
    render(accepted);
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
$('extract-count').addEventListener('input', () => {
  if (state.pending?.kind === 'extract') { state.edit(clone(state.draft)); render(); }
});
$('extract').addEventListener('click', async () => {
  if (!state.canPreview) return;
  let operation;
  try {
    operation = state.beginExtraction(Number($('extract-count').value));
    abortController = new AbortController(); render();
    const result = await app.callServerTool({ name: 'dither_palette_extract', arguments: operation.request }, { signal: abortController.signal });
    const accepted = state.finishExtraction(operation, result);
    if (accepted) customDirty = false;
    render(accepted);
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
$('cancel').addEventListener('click', () => { historyChoice = state.historyActive === null ? '' : String(state.historyActive); const kind = state.pending?.kind; abortController?.abort(); state.cancel(kind === 'favorites-list' ? 'Favorite refresh cancelled.' : kind === 'extract' ? 'Color extraction cancelled. Your settings were kept.' : kind === 'random' ? 'Random style cancelled. Your settings were kept.' : undefined); render(); });
$('output').addEventListener('input', () => render());
$('recipe-output').addEventListener('input', () => render());
for (const id of ['export-size', 'export-width', 'export-height']) $(id).addEventListener('input', () => render());
$('save').addEventListener('click', async (event) => {
  event.preventDefault();
  if (!state.canSave || jsonDirty || customDirty) return;
  let operation;
  try {
    operation = state.beginSave($('output').value, exportSelection()); render();
    const result = await app.callServerTool({ name: 'dither_render', arguments: operation.request });
    state.finishSave(operation, result); render();
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});
$('save-recipe').addEventListener('click', async () => {
  if (!state.canSave || jsonDirty || customDirty) return;
  let operation;
  try {
    operation = state.beginRecipeSave($('recipe-output').value, $('output').value); render();
    const result = await app.callServerTool({ name: 'dither_recipe_save', arguments: operation.request });
    state.finishRecipeSave(operation, result); render();
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});

// MCP Apps hosts can omit allow-forms. Route explicit actions through buttons.
for (const [form, button] of [['controls', 'apply'], ['save-form', 'save'], ['recipe-form', 'save-recipe'], ['favorite-form', 'favorite-save']]) {
  $(form).addEventListener('submit', (event) => event.preventDefault());
  $(form).addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && event.target.tagName === 'INPUT') {
      event.preventDefault();
      $(event.target.id === 'extract-count' ? 'extract' : button).click();
    }
  });
}

function applyContext(context) {
  if (!context) return;
  if (context.theme) applyDocumentTheme(context.theme);
  if (context.styles?.variables) {
    // System fonts stay inline. Host font CSS may reference external resources.
    const variables = Object.fromEntries(Object.entries(context.styles.variables).filter(([key, value]) => !key.startsWith('--font-') && typeof value === 'string' && !/url\s*\(/i.test(value)));
    applyHostStyleVariables(variables);
  }
  if (context.containerDimensions) {
    const dimensions = context.containerDimensions;
    const root = document.documentElement;
    for (const axis of ['width', 'height']) {
      const maxKey = `max${axis[0].toUpperCase()}${axis.slice(1)}`;
      root.style[axis] = Number.isFinite(dimensions[axis]) ? `${dimensions[axis]}px` : '';
      root.style[maxKey] = Number.isFinite(dimensions[maxKey]) ? `${dimensions[maxKey]}px` : '';
    }
  }
  if (context.safeAreaInsets) {
    const insets = context.safeAreaInsets;
    document.body.style.padding = ['top', 'right', 'bottom', 'left'].map((side) => `${Math.max(0, Number(insets[side]) || 0)}px`).join(' ');
  }
}

// Lifecycle handlers are registered before connecting so initial data cannot race them.
app.ontoolinput = ({ arguments: arguments_ }) => {
  abortController?.abort(); jsonDirty = false; customDirty = false; historyChoice = ''; lastHistoryActive = null; favoriteChoice = ''; favoritesAutoPending = true;
  $('export-size').value = 'preview'; $('export-width').value = ''; $('export-height').value = '';
  try { state.setInput(arguments_); } catch (error) { state.fail(error); }
  render(true);
};
app.ontoolresult = (result) => { const accepted = state.hostResult(result); if (accepted) { jsonDirty = false; customDirty = false; } render(accepted); };
app.ontoolcancelled = () => { historyChoice = state.historyActive === null ? '' : String(state.historyActive); abortController?.abort(); state.cancel('The host cancelled the request. Apply preview to try again.'); render(); };
app.onhostcontextchanged = applyContext;
app.onteardown = async () => { historyChoice = state.historyActive === null ? '' : String(state.historyActive); abortController?.abort(); state.cancel('The studio has closed.'); state.canCallTools = false; render(); return {}; };
// Protocol.onerror also reports late replies after cancellation; it does not
// establish transport closure. Ignore only those stale-response diagnostics.
app.onerror = (error) => {
  if (error?.message?.startsWith('Received a response for an unknown message ID:') || error?.message?.startsWith('Received a progress notification for an unknown token:')) return;
  state.fail(new Error('The host reported a communication error. Try the operation again.')); render();
};
app.onclose = () => {
  historyChoice = state.historyActive === null ? '' : String(state.historyActive);
  state.canCallTools = false;
  state.cancel('The host connection closed. Open the studio again to continue.');
  $('host-note').hidden = false; $('host-note').textContent = state.status; render();
};

render();
if (window.parent === window) {
  state.status = 'Open dither_studio in an MCP Apps host to use this interface.';
  $('host-note').hidden = false; $('host-note').textContent = 'This embedded interface communicates through an MCP Apps host. A standard MCP client still receives the PNG preview and recipe as a tool result.'; render();
} else {
  app.connect(new PostMessageTransport(window.parent, window.parent)).then(() => {
    state.canCallTools = !!app.getHostCapabilities()?.serverTools;
    applyContext(app.getHostContext());
    if (!state.canCallTools) {
      $('host-note').hidden = false;
      $('host-note').textContent = 'This host can display the preview but does not support app tool calls. Ask your agent to apply changes or save the image with dither_render.';
    }
    if (!state.draft && !state.rendered) state.status = 'Connected. Waiting for the source image…';
    render();
  }).catch(() => {
    state.canCallTools = false; state.status = 'The host could not connect to the studio.';
    $('host-note').hidden = false; $('host-note').textContent = 'Open dither_studio in an MCP Apps host. The tool result also includes a PNG preview and a text summary for standard MCP clients.'; render();
  });
}
