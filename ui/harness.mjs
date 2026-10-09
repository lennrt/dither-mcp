// Development-only host. It serves an isolated copy of original sample artwork.
import http from 'node:http';
import { readFile, mkdir, mkdtemp, copyFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Client } from '@modelcontextprotocol/client';
import { StdioClientTransport } from '@modelcontextprotocol/client/stdio';
import { build } from 'esbuild';

const directory = path.dirname(fileURLToPath(import.meta.url));
const root = path.dirname(directory);
const port = Number(process.env.DITHER_APPS_PORT || 8787);
if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('DITHER_APPS_PORT must be an integer from 1024 through 65535.');
const origin = `http://localhost:${port}`;
const sandboxOrigin = `http://127.0.0.1:${port}`;
await mkdir(path.join(root, 'work'), { recursive: true });
const workspace = await mkdtemp(path.join(root, 'work/apps-host-'));
await copyFile(path.join(root, 'docs/assets/source/moon-garden.png'), path.join(workspace, 'source.png'));
const client = new Client({ name: 'dither-apps-local-test-host', version: '1' }, { capabilities: { extensions: { 'io.modelcontextprotocol/ui': { mimeTypes: ['text/html;profile=mcp-app'] } } } });
const transport = new StdioClientTransport({ command: path.resolve(process.env.DITHER_MCP_BINARY || path.join(root, 'bin/dither-mcp')), args: ['mcp', '--root', workspace], cwd: root, stderr: 'inherit' });
await client.connect(transport);
const tools = await client.listTools();
const studio = tools.tools.find((tool) => tool.name === 'dither_studio');
if (studio?._meta?.ui?.resourceUri !== 'ui://dither/studio.html') throw new Error('The Go server did not negotiate MCP Apps. Rebuild bin/dither-mcp.');
const resource = (await client.readResource({ uri: studio._meta.ui.resourceUri })).contents[0];
const args = { input: 'source.png', palette: 'oat-and-ink', options: { algorithm: 'atkinson', width: 512 } };
const result = await client.callTool({ name: 'dither_studio', arguments: args });
if (result.isError) throw new Error('The initial sample preview failed.');
const script = (await build({ entryPoints: [path.join(directory, 'harness-client.mjs')], bundle: true, write: false, format: 'esm', platform: 'browser', target: ['es2022'], logLevel: 'silent' })).outputFiles[0].text;
const staticPaths = new Map([['/', ['harness.html', 'text/html; charset=utf-8']], ['/sandbox', ['harness-sandbox.html', 'text/html; charset=utf-8']]]);
let pending = false;
function send(response, status, type, value) {
  response.writeHead(status, { 'Content-Type': type, 'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff', 'Referrer-Policy': 'no-referrer' });
  response.end(value);
}
const server = http.createServer(async (request, response) => {
  try {
    if (![new URL(origin).host, new URL(sandboxOrigin).host].includes(request.headers.host)) return send(response, 403, 'text/plain', 'Host rejected.');
    const url = new URL(request.url, origin);
    if (request.method === 'GET' && staticPaths.has(url.pathname)) {
      const [name, type] = staticPaths.get(url.pathname);
      return send(response, 200, type, await readFile(path.join(directory, name)));
    }
    if (request.method === 'GET' && url.pathname === '/host.js') return send(response, 200, 'text/javascript', script);
    if (request.headers.host !== new URL(origin).host || (request.headers.origin && request.headers.origin !== origin) || ['cross-site', 'same-site'].includes(request.headers['sec-fetch-site'])) return send(response, 403, 'text/plain', 'Origin rejected.');
    if (request.method === 'GET' && url.pathname === '/bootstrap') return send(response, 200, 'application/json', JSON.stringify({ resource, arguments: args, result }));
    if (request.method !== 'POST' || url.pathname !== '/call') return send(response, 404, 'text/plain', 'Not found.');
    if (request.headers.origin !== origin || request.headers['content-type']?.split(';')[0] !== 'application/json') return send(response, 403, 'text/plain', 'Use the local host interface.');
    if (pending) return send(response, 429, 'application/json', JSON.stringify({ error: 'A tool call is already running.' }));
    let body = '';
    for await (const chunk of request) {
      body += chunk.toString('utf8');
      if (Buffer.byteLength(body) > 1 << 20) return send(response, 413, 'text/plain', 'Request too large.');
    }
    const call = JSON.parse(body);
    if (!['dither_studio', 'dither_render'].includes(call.name) || call.arguments?.input !== 'source.png' || call.arguments.mask_input || call.arguments.recipe) return send(response, 403, 'application/json', JSON.stringify({ error: 'This test host uses only its copied sample image.' }));
    if (pending) return send(response, 429, 'application/json', JSON.stringify({ error: 'A tool call is already running.' }));
    pending = true;
    const controller = new AbortController();
    response.on('close', () => { if (!response.writableEnded) controller.abort(); });
    try {
      const toolResult = await client.callTool({ name: call.name, arguments: call.arguments }, { signal: controller.signal });
      send(response, 200, 'application/json', JSON.stringify(toolResult));
    } finally { pending = false; }
  } catch (error) {
    if (!response.headersSent) send(response, 500, 'application/json', JSON.stringify({ error: error instanceof Error ? error.message : 'The test host request failed.' }));
    else response.end();
  }
});
server.requestTimeout = 125_000;
server.headersTimeout = 10_000;
server.listen(port, '127.0.0.1', () => {
  console.log(`MCP Apps test host: ${origin}`);
  console.log(`Sample workspace: ${workspace}`);
  console.log('Tool calls use the actual Go server over stdio. Stop with Ctrl+C.');
});
async function close() {
  server.closeAllConnections();
  server.close();
  await client.close();
}
process.once('SIGINT', () => { void close(); });
process.once('SIGTERM', () => { void close(); });
server.once('error', (error) => { console.error(error.message); void client.close(); process.exitCode = 1; });
