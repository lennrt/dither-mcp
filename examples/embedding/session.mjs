// Node/desktop side: one source image, eight tools, and one embedded UI resource.
import path from 'node:path';
import { realpath, stat } from 'node:fs/promises';
import { isDeepStrictEqual } from 'node:util';
import { favoriteName } from '../../ui/state.mjs';
import { Client } from '@modelcontextprotocol/client';
import { StdioClientTransport } from '@modelcontextprotocol/client/stdio';

export function relativeImagePath(value, label = 'path') {
  if (typeof value !== 'string' || !value.trim() || value.includes('\\') || value.includes('\0') || path.posix.isAbsolute(value) || /^[a-z]+:/i.test(value) || value.split('/').some((part) => !part || part === '.' || part === '..')) {
    throw new Error(`${label} must be a relative workspace path with no traversal or URL.`);
  }
  return value;
}

export function validateStudioCall(params, { input, maskInput, acceptedRecipe }) {
  if (!params || !['dither_studio', 'dither_render', 'dither_palette_extract', 'dither_recipe_save', 'dither_random_style', 'dither_favorites_list', 'dither_favorite_load', 'dither_favorite_save'].includes(params.name)) throw new Error('Only studio, render, palette extraction, recipe save, random style, and favorite tools are allowed.');
  if (Object.keys(params).some((key) => !['name', 'arguments', '_meta'].includes(key))) throw new Error('Unknown tool call field.');
  const args = params.arguments;
  if (params.name === 'dither_favorites_list') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || Object.keys(args).length) throw new Error('Favorites listing accepts an empty argument object.');
    return params;
  }
  if (params.name === 'dither_favorite_load') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || Object.keys(args).some(key => key !== 'id') || typeof args.id !== 'string' || !/^[a-f0-9]{64}$/.test(args.id)) throw new Error('Favorite loading accepts only a valid favorite ID.');
    return params;
  }
  if (params.name === 'dither_favorite_save') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || Object.keys(args).some(key => !['name', 'recipe'].includes(key))) throw new Error('Favorite saving accepts only name and recipe.');
    favoriteName(args.name);
    if (!acceptedRecipe || !isDeepStrictEqual(args.recipe, acceptedRecipe)) throw new Error('Apply a successful preview before saving its exact favorite recipe.');
    return params;
  }
  if (params.name === 'dither_random_style') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || Object.keys(args).some(key => key !== 'seed') || ('seed' in args && !Number.isSafeInteger(args.seed))) throw new Error('Random style accepts only an optional safe integer seed.');
    return params;
  }
  if (params.name === 'dither_recipe_save') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || Object.keys(args).some((key) => !['output', 'recipe'].includes(key))) throw new Error('Recipe saving accepts only output and recipe.');
    relativeImagePath(args.output, 'recipe output');
    if (!/\.json$/i.test(args.output)) throw new Error('The recipe output must use a .json extension.');
    if (args.output === input || args.output === maskInput) throw new Error('The recipe output cannot replace the source or mask.');
    if (!acceptedRecipe || !isDeepStrictEqual(args.recipe, acceptedRecipe)) throw new Error('Apply a successful preview before saving its exact recipe.');
    return params;
  }
  if (params.name === 'dither_palette_extract') {
    if (!args || Array.isArray(args) || typeof args !== 'object' || args.input !== input) throw new Error('The source must match this embedding session.');
    if (Object.keys(args).some((key) => !['input', 'count'].includes(key))) throw new Error('Unsupported palette extraction argument.');
    if (!Number.isInteger(args.count) || args.count < 2 || args.count > 256) throw new Error('Palette extraction needs a whole number from 2 to 256 colors.');
    return params;
  }
  if (!args || Array.isArray(args) || typeof args !== 'object' || args.input !== input || (args.mask_input ?? '') !== (maskInput ?? '')) throw new Error('The source and mask must match this embedding session.');
  const fields = ['input', 'mask_input', 'palette', 'colors', 'options'];
  if (params.name === 'dither_render') fields.push('output', 'expected_source_sha256', 'expected_mask_sha256');
  if (Object.keys(args).some((key) => !fields.includes(key))) throw new Error('Unsupported studio argument.');
  if (params.name === 'dither_render') {
    relativeImagePath(args.output, 'output');
    if (args.output === input || args.output === maskInput) throw new Error('The output cannot replace the source or mask.');
  }
  return params; // Preserve _meta, render guards, options and result shape verbatim.
}

export async function createStudioSession({ binary, root, input, maskInput, initial = {} }) {
  if (!path.isAbsolute(root ?? '') || !path.isAbsolute(binary ?? '')) throw new Error('binary and root must be absolute paths.');
  const workspace = await realpath(root);
  if (!(await stat(workspace)).isDirectory()) throw new Error('root must be a directory.');
  relativeImagePath(input, 'input');
  if (maskInput) relativeImagePath(maskInput, 'mask');
  for (const value of [input, maskInput].filter(Boolean)) {
    const resolved = await realpath(path.join(workspace, value));
    if (!resolved.startsWith(workspace + path.sep) || !(await stat(resolved)).isFile()) throw new Error('The source and mask must be files inside root.');
  }
  const client = new Client({ name: 'dither-embedding-example', version: '1' }, { capabilities: { extensions: { 'io.modelcontextprotocol/ui': { mimeTypes: ['text/html;profile=mcp-app'] } } } });
  try {
    await client.connect(new StdioClientTransport({ command: binary, args: ['mcp', '--root', workspace], cwd: workspace, stderr: 'inherit' }));
    const descriptor = (await client.listTools()).tools.find((tool) => tool.name === 'dither_studio');
    if (descriptor?._meta?.ui?.resourceUri !== 'ui://dither/studio.html') throw new Error('The server did not negotiate the studio MCP App. Rebuild the binary.');
    const resource = (await client.readResource({ uri: descriptor._meta.ui.resourceUri })).contents[0];
    if (resource?.mimeType !== 'text/html;profile=mcp-app' || typeof resource.text !== 'string') throw new Error('Invalid studio resource.');
    const args = { options: { algorithm: 'atkinson' }, ...initial, input, ...(maskInput ? { mask_input: maskInput } : {}) };
    if (args.palette === undefined && args.colors === undefined) args.palette = 'oat-and-ink';
    validateStudioCall({ name: 'dither_studio', arguments: args }, { input, maskInput });
    const result = await client.callTool({ name: 'dither_studio', arguments: args });
    if (result.isError) throw new Error(result.content.filter((item) => item.type === 'text').map((item) => item.text).join('\n'));
    let pending = null;
    let acceptedRecipe = structuredClone(result.structuredContent?.recipe);
    return {
      bootstrap: { resource, arguments: args, result },
      async callTool(params, options) {
        if (options?.signal?.aborted) throw new DOMException('The studio request was aborted.', 'AbortError');
        validateStudioCall(params, { input, maskInput, acceptedRecipe });
        const readOnly = name => ['dither_studio', 'dither_palette_extract', 'dither_random_style', 'dither_favorites_list', 'dither_favorite_load'].includes(name);
        if (pending && (params.name !== 'dither_random_style' || !readOnly(pending.name))) throw new Error('A studio operation is already running.');
        pending?.controller.abort();
        const operation = { name: params.name, controller: new AbortController() };
        const signal = options?.signal ? AbortSignal.any([options.signal, operation.controller.signal]) : operation.controller.signal;
        pending = operation;
        if (readOnly(params.name) && params.name !== 'dither_favorites_list') acceptedRecipe = undefined;
        try {
          const result = await client.callTool(params, { ...options, signal });
          if (pending === operation && params.name === 'dither_studio' && !result.isError && !signal.aborted) acceptedRecipe = structuredClone(result.structuredContent?.recipe);
          return result;
        } finally { if (pending === operation) pending = null; }
      },
      close: () => client.close(),
    };
  } catch (error) { await client.close(); throw error; }
}
