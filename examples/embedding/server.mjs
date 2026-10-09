import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { readFile } from 'node:fs/promises';
import { randomBytes } from 'node:crypto';
import { parseArgs } from 'node:util';
import { build } from 'esbuild';
import { createStudioSession } from './session.mjs';

const directory = path.dirname(fileURLToPath(import.meta.url));
const repository = path.resolve(directory, '../..');
export async function startStudioHost({ root, input, maskInput, binary = path.join(repository, 'bin', process.platform === 'win32' ? 'dither-mcp.exe' : 'dither-mcp'), port = 8788, initial }) {
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('port must be an integer from 1024 through 65535.');
  const origin = `http://localhost:${port}`;
  const sandboxOrigin = `http://127.0.0.1:${port}`;
  const host = new URL(origin).host;
  const sandboxHost = new URL(sandboxOrigin).host;
  const token = randomBytes(32).toString('hex');
  const session = await createStudioSession({ binary, root, input, maskInput, initial });
  try {
    const [template, sandbox, bundle] = await Promise.all([
      readFile(path.join(directory, 'index.html'), 'utf8'), readFile(path.join(directory, 'sandbox.html'), 'utf8'),
      build({ entryPoints: [path.join(directory, 'client.mjs')], bundle: true, write: false, format: 'esm', platform: 'browser', target: ['es2022'], logLevel: 'silent' }),
    ]);
    const send = (res, status, type, body, policy = "default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; frame-src " + sandboxOrigin + "; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'") => {
      res.writeHead(status, { 'Content-Type': type, 'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff', 'Referrer-Policy': 'no-referrer', 'Content-Security-Policy': policy });
      res.end(body);
    };
    const server = http.createServer(async (req, res) => {
      try {
        if (![host, sandboxHost].includes(req.headers.host)) return send(res, 403, 'text/plain', 'Host rejected.');
        const url = new URL(req.url, origin);
        if (req.method === 'GET' && req.headers.host === sandboxHost && url.pathname === '/sandbox') return send(res, 200, 'text/html; charset=utf-8', sandbox.replaceAll('HOST_ORIGIN', origin), "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-src 'self'; img-src data:; connect-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors " + origin);
        if (req.headers.host !== host || (req.headers.origin && req.headers.origin !== origin) || ['cross-site', 'same-site'].includes(req.headers['sec-fetch-site'])) return send(res, 403, 'text/plain', 'Origin rejected.');
        if (req.method === 'GET' && url.pathname === '/') return send(res, 200, 'text/html; charset=utf-8', template.replace('SESSION_TOKEN', token));
        if (req.method === 'GET' && url.pathname === '/host.js') return send(res, 200, 'text/javascript', bundle.outputFiles[0].text);
        if (req.headers['x-studio-session'] !== token) return send(res, 403, 'text/plain', 'Session rejected.');
        if (req.method === 'GET' && url.pathname === '/bootstrap') return send(res, 200, 'application/json', JSON.stringify(session.bootstrap));
        if (req.method !== 'POST' || url.pathname !== '/call') return send(res, 404, 'text/plain', 'Not found.');
        if (req.headers.origin !== origin || req.headers['content-type']?.split(';')[0] !== 'application/json') return send(res, 403, 'text/plain', 'Origin or content type rejected.');
        const chunks = []; let bytes = 0;
        for await (const chunk of req) {
          bytes += chunk.length;
          if (bytes > 1 << 20) return send(res, 413, 'text/plain', 'Request too large.');
          chunks.push(chunk);
        }
        const controller = new AbortController();
        res.on('close', () => { if (!res.writableEnded) controller.abort(); });
        const result = await session.callTool(JSON.parse(Buffer.concat(chunks).toString('utf8')), { signal: controller.signal });
        send(res, 200, 'application/json', JSON.stringify(result));
      } catch (error) { if (!res.headersSent) send(res, 400, 'text/plain', error.message); else res.end(); }
    });
    server.requestTimeout = 125_000; server.headersTimeout = 10_000;
    await new Promise((resolve, reject) => { server.once('error', reject); server.listen(port, '127.0.0.1', resolve); });
    return { origin, sandboxOrigin, async close() { server.closeAllConnections(); await new Promise((resolve) => server.close(resolve)); await session.close(); } };
  } catch (error) { await session.close(); throw error; }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const { values } = parseArgs({ options: { root: { type: 'string' }, input: { type: 'string' }, mask: { type: 'string' }, binary: { type: 'string' }, port: { type: 'string' } } });
    if (!values.root || !values.input) throw new Error('Usage: node examples/embedding/server.mjs --root /absolute/workspace --input relative/image.png [--mask mask.png] [--binary /absolute/dither-mcp] [--port 8788]');
    const host = await startStudioHost({ root: values.root, input: values.input, maskInput: values.mask, binary: values.binary, port: values.port === undefined ? 8788 : Number(values.port) });
    console.log(`Dither embedding example: ${host.origin}`);
    console.log('One selected source; the Go MCP server runs over local stdio. Stop with Ctrl+C.');
    const shutdown = () => { void host.close(); };
    process.once('SIGINT', shutdown); process.once('SIGTERM', shutdown);
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
