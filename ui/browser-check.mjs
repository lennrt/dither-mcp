// Optional Chromium regression check. It starts an isolated Go-backed test host.
import { chromium } from "playwright";
import { expect } from "playwright/test";
import { readFile, writeFile, readdir, mkdir } from "node:fs/promises";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { fileURLToPath } from "node:url";
const root = fileURLToPath(new URL("..", import.meta.url));
process.chdir(root);
const out = "work/browser-check";
await mkdir(out, { recursive: true });
const probe = createServer();
await new Promise((resolve, reject) => {
  probe.once("error", reject);
  probe.listen(0, "127.0.0.1", resolve);
});
const port = probe.address().port;
await new Promise((resolve) => probe.close(resolve));
const baseURL = `http://localhost:${port}`;
const siteURL = process.env.DITHER_SITE_TEST_URL;
if (siteURL && !["localhost", "127.0.0.1"].includes(new URL(siteURL).hostname)) throw new Error("Use a loopback URL for the optional showcase check.");
const host = spawn(process.execPath, ["ui/harness.mjs"], {
  cwd: root,
  env: { ...process.env, DITHER_APPS_PORT: String(port) },
  stdio: ["ignore", "pipe", "inherit"]
});
process.once("exit", () => host.kill("SIGTERM"));
const workspace = await new Promise((resolve, reject) => {
  let output = "";
  const timeout = setTimeout(() => {
    host.kill("SIGTERM");
    reject(new Error("Test host startup timed out."));
  }, 2e4);
  host.once("error", (error) => {
    clearTimeout(timeout);
    reject(error);
  });
  host.once("exit", (code) => {
    clearTimeout(timeout);
    reject(new Error(`Test host exited with code ${code}. Run make build first.`));
  });
  host.stdout.on("data", (chunk) => {
    output += chunk.toString();
    const match = output.match(/Sample workspace: ([^\n]+)\n/);
    if (match) {
      clearTimeout(timeout);
      resolve(match[1]);
    }
  });
});
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1100 }, deviceScaleFactor: 1 });
const checks = [];
const record = (message) => {
  checks.push(message);
  console.log(`ok  ${message}`);
};
const errors = [];
const external = [];
const failures = [];
const initialFiles = (await readdir(workspace)).sort();
context.on("page", (page2) => {
  page2.on("pageerror", (e) => errors.push(e.message));
  page2.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text());
  });
  page2.on("requestfailed", (r) => {
    if (!r.failure()?.errorText.includes("ERR_ABORTED")) failures.push({ url: r.url(), error: r.failure() });
  });
  page2.on("request", (r) => {
    if (/^https?:/.test(r.url()) && !["localhost", "127.0.0.1"].includes(new URL(r.url()).hostname)) external.push(r.url());
  });
});
let page;
try {
  page = await context.newPage();
  await page.goto(baseURL);
  await expect(page.locator("#host-status")).toContainText("Connected through");
  const studio = page.frames().find((f) => f.url() === "about:srcdoc");
  assert(studio);
  await expect(studio.locator("#status")).toContainText("Preview ready");
  await expect(studio.locator("#preview")).toBeVisible();
  await expect.poll(() => studio.locator("#preview").evaluate((e) => e.complete && e.naturalWidth)).toBe(512);
  assert.equal(await studio.locator("#algorithm option").count(), 41);
  assert.equal(await studio.locator("#palette option").count(), 256);
  await expect(studio.locator("#save")).toBeDisabled();
  await studio.locator("#output").fill("artwork/garden.png");
  await expect(studio.locator("#save")).toBeEnabled();
  await studio.locator(".studio").screenshot({ path: `${out}/studio-desktop.png` });
  record("Initial preview, complete catalogs, and explicit save gating");
  await studio.locator("#algorithm").selectOption("bayer-8");
  await expect(studio.locator("#save")).toBeDisabled();
  await expect(studio.locator("#preview-badge")).toHaveText("UNAPPLIED");
  await page.locator("#late").click();
  await studio.locator("#palette-search").fill("oat");
  await expect(studio.locator("#algorithm")).toHaveValue("bayer-8");
  assert(await studio.locator("#palette option").count() < 256);
  await expect(studio.locator("#palette")).toHaveValue("oat-and-ink");
  await studio.locator("#palette-search").fill("");
  await studio.locator("#apply").click();
  await expect(studio.locator("#status")).toContainText("Preview ready");
  await expect(studio.locator("#save")).toBeEnabled();
  assert.deepEqual((await readdir(workspace)).sort(), initialFiles);
  record("Dirty settings reject late initial results; filtered palette library and real preview call");
  await studio.getByRole("button", { name: "2\xD7", exact: true }).click();
  assert.equal(await studio.locator("#preview").evaluate((e) => e.style.width), "1024px");
  await studio.getByRole("button", { name: "Fit", exact: true }).click();
  const preview = await studio.locator("#preview").getAttribute("src");
  const destination = `browser-${Date.now()}.png`;
  await studio.locator("#output").fill(destination);
  await studio.locator("#save").click();
  await expect(studio.locator("#status")).toHaveText(`Saved ${destination}`);
  assert.deepEqual(await readFile(`${workspace}/${destination}`), Buffer.from(preview.split(",")[1], "base64"));
  await studio.locator("#save").click();
  await expect(studio.locator("#status")).toContainText(/exist|replace/i);
  assert.deepEqual(await readFile(`${workspace}/${destination}`), Buffer.from(preview.split(",")[1], "base64"));
  record("Zoom, byte-identical PNG save, and overwrite rejection");
  await studio.locator('details.advanced > summary').click();
  await studio.locator('.json-settings > summary').click();
  const editedOptions = JSON.parse(await studio.locator('#options-json').inputValue());
  editedOptions.effects = { noise: 0.1 };
  const optionsJSON = JSON.stringify(editedOptions, null, 2);
  await studio.locator('#options-json').fill(optionsJSON);
  await studio.locator('#extract-count').fill('1');
  await studio.locator('#extract').click();
  await expect(studio.locator('#status')).toContainText('whole number');
  await studio.locator('#extract-count').fill('6');
  await studio.locator('#extract-count').press('Enter');
  await expect(studio.locator('#status')).toHaveText('Extracted 6 colors. Apply preview before saving.');
  await expect(studio.locator('[data-mode="custom"]')).toHaveAttribute('aria-pressed', 'true');
  await expect(studio.locator('#options-json')).toHaveValue(optionsJSON);
  const extractedColors = await studio.locator('#custom-colors').inputValue();
  assert.equal(extractedColors.split(' ').length, 6);
  await expect(studio.locator('#algorithm')).toHaveValue('bayer-8');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  assert.equal(await studio.locator('#preview').getAttribute('src'), preview);
  const extractionCalls = await page.evaluate(() => window.ditherHarness.calls.filter(call => call.name === 'dither_palette_extract'));
  assert.deepEqual(extractionCalls.map(call => call.arguments), [{ input: 'source.png', count: 6 }]);
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  assert.deepEqual(await page.evaluate(() => window.ditherHarness.calls.at(-1).arguments.options.effects), { noise: 0.1 });
  const extractedPreview = await studio.locator('#preview').getAttribute('src');
  await studio.locator('#output').fill('extracted.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved extracted.png');
  assert.deepEqual(await readFile(`${workspace}/extracted.png`), Buffer.from(extractedPreview.split(',')[1], 'base64'));
  record('Source palette extraction validates count, preserves settings, requires preview, and saves exact extracted-color pixels');

  await studio.locator('#recipe-output').fill('recipes/extracted.json');
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  // Recipe dimensions come from the accepted preview, even with invalid custom image export dimensions.
  await studio.locator('#export-size').selectOption('custom');
  await studio.locator('#export-width').fill('0');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  await studio.locator('#recipe-output').press('Enter');
  await expect(studio.locator('#status')).toHaveText('Saved recipe recipes/extracted.json');
  const savedRecipeBytes = await readFile(`${workspace}/recipes/extracted.json`);
  const savedRecipe = JSON.parse(savedRecipeBytes);
  assert.deepEqual(savedRecipe.colors, extractedColors.split(' '));
  assert.deepEqual(savedRecipe.options, JSON.parse(await studio.locator('#options-json').inputValue()));
  assert.deepEqual(savedRecipe.options.effects, { noise: 0.1 });
  const recipeCall = await page.evaluate(() => window.ditherHarness.calls.at(-1));
  assert.equal(recipeCall.name, 'dither_recipe_save');
  assert.deepEqual(recipeCall.arguments, { output: 'recipes/extracted.json', recipe: savedRecipe });
  await studio.locator('#save-recipe').click();
  await expect(studio.locator('#status')).toContainText(/exist|replace/i);
  assert.deepEqual(await readFile(`${workspace}/recipes/extracted.json`), savedRecipeBytes);
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  // Hold a real save result to exercise pending controls and validate the response before success.
  await page.evaluate(() => {
    const host = window.ditherHarness; const original = host.bridge.oncalltool;
    host.bridge.oncalltool = async (params, extra) => {
      const result = await original(params, extra);
      if (params.name === 'dither_recipe_save') {
        await new Promise(resolve => { host.releaseRecipe = resolve; });
        return { ...result, structuredContent: { ...result.structuredContent, path: 'different.json' } };
      }
      return result;
    };
    host.restoreRecipe = () => { host.bridge.oncalltool = original; };
  });
  await studio.locator('#recipe-output').fill('recipes/receipt.json');
  await studio.locator('#save-recipe').click();
  await expect(studio.locator('#preview-badge')).toHaveText('SAVING RECIPE');
  await expect(studio.locator('#random-style')).toBeDisabled();
  await expect(studio.locator('#algorithm')).toBeDisabled();
  await expect(studio.locator('#recipe-output')).toBeDisabled();
  await expect(studio.locator('#output')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  await expect.poll(() => page.evaluate(() => !!window.ditherHarness.releaseRecipe)).toBe(true);
  await page.evaluate(() => { window.ditherHarness.releaseRecipe(); window.ditherHarness.restoreRecipe(); });
  await expect(studio.locator('#status')).toContainText('different recipe save receipt');
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  await studio.locator('#recipe-output').fill('recipes/retry.json');
  await studio.locator('#save-recipe').click();
  await expect(studio.locator('#status')).toHaveText('Saved recipe recipes/retry.json');
  await studio.locator('#export-size').selectOption('preview');
  record('Recipe save keeps accepted dimensions, extracted colors and effects; rejects overwrite and mismatched receipt; freezes pending settings and permits retry');

  const beforeMixed = JSON.parse(await studio.locator('#options-json').inputValue());
  const mixed = { ...beforeMixed, brightness: -0.1, algorithm: 'atkinson', serpentine: true, contrast: 1.5, crop: { x: 2, y: 3, width: 120, height: 90 }, effects: { noise: 0.2, scanlines: 0.1 } };
  await studio.locator('#options-json').fill(JSON.stringify(mixed));
  await studio.locator('#brightness').fill('0.35');
  await expect(studio.locator('#algorithm')).toHaveValue('atkinson');
  await expect(studio.locator('#contrast')).toHaveValue('1.5');
  await expect(studio.locator('[data-option="serpentine"]')).toBeChecked();
  const afterSlider = JSON.parse(await studio.locator('#options-json').inputValue());
  assert.deepEqual(afterSlider, { ...mixed, brightness: 0.35 });
  await studio.locator('#options-json').fill(JSON.stringify({ ...afterSlider, gamma: 1.2 }));
  await studio.locator('[data-option="serpentine"]').uncheck();
  const afterCheckbox = JSON.parse(await studio.locator('#options-json').inputValue());
  assert.equal(afterCheckbox.serpentine, false); assert.equal(afterCheckbox.gamma, 1.2);
  await studio.locator('#options-json').fill(JSON.stringify({ ...afterCheckbox, saturation: 0.8 }));
  await studio.locator('#algorithm').selectOption('bayer-8');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  const mergedCall = await page.evaluate(() => window.ditherHarness.calls.at(-1));
  assert.equal(mergedCall.name, 'dither_studio');
  assert.deepEqual(mergedCall.arguments.options, { ...mixed, brightness: 0.35, serpentine: false, gamma: 1.2, algorithm: 'bayer-8', saturation: 0.8 });
  record('Advanced JSON merges later slider, checkbox, and select edits while preserving crop and effects in the real preview request');

  const callsBeforeInvalidJSON = await page.evaluate(() => window.ditherHarness.calls.length);
  for (const [text, control] of [['{', 'slider'], ['null', 'select'], ['[]', 'checkbox']]) {
    await studio.locator('#options-json').fill(text);
    if (control === 'slider') await studio.locator('#brightness').fill('0.55');
    if (control === 'select') await studio.locator('#algorithm').selectOption('atkinson');
    if (control === 'checkbox') await studio.locator('[data-option="serpentine"]').click();
    await expect(studio.locator('#status')).toContainText('Correct the advanced options JSON');
    await expect(studio.locator('#options-json')).toHaveValue(text);
    await expect(studio.locator('#brightness')).toHaveValue('0.35');
    await expect(studio.locator('#algorithm')).toHaveValue('bayer-8');
    await expect(studio.locator('[data-option="serpentine"]')).not.toBeChecked();
    await expect(studio.locator('#save')).toBeDisabled();
    await expect(studio.locator('#save-recipe')).toBeDisabled();
  }
  await studio.locator('#options-json').fill('null');
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Options must be a JSON object');
  await expect(studio.locator('#options-json')).toHaveValue('null');
  await expect(studio.locator('#brightness')).toHaveValue('0.35');
  await expect(studio.locator('#algorithm')).toHaveValue('bayer-8');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  assert.equal(await page.evaluate(() => window.ditherHarness.calls.length), callsBeforeInvalidJSON);
  await studio.locator('#options-json').fill(JSON.stringify(beforeMixed));
  // The first numeric edit must merge JSON without resetting a cleared field.
  await studio.locator('[data-option="seed"]').fill('');
  await expect(studio.locator('[data-option="seed"]')).toHaveValue('');
  await studio.locator('[data-option="seed"]').pressSequentially('-17');
  await expect(studio.locator('[data-option="seed"]')).toHaveValue('-17');
  assert.equal(JSON.parse(await studio.locator('#options-json').inputValue()).seed, -17);
  // Select-and-type makes '-' itself the first input event while JSON is dirty.
  await studio.locator('#options-json').fill(JSON.stringify(beforeMixed));
  await studio.locator('[data-option="seed"]').focus();
  await studio.locator('[data-option="seed"]').press('ControlOrMeta+A');
  await studio.locator('[data-option="seed"]').pressSequentially('-17');
  await expect(studio.locator('[data-option="seed"]')).toHaveValue('-17');
  assert.equal(JSON.parse(await studio.locator('#options-json').inputValue()).seed, -17);
  await studio.locator('[data-option="seed"]').fill(String(beforeMixed.seed ?? 0));
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  assert.equal((await page.evaluate(() => window.ditherHarness.calls.at(-1).arguments.options)).crop, undefined);
  record('Invalid advanced JSON remains visible, restores controls without tool calls, and recovers after correction');


  // Delay completed real tool results to exercise edits and cancellation races.
  await page.evaluate(() => {
    const harness = window.ditherHarness;
    const original = harness.bridge.oncalltool;
    harness.originalCallTool = original;
    harness.bridge.oncalltool = async (params, extra) => {
      const result = await original(params, extra);
      if (params.name === 'dither_palette_extract') {
        harness.delayedExtraction = { id: extra.mcpReq.id, result };
        await new Promise(resolve => { harness.releaseExtraction = resolve; });
      }
      return result;
    };
  });
  for (const change of ['tone', 'count', 'cancel']) {
    await page.evaluate(() => { window.ditherHarness.releaseExtraction = null; });
    await studio.locator('#extract').click();
    await page.waitForFunction(() => !!window.ditherHarness.releaseExtraction);
    await expect(studio.locator('#apply')).toBeDisabled();
    await expect(studio.locator('#extract')).toBeDisabled();
    await expect(studio.locator('#save')).toBeDisabled();
    await expect(studio.locator('#save-recipe')).toBeDisabled();
    if (change === 'tone') await studio.locator('#brightness').fill('0.2');
    if (change === 'count') await studio.locator('#extract-count').fill('7');
    if (change === 'cancel') await studio.locator('#cancel').click();
    await page.evaluate(() => window.ditherHarness.releaseExtraction());
    if (change === 'cancel') {
      // Reproduce a completed reply arriving after its local cancellation,
      // independent of whether the host receives the cancellation first.
      await page.evaluate(() => {
        const { bridge, delayedExtraction } = window.ditherHarness;
        return bridge.transport.send({ jsonrpc: '2.0', ...delayedExtraction });
      });
    }
    await expect(studio.locator('#status')).toContainText(change === 'cancel' ? 'extraction cancelled' : 'Settings changed while extracting');
    await expect(studio.locator('#custom-colors')).toHaveValue(extractedColors);
    await expect(studio.locator('#save')).toBeDisabled();
    await expect(studio.locator('#save-recipe')).toBeDisabled();
    await expect(studio.locator('#apply')).toBeEnabled();
    await expect(studio.locator('#host-note')).toBeHidden();
  }
  await page.evaluate(() => { window.ditherHarness.bridge.oncalltool = window.ditherHarness.originalCallTool; });
  record('Pending extraction gates preview/save; edited controls, changed count, and cancellation discard delayed results without losing the host connection');

  const filesBeforeRandom = (await readdir(workspace)).sort();
  const callsBeforeRandom = await page.evaluate(() => window.ditherHarness.calls.length);
  for (const invalid of ['{', 'null', '[]']) {
    await studio.locator('#options-json').fill(invalid);
    await studio.locator('#random-style').click();
    await expect(studio.locator('#status')).toContainText(invalid === '{' ? 'Correct the advanced options JSON' : 'Options must be a JSON object');
    await expect(studio.locator('#options-json')).toHaveValue(invalid);
  }
  assert.equal(await page.evaluate(() => window.ditherHarness.calls.length), callsBeforeRandom);
  const randomBase = { ...beforeMixed, crop: { x: 2, y: 3, width: 120, height: 90 }, mask: { shape: 'rectangle', x: 4, y: 6, width: 40, height: 30, invert: true }, pixel_scale: 2, resize_filter: 'nearest', effects: { noise: .4, pixel_sort: true } };
  await studio.locator('#options-json').fill(JSON.stringify(randomBase));
  await studio.locator('#custom-colors').fill('replace this unfinished palette');
  await page.evaluate(() => {
    const host = window.ditherHarness; const original = host.bridge.oncalltool;
    host.randomOriginal = original;
    host.bridge.oncalltool = async (params, extra) => {
      const result = await original(params, extra);
      if (params.name === 'dither_random_style') host.randomResult = result;
      return result;
    };
  });
  await studio.locator('#random-style').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  const randomCalls = await page.evaluate(start => window.ditherHarness.calls.slice(start), callsBeforeRandom);
  assert.deepEqual(randomCalls.map(call => call.name), ['dither_random_style', 'dither_studio']);
  const generated = await page.evaluate(() => window.ditherHarness.randomResult.structuredContent);
  const randomOptions = JSON.parse(await studio.locator('#options-json').inputValue());
  for (const key of ['width', 'height', 'crop', 'mask', 'pixel_scale', 'resize_filter']) assert.deepEqual(randomOptions[key], randomBase[key]);
  assert.deepEqual(randomOptions.effects, generated.recipe.options.effects);
  assert.equal(randomOptions.seed ?? 0, generated.seed);
  await expect(studio.locator('[data-mode="builtin"]')).toHaveAttribute('aria-pressed', 'true');
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  assert.deepEqual((await readdir(workspace)).sort(), filesBeforeRandom);
  // Editing or canceling a delayed style cannot start its automatic preview.
  await page.evaluate(() => {
    const host = window.ditherHarness;
    host.bridge.oncalltool = async (params, extra) => {
      const result = await host.randomOriginal(params, extra);
      if (params.name === 'dither_random_style') await new Promise(resolve => { host.releaseRandom = resolve; });
      return result;
    };
  });
  for (const change of ['edit', 'cancel']) {
    await page.evaluate(() => { window.ditherHarness.releaseRandom = null; });
    const start = await page.evaluate(() => window.ditherHarness.calls.length);
    await studio.locator('#random-style').click();
    await page.waitForFunction(() => !!window.ditherHarness.releaseRandom);
    await expect(studio.locator('#random-style')).toBeEnabled();
    await expect(studio.locator('#cancel')).toBeVisible();
    if (change === 'edit') await studio.locator('#brightness').fill('0.45');
    else await studio.locator('#cancel').click();
    await page.evaluate(() => window.ditherHarness.releaseRandom());
    await expect(studio.locator('#status')).toContainText(change === 'edit' ? 'Settings changed while choosing' : 'Random style cancelled');
    assert.deepEqual(await page.evaluate(start => window.ditherHarness.calls.slice(start).map(call => call.name), start), ['dither_random_style']);
    await expect(studio.locator('#save-recipe')).toBeDisabled();
  }
  await page.evaluate(() => { window.ditherHarness.bridge.oncalltool = window.ditherHarness.randomOriginal; });
  await studio.locator('#options-json').fill(JSON.stringify(beforeMixed));
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  record('Prominent random action preserves geometry, replaces unfinished custom colors and all style effects, validates pending JSON, and discards edited or cancelled generation before auto-preview');

  await studio.getByRole("button", { name: "Custom colors", exact: true }).click();
  await studio.locator("#custom-colors").fill("#123 #123");
  await studio.locator("#apply").click();
  await expect(studio.locator("#status")).toContainText("unique");
  await expect(studio.locator("#save")).toBeDisabled();
  await studio.locator("#custom-colors").fill("#141413 #3d3d3a #d97757 #e3dacc #f0eee6 #faf9f5");
  await studio.locator("#apply").click();
  await expect(studio.locator("#status")).toContainText("Preview ready");
  record("Custom-color validation and real custom-palette preview");
  await studio.locator("#output").fill("artwork/garden.png");
  await page.locator("#theme").click();
  await expect(studio.locator("html")).toHaveAttribute("data-theme", "dark");
  await studio.locator(".studio").screenshot({ path: `${out}/studio-dark.png` });
  await page.locator("#width").click();
  await expect.poll(() => studio.locator("html").evaluate((e) => e.clientWidth)).toBeLessThanOrEqual(370);
  assert(await studio.locator("html").evaluate((e) => e.scrollWidth <= e.clientWidth + 1));
  await studio.locator(".studio").screenshot({ path: `${out}/studio-narrow-dark.png` });
  await page.locator("#theme").click();
  await page.locator("#width").click();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect.poll(() => studio.locator("html").evaluate((e) => e.clientWidth)).toBeLessThanOrEqual(390);
  assert(await studio.locator("html").evaluate((e) => e.scrollWidth <= e.clientWidth + 1));
  await studio.locator(".studio").screenshot({ path: `${out}/studio-mobile.png` });
  await studio.locator('#export-size').selectOption('custom');
  await studio.locator('#export-width').fill('240');
  await studio.locator('#export-height').fill('160');
  await expect(studio.locator('#save')).toBeEnabled();
  assert(await studio.locator('html').evaluate(e => e.scrollWidth <= e.clientWidth + 1));
  await studio.locator('.studio').screenshot({ path: `${out}/studio-export-mobile.png` });
  await studio.locator('#export-size').selectOption('preview');
  record("Host theme changes, 370px host and 390px viewport without horizontal overflow");
  await page.setViewportSize({ width: 1440, height: 1100 });
  await page.locator("#cancel").click();
  await expect(studio.locator("#status")).toContainText("host cancelled");
  await expect(studio.locator("#save")).toBeDisabled();
  await page.locator("#late").click();
  await studio.locator("#output").focus();
  await expect(studio.locator("#save")).toBeDisabled();
  await studio.locator("#apply").click();
  await expect(studio.locator("#status")).toContainText("Preview ready");
  record("Host cancellation invalidates saving and a fresh preview recovers");
  const keyboardDestination = `keyboard-${Date.now()}.png`;
  await studio.locator("#output").fill(keyboardDestination);
  const keyboardPreview = await studio.locator("#preview").getAttribute("src");
  await studio.locator("#output").press("Enter");
  await expect(studio.locator("#status")).toHaveText(`Saved ${keyboardDestination}`);
  assert.deepEqual(await readFile(`${workspace}/${keyboardDestination}`), Buffer.from(keyboardPreview.split(",")[1], "base64"));
  record("Keyboard save works without sandbox form permission");
  const sourceBytes = await readFile(`${workspace}/source.png`);
  await studio.locator('#export-size').selectOption('source');
  await expect(studio.locator('#export-note')).toContainText('Changing resolution');
  await studio.locator('#output').fill('source-size.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved source-size.png');
  const fullPNG = await readFile(`${workspace}/source-size.png`);
  assert.equal(fullPNG.readUInt32BE(16), sourceBytes.readUInt32BE(16));
  assert.equal(fullPNG.readUInt32BE(20), sourceBytes.readUInt32BE(20));
  await studio.locator('#export-size').selectOption('custom');
  await studio.locator('#export-width').fill('');
  await studio.locator('#export-height').fill('');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  await studio.locator('#export-width').fill('4097');
  await studio.locator('#export-height').fill('4096');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeEnabled();
  await expect(studio.locator('#export-note')).toContainText('Export must fit');
  await studio.locator('#export-width').fill('240');
  await studio.locator('#export-height').fill('160');
  await studio.locator('#output').fill('custom-size.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved custom-size.png');
  const customPNG = await readFile(`${workspace}/custom-size.png`);
  assert.equal(customPNG.readUInt32BE(16), 240); assert.equal(customPNG.readUInt32BE(20), 160);
  await expect.poll(() => studio.locator('#preview').evaluate(e => e.naturalWidth)).toBe(512);
  record('Source-size and custom exports, pixel-budget validation, and unchanged preview size');
  await studio.locator('#export-size').selectOption('preview');
  await writeFile(`${workspace}/source.png`, customPNG);
  await studio.locator('#output').fill('stale.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toContainText('source image changed');
  await expect(studio.locator('#status')).toContainText('Apply preview again');
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  assert(!(await readdir(workspace)).includes('stale.png'));
  await studio.locator('#apply').click();
  await expect(studio.locator('#status')).toContainText('Preview ready');
  await expect(studio.locator('#save')).toBeEnabled();
  await studio.locator('#output').fill('fresh.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved fresh.png');
  record('Changed source creates no artifact, invalidates save, and requires a fresh preview');
  await studio.locator('#export-size').selectOption('custom');
  await studio.locator('#export-width').fill('1');
  await studio.locator('#export-height').fill('1');
  await studio.locator('#output').fill('solid.png');
  await studio.locator('#save').click();
  await expect(studio.locator('#status')).toHaveText('Saved solid.png');
  await writeFile(`${workspace}/source.png`, await readFile(`${workspace}/solid.png`));
  const beforeMono = await studio.locator('#custom-colors').inputValue();
  await studio.locator('#extract').click();
  await expect(studio.locator('#status')).toContainText('fewer than two visible colors');
  await expect(studio.locator('#custom-colors')).toHaveValue(beforeMono);
  await expect(studio.locator('#save')).toBeDisabled();
  await expect(studio.locator('#save-recipe')).toBeDisabled();
  await expect(studio.locator('#apply')).toBeEnabled();
  record('Monochrome source extraction reports the tool error and keeps the current custom palette');

  const noTools = await context.newPage();
  await noTools.goto(baseURL + "/?no-tools");
  await expect(noTools.locator("#host-status")).toContainText("Connected through");
  const limited = noTools.frames().find((f) => f.url() === "about:srcdoc");
  await expect(limited.locator("#preview")).toBeVisible();
  await expect(limited.locator("#host-note")).toContainText("does not support app tool calls");
  await expect(limited.locator("#apply")).toBeDisabled();
  await expect(limited.locator("#extract")).toBeDisabled();
  await expect(limited.locator("#random-style")).toBeDisabled();
  await limited.locator("#output").fill("no-tools.png");
  await expect(limited.locator("#save")).toBeDisabled();
  await limited.locator("#recipe-output").fill("no-tools.json");
  await expect(limited.locator("#save-recipe")).toBeDisabled();
  await noTools.close();
  record("Host without serverTools displays result and disables mutations");
  if (siteURL) {
    const site = await context.newPage();
    await site.goto(siteURL);
    await site.locator("#studio").scrollIntoViewIfNeeded();
    await expect.poll(() => site.locator("#studio img").evaluate((e) => e.complete && e.naturalWidth > 0)).toBe(true);
    assert(await site.locator("html").evaluate((e) => e.scrollWidth <= e.clientWidth + 1));
    await site.locator("#studio").screenshot({ path: `${out}/site-studio-desktop.png` });
    await site.getByRole("link", { name: "Explore the studio" }).click();
    await expect(site).toHaveURL(/#studio$/);
    await site.setViewportSize({ width: 390, height: 844 });
    await site.locator("#studio").scrollIntoViewIfNeeded();
    assert(await site.locator("html").evaluate((e) => e.scrollWidth <= e.clientWidth + 1));
    await site.screenshot({ path: `${out}/site-mobile.png`, fullPage: true });
    await site.locator("#studio").screenshot({ path: `${out}/site-studio-mobile.png` });
    for (const viewport of [{ width: 1440, height: 1100 }, { width: 390, height: 844 }]) {
      await site.setViewportSize(viewport);
      await site.locator("footer").scrollIntoViewIfNeeded();
      for (const image of await site.locator("img[src]").all()) {
        if (await image.isVisible()) {
          await image.scrollIntoViewIfNeeded();
          await expect.poll(() => image.evaluate((e) => e.complete && e.naturalWidth > 0)).toBe(true);
        }
      }
    }
    record("Showcase studio navigation, all visible images loaded, desktop/mobile layout without overflow");
  }
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  assert.deepEqual(failures, []);
  await writeFile(`${out}/verification.json`, JSON.stringify({ date: (new Date()).toISOString().slice(0, 10), browser: await browser.version(), checks, errors, externalRequests: external, failedRequests: failures, screenshots: ["studio-desktop.png", "studio-dark.png", "studio-narrow-dark.png", "studio-mobile.png", "studio-export-mobile.png", ...siteURL ? ["site-studio-desktop.png", "site-studio-mobile.png", "site-mobile.png"] : []] }, null, 2) + "\n");
  console.log(JSON.stringify({ passed: checks, errors, externalRequests: external, browser: await browser.version() }, null, 2));
} catch (error) {
  console.error(error);
  if (page) await page.screenshot({ path: `${out}/failure.png`, fullPage: true });
  console.error({ checks, errors, external, failures });
  process.exitCode = 1;
} finally {
  await browser.close();
  if (host.exitCode === null && host.signalCode === null) {
    const stopped = new Promise((resolve) => host.once("exit", resolve));
    host.kill("SIGTERM");
    const timeout = setTimeout(() => host.kill("SIGKILL"), 5000);
    timeout.unref();
    await stopped;
    clearTimeout(timeout);
  }
}
