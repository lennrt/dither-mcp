// Browser integration and local HTTP boundary checks; uses an isolated copy of
// the public artwork and the compiled Go server. No third-party host is implied.
import { chromium } from 'playwright';
import { expect } from 'playwright/test';
import assert from 'node:assert/strict';
import { createServer } from 'node:net';
import { request } from 'node:http';
import { mkdir, mkdtemp, copyFile, readFile, readdir, writeFile, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Client } from '@modelcontextprotocol/client';
import { startStudioHost } from './server.mjs';

const repository = fileURLToPath(new URL('../..', import.meta.url));
const out = path.join(repository, 'work/embedding-browser-check');
await mkdir(out, { recursive: true });
const root = await mkdtemp(path.join(out, 'workspace-'));
await copyFile(path.join(repository, 'docs/assets/source/moon-garden.png'), path.join(root, 'source.png'));
const probe = createServer();
await new Promise((resolve, reject) => { probe.once('error', reject); probe.listen(0, '127.0.0.1', resolve); });
const port = probe.address().port;
await new Promise((resolve) => probe.close(resolve));
let host, browser;
const originalCallTool = Client.prototype.callTool;
const errors = [], external = [], failures = [], checks = [];
try {
  host = await startStudioHost({ root, input: 'source.png', maskInput: 'source.png', port });
  const rejected = await fetch(host.sandboxOrigin + '/bootstrap');
  assert.equal(rejected.status, 403);
  assert.equal((await fetch(host.origin + '/bootstrap')).status, 403);
  assert.equal((await fetch(host.origin, { headers: { origin: 'https://example.invalid' } })).status, 403);
  const hostileHost = await new Promise((resolve, reject) => {
    const req = request({ hostname: '127.0.0.1', port, path: '/', headers: { host: 'example.invalid' } }, (response) => { response.resume(); resolve(response.statusCode); });
    req.once('error', reject); req.end();
  });
  assert.equal(hostileHost, 403);
  checks.push('Exact Host and Origin checks, sandbox denied access to bootstrap, per-session token required');
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });
  page.on('requestfailed', (request) => { if (!request.failure()?.errorText.includes('ERR_ABORTED')) failures.push({ url: request.url(), error: request.failure()?.errorText }); });
  page.on('request', (request) => { if (/^https?:/.test(request.url()) && !['localhost', '127.0.0.1'].includes(new URL(request.url()).hostname)) external.push(request.url()); });
  await page.goto(host.origin);
  await expect(page.locator('#status')).toHaveText('Connected to the local Go server.');
  const studio = page.frames().find((frame) => frame.url() === 'about:srcdoc');
  assert(studio);
  await expect(studio.locator('#status')).toContainText('Preview ready');
  await expect.poll(() => studio.locator('#preview').evaluate((image) => image.complete && image.naturalWidth)).toBe(512);
  assert.deepEqual(await readdir(root), ['source.png']);
  checks.push('Opaque sandbox renders real Go preview through official AppBridge without writing');
  // Hold real Go results inside the Node host while their HTTP calls remain in flight.
  // Deliberately ignore abort during this hold so late completions challenge host identity.
  const held = new Map(); let randomSeed = 100; let delayRandom = true;
  Client.prototype.callTool = async function(params, options) {
    let seed;
    if (delayRandom && params.name === 'dither_random_style') {
      seed = ++randomSeed; params = { ...params, arguments: { seed } };
    } else if (delayRandom && params.name === 'dither_studio') seed = params.arguments.options.seed;
    const result = await originalCallTool.call(this, params, options);
    if (seed !== undefined) await new Promise(resolve => held.set(`${params.name}:${seed}`, { resolve, signal: options.signal }));
    return result;
  };
  await studio.locator('#recipe-output').fill('random.json');
  for (let seed = 101; seed <= 105; seed++) {
    await studio.locator('#random-style').click();
    await expect.poll(() => held.has(`dither_random_style:${seed}`)).toBe(true);
    await expect(studio.locator('#random-style')).toBeEnabled();
    await expect(studio.locator('#save-recipe')).toBeDisabled();
  }
  for (const seed of [104, 102, 101, 103]) {
    assert.equal(held.get(`dither_random_style:${seed}`).signal.aborted, true);
    held.get(`dither_random_style:${seed}`).resolve();
  }
  await expect(studio.locator('#status')).toHaveText('Finding a new look…');
  held.get('dither_random_style:105').resolve();
  await expect.poll(() => held.has('dither_studio:105')).toBe(true);
  await studio.locator('#random-style').click();
  await expect.poll(() => held.has('dither_random_style:106')).toBe(true);
  held.get('dither_random_style:106').resolve();
  await expect.poll(() => held.has('dither_studio:106')).toBe(true);
  assert.equal(held.get('dither_studio:105').signal.aborted, true);
  held.get('dither_studio:105').resolve();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  held.get('dither_studio:106').resolve();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  assert.equal(JSON.parse(await studio.locator('#options-json').inputValue()).seed, 106);
  assert.equal(JSON.parse(await studio.locator('#options-json').inputValue()).mask.shape, 'image');
  assert.deepEqual(await readdir(root), ['source.png']);
  delayRandom = false; Client.prototype.callTool = originalCallTool;
  await studio.locator('#save-recipe').click();
  await expect(studio.locator('#status')).toHaveText('Saved recipe random.json');
  const randomRecipe = JSON.parse(await readFile(path.join(root, 'random.json'), 'utf8'));
  assert.equal(randomRecipe.options.seed, 106); assert.equal(randomRecipe.options.mask.shape, 'image');
  await rm(path.join(root, 'random.json'));
  checks.push('Six repeated random clicks supersede in-flight Go/HTTP style and preview calls; late results cannot restore old settings or recipe authority');
  await studio.locator('#extract-count').fill('5');
  await studio.locator('#extract').click();
  await expect(studio.locator('#status')).toHaveText('Extracted 5 colors. Apply preview before saving.');
  await expect(studio.locator('#save')).toBeDisabled();
  assert.equal((await studio.locator('#custom-colors').inputValue()).split(' ').length, 5);
  assert.deepEqual(await readdir(root), ['source.png']);
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  checks.push('Current-source extraction crosses the restricted host bridge and requires a new preview');
  const token = await page.locator('meta[name="studio-session"]').getAttribute('content');
  const boundary = await Promise.all([
    { name: 'dither_palette_extract', arguments: { input: 'other.png', count: 5 } },
    { name: 'dither_palette_extract', arguments: { input: 'source.png', count: 257 } },
    { name: 'dither_palette_extract', arguments: { input: 'source.png', count: 5, output: 'hidden.png' } },
  ].map(async params => {
    const response = await fetch(host.origin + '/call', { method: 'POST', headers: { origin: host.origin, 'content-type': 'application/json', 'x-studio-session': token }, body: JSON.stringify(params) });
    return response.status;
  }));
  assert.deepEqual(boundary, [400, 400, 400]);
  assert.deepEqual(await readdir(root), ['source.png']);
  checks.push('Extraction rejects other sources, invalid counts, and extra fields at the HTTP boundary');
  await studio.locator('#output').fill('saved.png');
  const preview = await studio.locator('#preview').getAttribute('src');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved saved.png');
  assert.deepEqual(await readFile(path.join(root, 'saved.png')), Buffer.from(preview.split(',')[1], 'base64'));
  checks.push('App-initiated guarded save writes the selected relative output byte-identically');
  await studio.locator('#recipe-output').fill('recipes/masked.json');
  await expect(studio.locator('#recipe-note')).toContainText('mask_input: source.png');
  await studio.locator('#save-recipe').click();
  await expect(studio.locator('#status')).toHaveText('Saved recipe recipes/masked.json');
  const savedRecipe = JSON.parse(await readFile(path.join(root, 'recipes/masked.json'), 'utf8'));
  assert.equal(savedRecipe.options.mask.shape, 'image');
  assert.equal(savedRecipe.colors.length, 5);
  assert.deepEqual(savedRecipe.options, JSON.parse(await studio.locator('#options-json').inputValue()));
  const recipeBoundary = await Promise.all([
    { output: '../escape.json', recipe: savedRecipe },
    { output: 'recipe.png', recipe: savedRecipe },
    { output: 'forged.json', recipe: { ...savedRecipe, options: { ...savedRecipe.options, seed: 99 } } },
    { output: 'extra.json', recipe: savedRecipe, input: 'other.png' },
  ].map(async arguments_ => (await fetch(host.origin + '/call', { method: 'POST', headers: { origin: host.origin, 'content-type': 'application/json', 'x-studio-session': token }, body: JSON.stringify({ name: 'dither_recipe_save', arguments: arguments_ }) })).status));
  assert.deepEqual(recipeBoundary, [400, 400, 400, 400]);
  await studio.locator('#custom-colors').fill('#000 #fff');
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  assert.deepEqual((await readdir(root)).sort(), ['recipes', 'saved.png', 'source.png']);
  checks.push('Masked custom recipe saves exactly; mask reuse reminder, dirty palette gating, and host rejects forged recipes and paths');
  await page.locator('#theme').click();
  await expect(studio.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.screenshot({ path: path.join(out, 'embedding-desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  assert(await studio.locator('html').evaluate((html) => html.scrollWidth <= html.clientWidth + 1));
  await page.screenshot({ path: path.join(out, 'embedding-mobile.png'), fullPage: true });
  checks.push('Host theme update and 390px responsive layout');
  await page.locator('#close').click();
  await expect(page.locator('#status')).toHaveText('Studio closed.');
  assert.equal(await page.locator('#studio').getAttribute('src'), null);
  checks.push('Official teardown handshake closes the view before unmount');
  assert.deepEqual(errors, []); assert.deepEqual(external, []); assert.deepEqual(failures, []);
  await writeFile(path.join(out, 'verification.json'), JSON.stringify({ browser: await browser.version(), checks, errors, externalRequests: external, failedRequests: failures }, null, 2) + '\n');
  console.log(JSON.stringify({ checks, errors, externalRequests: external, failedRequests: failures }, null, 2));
} finally { Client.prototype.callTool = originalCallTool; await browser?.close(); await host?.close(); await rm(root, { recursive: true, force: true }); }
