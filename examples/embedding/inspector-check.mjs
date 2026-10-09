// Optional independent host check. Install the published Inspector separately,
// then pass its package directory. This script never installs software itself.
import { chromium } from 'playwright';
import { expect } from 'playwright/test';
import assert from 'node:assert/strict';
import { parseArgs } from 'node:util';
import { spawn, execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { randomBytes } from 'node:crypto';
import { createServer } from 'node:net';
import { mkdir, mkdtemp, copyFile, readFile, readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const { values } = parseArgs({ options: { inspector: { type: 'string' }, out: { type: 'string' } } });
if (!values.inspector) throw new Error('Pass --inspector /absolute/path/to/node_modules/@modelcontextprotocol/inspector.');
const repository = fileURLToPath(new URL('../..', import.meta.url));
const packageRoot = path.resolve(values.inspector);
const launcher = path.join(packageRoot, 'clients/launcher/build/index.js');
const inspector = JSON.parse(await readFile(path.join(packageRoot, 'package.json'), 'utf8'));
assert.equal(inspector.name, '@modelcontextprotocol/inspector');
const out = path.resolve(values.out ?? path.join(repository, 'work/inspector-check'));
await mkdir(out, { recursive: true });
const run = await mkdtemp(path.join(out, 'run-'));
const root = path.join(run, 'images');
await mkdir(root);
await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
const config = path.join(run, 'config.json');
await writeFile(config, JSON.stringify({ mcpServers: { dither: { type: 'stdio', command: path.join(repository, 'bin', process.platform === 'win32' ? 'dither-mcp.exe' : 'dither-mcp'), args: ['mcp', '--root', root] } } }));
const env = { ...process.env, MCP_INSPECTOR_SECRET_STORE: 'memory', MCP_AUTO_OPEN_ENABLED: 'false', MCP_STORAGE_DIR: path.join(run, 'state'), MCP_CLIENT_CONFIG_PATH: path.join(run, 'state/client.json'), MCP_INSPECTOR_OAUTH_STATE_PATH: path.join(run, 'state/oauth.json') };
delete env.MCP_CATALOG_PATH;
const probe = await promisify(execFile)(process.execPath, [launcher, '--cli', '--config', config, '--server', 'dither', '--method', 'tools/call', '--tool-name', 'dither_studio', '--advertise-apps', '--app-info'], { env });
const appInfo = JSON.parse(probe.stdout.trim());
assert.equal(appInfo.hasApp, true); assert.equal(appInfo.resourceUri, 'ui://dither/studio.html');
assert.equal(appInfo.resourceMimeType, 'text/html;profile=mcp-app');
async function freePort() {
  const listener = createServer();
  await new Promise((resolve, reject) => { listener.once('error', reject); listener.listen(0, '127.0.0.1', resolve); });
  const port = listener.address().port;
  await new Promise((resolve) => listener.close(resolve));
  return port;
}
const port = await freePort(), sandboxPort = await freePort(), appPort = await freePort();
const token = randomBytes(32).toString('hex');
Object.assign(env, { CLIENT_PORT: String(port), MCP_SANDBOX_PORT: String(sandboxPort), MCP_APP_ORIGIN_PORT: String(appPort), MCP_INSPECTOR_API_TOKEN: token });
const host = spawn(process.execPath, [launcher, '--web', '--config', config], { env, stdio: ['ignore', 'pipe', 'pipe'] });
const ready = new Promise((resolve, reject) => {
  let output = '';
  const timer = setTimeout(() => reject(new Error('Inspector did not start within 20 seconds.')), 20_000);
  host.stdout.on('data', (chunk) => { output += chunk.toString(); if (output.includes('Web is up and running')) { clearTimeout(timer); resolve(); } });
  host.once('exit', (code) => { clearTimeout(timer); reject(new Error(`Inspector exited with status ${code}.`)); });
});
let browser;
const errors = [], external = [], blockedHostRequests = [], checks = ['Official Inspector CLI negotiates Apps and reads the studio resource over Go stdio'];
try {
  await ready;
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
  await page.route('**/*', async (route) => {
    const url = new URL(route.request().url());
    if (!['http:', 'https:'].includes(url.protocol) || ['localhost', '127.0.0.1'].includes(url.hostname)) return route.continue();
    if (route.request().frame() === page.mainFrame() && ['fonts.googleapis.com', 'fonts.gstatic.com'].includes(url.hostname)) {
      blockedHostRequests.push(url.hostname);
      return route.fulfill({ status: 200, contentType: 'text/css', body: '' });
    }
    external.push(url.origin);
    return route.fulfill({ status: 403, body: 'External requests are blocked by this integration test.' });
  });
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });
  await page.goto(`http://127.0.0.1:${port}/?MCP_INSPECTOR_API_TOKEN=${token}`);
  const connect = page.getByRole('switch', { name: 'Connect or disconnect "dither"' });
  await connect.focus(); await connect.press('Space');
  await page.getByText('Apps', { exact: true }).click();
  await page.getByText('dither_studio', { exact: true }).click();
  const editorSwitch = page.getByRole('switch', { name: 'Edit as JSON' });
  await editorSwitch.focus(); await editorSwitch.press('Space');
  const editor = page.getByRole('textbox', { name: /^Arguments JSON/ });
  await editor.focus(); await editor.press('ControlOrMeta+A');
  await page.keyboard.insertText(JSON.stringify({ input: 'source.png', palette: 'oat-and-ink', options: { algorithm: 'atkinson', width: 512 } }));
  await page.getByTestId('open-app').click();
  await page.locator('[data-app-status="ready"]').waitFor();
  const studio = page.frames().find((frame) => frame.url() === 'about:srcdoc');
  assert(studio);
  await expect(studio.locator('#status')).toContainText('Preview ready');
  await expect.poll(() => studio.locator('#preview').evaluate((image) => image.complete && image.naturalWidth)).toBe(512);
  assert.equal(await studio.locator('#algorithm option').count(), 41);
  assert.equal(await studio.locator('#palette option').count(), 256);
  assert.deepEqual(await readdir(root), ['source.png']);
  checks.push('Published web Apps tab renders initial PNG and complete catalogs without writing files');
  await studio.locator('#algorithm').selectOption('bayer-8');
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  const preview = await studio.locator('#preview').getAttribute('src');
  await studio.locator('#output').fill('preview.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved preview.png');
  assert.deepEqual(await readFile(path.join(root, 'preview.png')), Buffer.from(preview.split(',')[1], 'base64'));
  checks.push('App-initiated studio preview and guarded PNG save work through Inspector serverTools');
  await studio.locator('#export-size').selectOption('custom');
  await studio.locator('#export-width').fill('1200'); await studio.locator('#export-height').fill('800');
  await studio.locator('#output').fill('large.png'); await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved large.png');
  const png = await readFile(path.join(root, 'large.png'));
  assert.deepEqual([png.readUInt32BE(16), png.readUInt32BE(20)], [1200, 800]);
  checks.push('Custom 1200 by 800 export is independent of the 512px preview');
  await page.setViewportSize({ width: 1800, height: 2200 });
  await page.getByRole('button', { name: 'Maximize', exact: true }).click();
  await expect.poll(() => studio.locator('.studio').evaluate((element) => element.clientWidth)).toBeGreaterThan(800);
  await studio.locator('.studio').screenshot({ path: path.join(out, 'studio-inspector.png') });
  await copyFile(path.join(repository, 'docs/assets/source/mineral-nocturne.png'), path.join(root, 'source.png'));
  await studio.locator('#output').fill('changed.png'); await studio.locator('#save').click();
  await expect(studio.locator('#status')).toContainText(/changed|Apply preview again/i);
  await expect(studio.locator('#save')).toBeDisabled();
  assert.deepEqual((await readdir(root)).sort(), ['large.png', 'preview.png', 'source.png']);
  checks.push('Changed source invalidates Save and leaves no stale export');
  assert.deepEqual(errors, []); assert.deepEqual(external, []);
  const report = { date: new Date().toISOString().slice(0, 10), host: inspector.name, version: inspector.version, transport: 'stdio', node: process.version, browser: await browser.version(), source: 'docs/assets/source/moon-garden.png (isolated copy)', appInfo, checks, errors, appExternalRequests: external, blockedHostRequests: [...new Set(blockedHostRequests)], screenshot: 'studio-inspector.png', limits: 'Inspector shell Google Fonts requests were fulfilled locally with empty CSS; no external request was allowed. Only the published official Inspector and this repository embedding example were executed. Other host entries are documentation-backed, not locally tested.' };
  await writeFile(path.join(out, 'host-verification.json'), JSON.stringify(report, null, 2) + '\n');
  console.log(JSON.stringify(report, null, 2));
} finally {
  await browser?.close();
  if (host.exitCode === null && host.signalCode === null) {
    const stopped = new Promise((resolve) => host.once('exit', resolve)); host.kill('SIGTERM');
    const timeout = setTimeout(() => host.kill('SIGKILL'), 5000); timeout.unref();
    await stopped; clearTimeout(timeout);
  }
}
