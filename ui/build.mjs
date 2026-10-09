import { build } from 'esbuild';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const directory = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(directory, '..');
const target = path.resolve(directory, '../internal/mcpapp/studio.html');
const mappingTarget = path.join(directory, 'bundle-meta.json');
const [template, css, result] = await Promise.all([
  readFile(path.join(directory, 'index.html'), 'utf8'), readFile(path.join(directory, 'style.css'), 'utf8'),
  build({ absWorkingDir: root, entryPoints: ['ui/app.mjs'], bundle: true, write: false, metafile: true, minify: true, format: 'iife', platform: 'browser', target: ['es2022'], legalComments: 'inline', charset: 'utf8', logLevel: 'silent' }),
]);
const script = result.outputFiles[0].text.replace(/<\/script/gi, '<\\/script');
// Function replacements preserve dollar sequences in bundled JavaScript.
const html = template.replace('<!-- UI_CSS -->', () => `<style>${css.trim()}</style>`).replace('<!-- UI_JS -->', () => `<script>${script.trim()}</script>`);
const mapping = JSON.stringify(result.metafile, null, 2) + '\n';
if (process.argv.includes('--check')) {
  const [checked, checkedMapping] = await Promise.all([readFile(target, 'utf8').catch(() => ''), readFile(mappingTarget, 'utf8').catch(() => '')]);
  if (checked !== html || checkedMapping !== mapping) { console.error('Embedded studio HTML or bundle mapping is stale. Run pnpm ui:build.'); process.exitCode = 1; }
  else console.log('Embedded studio HTML and bundle mapping match the pinned UI build.');
} else { await mkdir(path.dirname(target), { recursive: true }); await Promise.all([writeFile(target, html), writeFile(mappingTarget, mapping)]); console.log(`Built internal/mcpapp/studio.html (${Buffer.byteLength(html)} bytes) and ui/bundle-meta.json.`); }
