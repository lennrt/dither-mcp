// State transitions are independent of the DOM and the MCP transport.
export const clone = (value) => JSON.parse(JSON.stringify(value));
export function fingerprint(value) {
  if (Array.isArray(value)) return `[${value.map(fingerprint).join(',')}]`;
  if (value && typeof value === 'object') return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${fingerprint(value[key])}`).join(',')}}`;
  return JSON.stringify(value);
}

export function studioRequest(value) {
  if (!value || typeof value !== 'object' || typeof value.input !== 'string' || !value.input.trim()) throw new Error('The studio needs a source image from its tool call.');
  const request = { input: value.input, options: clone(value.options ?? {}) };
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
    this.status = 'Connecting to your host…';
    this.error = false;
  }
  get dirty() { return !this.rendered || !this.draft || fingerprint(this.draft) !== fingerprint(this.rendered.request); }
  get busy() { return !!this.pending; }
  get canPreview() { return this.canCallTools && !!this.draft && !this.busy; }
  get canSave() { return this.canCallTools && this.validPreview && !!this.rendered && !this.dirty && !this.busy; }
  setInput(value) {
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
  accept(preview, replaceDraft) {
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
