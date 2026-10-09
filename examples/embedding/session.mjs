// Node/desktop side: one source image, two tools, and one embedded UI resource.
import path from 'node:path';
import { realpath, stat } from 'node:fs/promises';
import { Client } from '@modelcontextprotocol/client';
import { StdioClientTransport } from '@modelcontextprotocol/client/stdio';

export function relativeImagePath(value, label = 'path') {
  if (typeof value !== 'string' || !value.trim() || value.includes('\\') || value.includes('\0') || path.posix.isAbsolute(value) || /^[a-z]+:/i.test(value) || value.split('/').some((part) => !part || part === '.' || part === '..')) {
    throw new Error(`${label} must be a relative workspace path with no traversal or URL.`);
  }
  return value;
}

export function validateStudioCall(params, { input, maskInput }) {
  if (!params || !['dither_studio', 'dither_render'].includes(params.name)) throw new Error('Only dither_studio and dither_render are allowed.');
  if (Object.keys(params).some((key) => !['name', 'arguments', '_meta'].includes(key))) throw new Error('Unknown tool call field.');
  const args = params.arguments;
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
    const args = { palette: 'oat-and-ink', options: { width: 512, algorithm: 'atkinson' }, ...initial, input, ...(maskInput ? { mask_input: maskInput } : {}) };
    validateStudioCall({ name: 'dither_studio', arguments: args }, { input, maskInput });
    const result = await client.callTool({ name: 'dither_studio', arguments: args });
    if (result.isError) throw new Error(result.content.filter((item) => item.type === 'text').map((item) => item.text).join('\n'));
    let pending = false;
    return {
      bootstrap: { resource, arguments: args, result },
      async callTool(params, options) {
        validateStudioCall(params, { input, maskInput });
        if (pending) throw new Error('A studio operation is already running.');
        pending = true;
        try { return await client.callTool(params, options); } finally { pending = false; }
      },
      close: () => client.close(),
    };
  } catch (error) { await client.close(); throw error; }
}
