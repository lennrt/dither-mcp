import { App, PostMessageTransport, applyDocumentTheme, applyHostStyleVariables } from '@modelcontextprotocol/ext-apps';
import { StudioState, clone, parseColors, studioRequest } from './state.mjs';

const app = new App({ name: 'Dither studio', version: '0.1.0' }, {}, { autoResize: true });
const state = new StudioState();
const $ = (id) => document.getElementById(id);
let abortController;
let zoom = 'fit';
let jsonDirty = false;
let customDirty = false;
let lastCatalog = '';

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
function syncControls() {
  const options = state.draft?.options ?? {};
  document.querySelectorAll('[data-option]').forEach((element) => {
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
function render(sync = false) {
  if (sync) { renderCatalog(); syncControls(); }
  $('settings').disabled = !state.draft || state.pending?.kind === 'save';
  $('apply').disabled = !state.canPreview;
  $('save').disabled = !state.canSave || jsonDirty || customDirty || !$('output').value.trim();
  $('output').disabled = state.pending?.kind === 'save';
  $('cancel').hidden = state.pending?.kind !== 'preview';
  $('status').textContent = state.status;
  $('status-dot').className = `status-dot${state.busy ? ' busy' : ''}${state.error ? ' error' : ''}`;
  const dirty = state.dirty || jsonDirty || customDirty;
  $('preview-badge').textContent = state.busy ? (state.pending.kind === 'save' ? 'SAVING' : 'RENDERING') : (dirty || !state.validPreview) && state.rendered ? 'UNAPPLIED' : state.rendered ? 'PREVIEW' : 'WAITING';
  $('preview-badge').classList.toggle('dirty', dirty && !!state.rendered);
  $('source').textContent = state.draft?.input?.split(/[\\/]/).pop() || 'Waiting for a source image';
  if (state.rendered) {
    const image = $('preview');
    if (image.getAttribute('src') !== state.rendered.src) image.src = state.rendered.src;
    image.hidden = false; $('empty').hidden = true;
    image.alt = `Dithered preview of ${state.rendered.request.input.split(/[\\/]/).pop()}, ${state.rendered.width} by ${state.rendered.height} pixels`;
    $('dimensions').textContent = `${state.rendered.width} × ${state.rendered.height} px`;
    $('preview-hint').textContent = dirty ? 'The image shows the last applied settings. Apply preview to see your changes.' : 'Save uses these exact preview dimensions and settings.';
    setZoom(zoom);
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
  updateDraft((draft) => { draft.options[element.dataset.option] = element.type === 'checkbox' ? element.checked : element.tagName === 'SELECT' ? element.value : element.value === '' ? 0 : Number(element.value); });
  if (element.type === 'range') $(`${element.id}-value`).textContent = String(Number(element.value));
  if (!jsonDirty) $('options-json').value = JSON.stringify(state.draft.options, null, 2);
  if (element.dataset.option === 'algorithm') $('algorithm-info').textContent = state.algorithms.find((algorithm) => algorithm.id === element.value)?.description ?? '';
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
$('controls').addEventListener('submit', async (event) => {
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
$('cancel').addEventListener('click', () => { abortController?.abort(); state.cancel(); render(); });
$('output').addEventListener('input', () => render());
$('save-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  if (!state.canSave || jsonDirty || customDirty) return;
  let operation;
  try {
    operation = state.beginSave($('output').value); render();
    const result = await app.callServerTool({ name: 'dither_render', arguments: operation.request });
    state.finishSave(operation, result); render();
  } catch (error) { if (operation) state.reject(operation, error); else state.fail(error); render(); }
});

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
  abortController?.abort(); jsonDirty = false; customDirty = false;
  try { state.setInput(arguments_); } catch (error) { state.fail(error); }
  render(true);
};
app.ontoolresult = (result) => { const accepted = state.hostResult(result); if (accepted) { jsonDirty = false; customDirty = false; } render(accepted); };
app.ontoolcancelled = () => { abortController?.abort(); state.cancel('The host cancelled the request. Apply preview to try again.'); render(); };
app.onhostcontextchanged = applyContext;
app.onteardown = async () => { abortController?.abort(); state.cancel('The studio has closed.'); state.canCallTools = false; render(); return {}; };
app.onerror = () => { state.canCallTools = false; $('host-note').hidden = false; $('host-note').textContent = 'The host connection was interrupted. Open the studio again to continue.'; render(); };

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
