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
  const noTools = await context.newPage();
  await noTools.goto(baseURL + "/?no-tools");
  await expect(noTools.locator("#host-status")).toContainText("Connected through");
  const limited = noTools.frames().find((f) => f.url() === "about:srcdoc");
  await expect(limited.locator("#preview")).toBeVisible();
  await expect(limited.locator("#host-note")).toContainText("does not support app tool calls");
  await expect(limited.locator("#apply")).toBeDisabled();
  await limited.locator("#output").fill("no-tools.png");
  await expect(limited.locator("#save")).toBeDisabled();
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
  await writeFile(`${out}/verification.json`, JSON.stringify({ date: (new Date()).toISOString().slice(0, 10), browser: await browser.version(), checks, errors, externalRequests: external, failedRequests: failures, screenshots: ["studio-desktop.png", "studio-dark.png", "studio-narrow-dark.png", "studio-mobile.png", ...siteURL ? ["site-studio-desktop.png", "site-studio-mobile.png", "site-mobile.png"] : []] }, null, 2) + "\n");
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
