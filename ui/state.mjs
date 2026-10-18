// State transitions are independent of the DOM and the MCP transport.
export const clone = (value) => JSON.parse(JSON.stringify(value));
export function fingerprint(value) {
  if (Array.isArray(value)) return `[${value.map(fingerprint).join(',')}]`;
  if (value && typeof value === 'object') return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${fingerprint(value[key])}`).join(',')}}`;
  return JSON.stringify(value);
}

// A control edit follows any pending JSON edit and wins for that one field.
export function editOption(options, key, value, pendingJSON) {
  let source = options;
  if (pendingJSON !== undefined) {
    try {
      source = JSON.parse(pendingJSON);
      if (!source || Array.isArray(source) || typeof source !== 'object') throw new Error('object required');
    } catch {
      throw new Error('Correct the advanced options JSON before changing controls. Use a valid JSON object.');
    }
  }
  return { ...clone(source), [key]: value };
}

export function studioRequest(value) {
  if (!value || typeof value !== 'object' || typeof value.input !== 'string' || !value.input.trim()) throw new Error('The studio needs a source image from its tool call.');
  const request = { input: value.input, options: clone(value.options === undefined ? {} : value.options) };
  if (value.palette) request.palette = value.palette;
  if (value.colors) request.colors = clone(value.colors);
  if (value.mask_input) request.mask_input = value.mask_input;
  if (request.palette && request.colors) throw new Error('Choose a built-in palette or custom colors.');
  if (request.colors) parseColors(request.colors);
  if (!request.options || Array.isArray(request.options) || typeof request.options !== 'object') throw new Error('Options must be a JSON object.');
  for (const dimension of ['width', 'height']) {
    const number = request.options[dimension] ?? 0;
    if (!Number.isInteger(number) || number < 0 || number > 1024) throw new Error('Width and height must be whole numbers from 0 to 1024.');
  }
  if (request.options.seed !== undefined && !Number.isSafeInteger(request.options.seed)) throw new Error('Seed must be a whole number from −9007199254740991 to 9007199254740991.');
  return request;
}

export function parseColors(value) {
  const colors = (Array.isArray(value) ? value : String(value).trim().split(/[\s,;]+/)).map((color) => {
    const hex = String(color).trim().replace(/^#/, '');
    if (!/^(?:[\da-f]{3}|[\da-f]{6})$/i.test(hex)) throw new Error('Custom colors use hex values such as #172A35 or #FA6.');
    return `#${(hex.length === 3 ? [...hex].map((letter) => letter + letter).join('') : hex).toUpperCase()}`;
  });
  if (colors.length < 2 || colors.length > 256) throw new Error('Use between 2 and 256 custom colors.');
  if (new Set(colors).size !== colors.length) throw new Error('Custom colors must be unique.');
  return colors;
}

export function toolError(result) {
  return result?.content?.filter((item) => item.type === 'text').map((item) => item.text).join('\n') || 'The tool could not complete this request.';
}

function matchesOptions(expected, actual) {
  return Object.entries(expected).every(([key, value]) => {
    if ((key === 'width' || key === 'height') && value === 0) return true;
    if (value && typeof value === 'object' && !Array.isArray(value)) return matchesOptions(value, actual?.[key] ?? {});
    if (actual?.[key] === undefined && (value === 0 || value === false || value === '')) return true;
    return fingerprint(value) === fingerprint(actual?.[key]);
  });
}
function matchesRequest(expected, actual) {
  return expected.input === actual.input && (expected.mask_input ?? '') === (actual.mask_input ?? '') &&
    (!expected.palette || expected.palette === actual.palette) &&
    (!expected.colors || fingerprint(parseColors(expected.colors)) === fingerprint(parseColors(actual.colors ?? []))) &&
    matchesOptions(expected.options ?? {}, actual.options ?? {});
}

export function decodePreview(result) {
  if (result?.isError) throw new Error(toolError(result));
  const meta = result?._meta?.dither;
  const data = result?.structuredContent;
  const image = result?.content?.find((item) => item.type === 'image' && item.mimeType === 'image/png');
  if (!meta?.request || !data || !image || typeof image.data !== 'string' || !/^[A-Za-z0-9+/=\r\n]+$/.test(image.data) || image.data.length > 16 * 1024 * 1024) throw new Error('The studio received an incomplete preview. Apply the preview again.');
  const request = studioRequest(meta.request);
  if (data.mime_type !== 'image/png') throw new Error('The studio preview must use PNG.');
  if (data.path !== request.input || (data.mask_input ?? '') !== (request.mask_input ?? '')) throw new Error('The preview source or mask does not match its settings.');
  if (![data.width, data.height].every((number) => Number.isInteger(number) && number > 0 && number <= 1024)) throw new Error('The preview dimensions are outside the studio limit.');
  if (request.options.width !== data.width || request.options.height !== data.height) throw new Error('The preview settings do not match its dimensions.');
  const recipe = data.recipe;
  if (recipe?.version !== 1 || fingerprint(recipe.options) !== fingerprint(request.options) || (recipe.palette ?? '') !== (request.palette ?? '') || fingerprint(recipe.colors ?? []) !== fingerprint(request.colors ?? [])) throw new Error('The preview recipe does not match its settings.');
  const validHash = (value) => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);
  if (!validHash(data.source_sha256) || (request.mask_input ? !validHash(data.mask_sha256) : data.mask_sha256 !== undefined)) throw new Error('The preview needs valid source and mask fingerprints. Apply preview again.');
  if (![data.source_width, data.source_height, data.export_limits?.max_width, data.export_limits?.max_height, data.export_limits?.max_pixels].every((value) => Number.isSafeInteger(value) && value > 0)) throw new Error('The preview needs source dimensions and export limits. Apply preview again.');
  return { request, width: data.width, height: data.height, sourceWidth: data.source_width, sourceHeight: data.source_height, sourceSHA256: data.source_sha256, maskSHA256: data.mask_sha256, exportLimits: clone(data.export_limits), src: `data:image/png;base64,${image.data}`, algorithms: clone(meta.algorithms ?? []), palettes: clone(meta.palettes ?? []), recipe: clone(recipe) };
}

// Random styles replace appearance while keeping the user's composition and mask.
export function randomStyleRequest(current, result, algorithms = [], palettes = []) {
  if (result?.isError) throw new Error(toolError(result));
  const data = result?.structuredContent;
  const recipe = data?.recipe;
  const options = recipe?.options;
  const geometry = ['width', 'height', 'crop', 'mask', 'pixel_scale', 'resize_filter'];
  if (!Number.isSafeInteger(data?.seed) || recipe?.version !== 1 || typeof recipe.palette !== 'string' || !recipe.palette || Object.keys(recipe).some(key => !['version', 'palette', 'options'].includes(key)) || !options || Array.isArray(options) || typeof options !== 'object' || typeof options.algorithm !== 'string' || (options.seed ?? 0) !== data.seed || geometry.some(key => key in options) || (algorithms.length && !algorithms.some(item => item.id === options.algorithm)) || (palettes.length && !palettes.some(item => item.id === recipe.palette))) throw new Error('The host returned an invalid random style. Your settings were kept.');
  const request = { input: current.input, palette: recipe.palette, options: clone(options) };
  if (current.mask_input) request.mask_input = current.mask_input;
  for (const key of geometry) if (current.options[key] !== undefined) request.options[key] = clone(current.options[key]);
  return studioRequest(request);
}

// Export size is separate from preview settings. Every other option stays intact.
export function exportDimensions(preview, selection = { mode: 'preview' }) {
  if (!preview) throw new Error('Apply preview to choose an export size.');
  let width, height;
  switch (selection.mode) {
    case 'preview': ({ width, height } = preview); break;
    case 'source': width = preview.sourceWidth; height = preview.sourceHeight; break;
    case 'custom': ({ width, height } = selection); break;
    default: throw new Error('Choose preview, source, or custom export size.');
  }
  if (![width, height].every((value) => Number.isSafeInteger(value) && value > 0)) throw new Error('Enter positive whole numbers for both export dimensions.');
  const limits = preview.exportLimits;
  if (width > limits.max_width || height > limits.max_height || width > Math.floor(limits.max_pixels / height)) throw new Error(`Export must fit within ${limits.max_width} × ${limits.max_height} px and ${limits.max_pixels.toLocaleString('en-US')} pixels in total.`);
  return { width, height };
}

export class StudioState {
  constructor() {
    this.draft = null;
    this.rendered = null;
    this.algorithms = [];
    this.palettes = [];
    this.revision = 0;
    this.sequence = 0;
    this.pending = null;
    this.validPreview = false;
    this.hostPending = null;
    this.canCallTools = false;
    this.history = [];
    this.historyActive = null;
    this.historySequence = 0;
    this.undoIDs = [];
    this.redoIDs = [];
    this.status = 'Connecting to your host…';
    this.error = false;
  }
  get dirty() { return !this.rendered || !this.draft || fingerprint(this.draft) !== fingerprint(this.rendered.request); }
  get busy() { return !!this.pending; }
  get saving() { return ['save', 'recipe'].includes(this.pending?.kind); }
  get canHistory() { return this.canCallTools && !!this.draft && !!this.history.length && !this.busy; }
  get canUndo() { return this.canHistory && !!this.undoIDs.length; }
  get canRedo() { return this.canHistory && !!this.redoIDs.length; }
  get canRandom() { return this.canCallTools && !!this.draft && !this.saving; }
  get canPreview() { return this.canCallTools && !!this.draft && !this.busy; }
  get canSave() { return this.canCallTools && this.validPreview && !!this.rendered && !this.dirty && !this.busy; }
  setInput(value) {
    this.history = []; this.historyActive = null; this.historySequence = 0; this.undoIDs = []; this.redoIDs = [];
    this.pending = null;
    this.validPreview = false;
    this.hostPending = null;
    this.draft = null;
    this.sequence++;
    this.revision++;
    this.draft = studioRequest(value);
    this.hostInput = clone(this.draft);
    this.hostPending = { revision: this.revision };
    this.validPreview = false;
    this.status = 'Preparing the source preview…';
    this.error = false;
  }
  edit(value) {
    // Validation happens on Apply; incomplete typing still makes the draft dirty.
    this.draft = clone(value);
    this.revision++;
    this.status = 'Settings changed. Apply preview before saving.';
    this.error = false;
  }
  hostResult(result) {
    if (!result?._meta?.dither && !result?.isError) return false;
    // Host results have no app request ID. Edits close their delivery window.
    if (!this.hostPending || this.pending || this.hostPending.revision !== this.revision) return false;
    try {
      const preview = decodePreview(result);
      // A response for an older source cannot replace the current source.
      if (this.hostInput && !matchesRequest(this.hostInput, preview.request)) return false;
      this.hostPending = null;
      this.accept(preview, true);
      return true;
    } catch (error) { this.hostPending = null; this.fail(error); return false; }
  }
  beginHistory(target) {
    if (!this.canHistory) throw new Error('History is unavailable while another request is running or without host tools.');
    const navigation = target === 'undo' || target === 'redo' ? target : 'pick';
    const targetId = navigation === 'undo' ? this.undoIDs.at(-1) : navigation === 'redo' ? this.redoIDs.at(-1) : target;
    const entry = this.history.find(item => item.id === targetId);
    if (!entry) throw new Error('Choose a look from this studio session.');
    this.hostPending = null; this.validPreview = false;
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'history', navigation, targetId, request: clone(entry.request) };
    this.pending = operation; this.status = `Restoring look ${entry.id}…`; this.error = false;
    return clone(operation);
  }
  finishHistory(operation, result) {
    if (!this.current(operation)) return false;
    this.pending = null;
    if (operation.revision !== this.revision) {
      this.status = 'Settings changed while restoring. Choose the look again to restore it.';
      return false;
    }
    try {
      const preview = decodePreview(result);
      if (fingerprint(operation.request) !== fingerprint(preview.request)) throw new Error('The restored preview returned different source settings.');
      this.accept(preview, true, operation);
      return true;
    } catch (error) { this.fail(error); return false; }
  }
  remember(request, navigation) {
    const previous = this.historyActive;
    const signature = fingerprint(request);
    let entry = navigation ? this.history.find(item => item.id === navigation.targetId) : this.history.find(item => fingerprint(item.request) === signature);
    if (!entry) entry = { id: ++this.historySequence, request: clone(request) };
    else entry = { id: entry.id, request: clone(request) };
    if (previous !== null && previous !== entry.id) {
      if (navigation?.navigation === 'undo') {
        this.undoIDs.pop(); this.redoIDs.push(previous);
      } else if (navigation?.navigation === 'redo') {
        this.redoIDs.pop(); this.undoIDs.push(previous);
      } else {
        this.undoIDs.push(previous); this.redoIDs = [];
      }
    }
    this.historyActive = entry.id;
    this.history = [...this.history.filter(item => item.id !== entry.id), entry].slice(-20);
    const available = new Set(this.history.map(item => item.id));
    const prune = ids => {
      const kept = [];
      for (const id of ids) if (available.has(id) && kept.at(-1) !== id) kept.push(id);
      while (kept.at(-1) === this.historyActive) kept.pop();
      return kept.slice(-20);
    };
    this.undoIDs = prune(this.undoIDs); this.redoIDs = prune(this.redoIDs);
  }
  beginRandom(value = this.draft) {
    if (!this.canRandom) throw new Error('Random styles are unavailable while saving or without host tools.');
    const base = studioRequest(value);
    this.hostPending = null;
    this.validPreview = false;
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'random', request: {}, base };
    this.pending = operation;
    this.status = 'Finding a new look…'; this.error = false;
    return clone(operation);
  }
  finishRandom(operation, result) {
    if (!this.current(operation)) return null;
    this.pending = null;
    if (operation.revision !== this.revision) {
      this.status = 'Settings changed while choosing a style. Click Random style again to try another.';
      return null;
    }
    try {
      const request = randomStyleRequest(operation.base, result, this.algorithms, this.palettes);
      const preview = { id: ++this.sequence, revision: this.revision, kind: 'preview', request };
      this.pending = preview;
      this.status = 'Previewing your new style…'; this.error = false;
      return clone(preview);
    } catch (error) { this.fail(error); return null; }
  }
  beginExtraction(count = 8) {
    if (!this.canPreview) throw new Error('Palette tools are unavailable or another request is running.');
    if (!Number.isInteger(count) || count < 2 || count > 256) throw new Error('Choose a whole number from 2 to 256 colors.');
    this.hostPending = null;
    this.validPreview = false;
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'extract', request: { input: this.draft.input, count } };
    this.pending = operation;
    this.status = 'Extracting colors from the source image…';
    this.error = false;
    return clone(operation);
  }
  finishExtraction(operation, result) {
    if (!this.current(operation)) return false;
    this.pending = null;
    if (operation.revision !== this.revision) {
      this.status = 'Settings changed while extracting. Extract colors again to use them.';
      return false;
    }
    try {
      if (result?.isError) throw new Error(toolError(result));
      const values = result?.structuredContent?.colors;
      if (!Array.isArray(values) || values.some((value) => typeof value !== 'string') || values.length > operation.request.count) throw new Error('The tool returned an invalid color palette. Your settings were kept.');
      if (values.length < 2) throw new Error('The source has fewer than two visible colors. Choose a library palette or add custom colors.');
      const colors = parseColors(values);
      const draft = { ...clone(this.draft), colors };
      delete draft.palette;
      this.edit(draft);
      this.status = `Extracted ${colors.length} colors. Apply preview before saving.`;
      return true;
    } catch (error) { this.fail(error); return false; }
  }
  beginPreview() {
    if (!this.canPreview) throw new Error('Preview tools are unavailable or another request is running.');
    const request = studioRequest(this.draft);
    this.hostPending = null;
    this.validPreview = false;
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'preview', request };
    this.pending = operation;
    this.status = 'Rendering preview…';
    this.error = false;
    return clone(operation);
  }
  finishPreview(operation, result) {
    if (!this.current(operation)) return false;
    this.pending = null;
    if (operation.revision !== this.revision) {
      this.status = 'Settings changed while rendering. Apply preview again.';
      return false;
    }
    try {
      const preview = decodePreview(result);
      if (!matchesRequest(operation.request, preview.request)) throw new Error('The preview returned different source settings.');
      this.accept(preview, true);
      return true;
    } catch (error) { this.fail(error); return false; }
  }
  accept(preview, replaceDraft, navigation) {
    this.remember(preview.request, navigation);
    this.rendered = preview;
    this.validPreview = true;
    if (replaceDraft) { this.draft = clone(preview.request); this.revision++; }
    if (preview.algorithms.length) this.algorithms = preview.algorithms;
    if (preview.palettes.length) this.palettes = preview.palettes;
    this.status = `Preview ready · ${preview.width} × ${preview.height} px`;
    this.error = false;
  }
  beginSave(output, selection = { mode: 'preview' }) {
    if (!this.canSave) throw new Error('Apply the current preview before saving.');
    if (typeof output !== 'string' || !output.trim()) throw new Error('Enter a new output path relative to the workspace.');
    const dimensions = exportDimensions(this.rendered, selection);
    const request = { ...clone(this.rendered.request), output: output.trim(), expected_source_sha256: this.rendered.sourceSHA256 };
    if (this.rendered.maskSHA256) request.expected_mask_sha256 = this.rendered.maskSHA256;
    Object.assign(request.options, dimensions);
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'save', request };
    this.pending = operation;
    this.status = `Saving ${dimensions.width} × ${dimensions.height} px…`;
    this.error = false;
    return clone(operation);
  }
  beginRecipeSave(output, imageOutput = '') {
    if (!this.canSave) throw new Error('Apply the current preview before saving a recipe.');
    if (typeof output !== 'string' || !output.trim()) throw new Error('Enter a new recipe path relative to the workspace.');
    output = output.trim();
    if (!/\.json$/i.test(output) || output.includes('\\') || output.includes('\0') || output.startsWith('/') || /^[a-z]+:/i.test(output) || output.split('/').some((part) => !part || part === '.' || part === '..')) throw new Error('Use a relative .json recipe path with no traversal or URL.');
    if ([this.rendered.request.input, this.rendered.request.mask_input, imageOutput.trim()].includes(output)) throw new Error('Choose a recipe path different from the source, mask, and image output.');
    const request = { output, recipe: clone(this.rendered.recipe) };
    const operation = { id: ++this.sequence, revision: this.revision, kind: 'recipe', request };
    this.pending = operation;
    this.status = 'Saving the preview recipe…';
    this.error = false;
    return clone(operation);
  }
  finishRecipeSave(operation, result) {
    if (!this.current(operation)) return false;
    this.pending = null;
    if (operation.revision !== this.revision) {
      this.status = 'Settings changed while saving the recipe. Check the destination before retrying.';
      return false;
    }
    if (result?.isError) { this.fail(new Error(toolError(result))); return false; }
    const receipt = result?.structuredContent;
    if (receipt?.path !== operation.request.output || receipt.format !== 'json' || !Number.isSafeInteger(receipt.bytes) || receipt.bytes <= 0 || typeof receipt.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(receipt.sha256) || fingerprint(receipt.recipe) !== fingerprint(operation.request.recipe)) {
      this.fail(new Error('The host returned an incomplete or different recipe save receipt. Check the destination before retrying.'));
      return false;
    }
    this.status = `Saved recipe ${receipt.path}`;
    this.error = false;
    return true;
  }
  finishSave(operation, result) {
    if (!this.current(operation)) return false;
    this.pending = null;
    if (result?.isError) {
      const message = toolError(result);
      const code = result?.structuredContent?.error?.code;
      if (['source_changed', 'mask_changed'].includes(code) || /^(source_changed|mask_changed):/.test(message)) {
        this.validPreview = false;
        const changed = code || message.split(':')[0];
        this.fail(new Error(`${changed === 'mask_changed' ? 'The image mask' : 'The source image'} changed or could not be read. Apply preview again before saving.`));
      } else this.fail(new Error(message));
      return false;
    }
    const path = result?.structuredContent?.path;
    this.status = `Saved ${typeof path === 'string' ? path : operation.request.output}${this.dirty ? '. Apply your changed settings before saving again.' : ''}`;
    this.error = false;
    return true;
  }
  current(operation) { return this.pending?.id === operation.id && this.pending?.kind === operation.kind; }
  reject(operation, error) { if (!this.current(operation)) return false; this.pending = null; this.fail(error); return true; }
  cancel(reason = 'Preview cancelled. You can apply it again.') { this.pending = null; this.hostPending = null; this.validPreview = false; this.sequence++; this.status = reason; this.error = false; }
  fail(error) { this.status = error instanceof Error ? error.message : String(error); this.error = true; }
}
