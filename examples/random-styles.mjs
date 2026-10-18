#!/usr/bin/env node
// Generate, render, and save three repeatable styles through the public CLI.
// Run from your image workspace after building bin/dither-mcp in this checkout.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const [input, directory, ...requestedSeeds] = process.argv.slice(2);
if (!input || !directory) {
  console.error('Usage: node examples/random-styles.mjs <relative image> <new output directory> [integer seeds...]');
  process.exit(1);
}
const seeds = requestedSeeds.length ? requestedSeeds.map(Number) : [42, 314, 2026];
if (seeds.some(seed => !Number.isSafeInteger(seed))) {
  console.error('Each seed must be a JavaScript-safe integer.');
  process.exit(1);
}
const binary = fileURLToPath(new URL(`../bin/dither-mcp${process.platform === 'win32' ? '.exe' : ''}`, import.meta.url));
function call(tool, args) {
  const result = spawnSync(binary, ['call', '--root', process.cwd(), tool, JSON.stringify(args)], {
    encoding: 'utf8', maxBuffer: 4 * 1024 * 1024,
  });
  if (result.error || result.status !== 0) {
    throw new Error(`${tool}: ${result.error?.message ?? result.stderr.trim()}`);
  }
  return JSON.parse(result.stdout);
}
try {
  for (const seed of seeds) {
    const style = call('dither_random_style', { seed });
    // Generation omits geometry. Add a fixed demo size to the reusable recipe.
    const recipe = { ...style.recipe, options: { ...style.recipe.options, width: 320, pixel_scale: 2 } };
    const stem = path.posix.join(directory, `seed-${seed}`);
    const render = call('dither_render', { input, output: `${stem}.png`, palette: recipe.palette, options: recipe.options });
    const saved = call('dither_recipe_save', { output: `${stem}.json`, recipe: render.recipe });
    console.log(JSON.stringify({ seed, palette: recipe.palette, algorithm: recipe.options.algorithm,
      image: render, recipe: saved }, null, 2));
  }
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
