#!/usr/bin/env node
// These development checks run no lifecycle scripts or model-checker downloads.
import { spawnSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import os from 'node:os';
const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const tool = {
  quint: path.join(root, 'node_modules/@informalsystems/quint/dist/src/cli.js'),
  openspec: path.join(root, 'node_modules/@fission-ai/openspec/bin/openspec.js'),
};
const env = { ...process.env, OPENSPEC_TELEMETRY: '0', DO_NOT_TRACK: '1' };
const records = [];
function run(name, args, expected = 0) {
  const started = Date.now();
  const result = spawnSync(process.execPath, [tool[name], ...args], {
    cwd: root, env, encoding: 'utf8', timeout: 120_000, maxBuffer: 16 << 20,
  });
  const output = `${result.stdout ?? ''}${result.stderr ?? ''}`;
  const record = { tool: name, args, status: result.status, expected, duration_ms: Date.now() - started };
  records.push(record);
  if (result.error || (expected === 0 ? result.status !== 0 : result.status === 0)) {
    process.stderr.write(output);
    throw new Error(`${name} ${args.join(' ')} failed: ${result.error ?? result.status}`);
  }
  if (expected !== 0 && !/Invariant violated|invariant.*violat|[0-9]+ fail(?:ed|ing)/i.test(output)) {
    process.stderr.write(output);
    throw new Error('Mutation failed for an unexpected reason. Expected an invariant or test failure.');
  }
  if (name === 'openspec') {
    const data = JSON.parse(result.stdout);
    if (data.summary.totals.failed !== 0) throw new Error('OpenSpec reported invalid contracts');
    record.passed = data.summary.totals.passed;
  }
  if (name === 'quint' && args[0] === 'test' && expected === 0) {
    const passed = output.match(/(?:^|\n)\s*(\d+) passing\b/);
    if (!passed) throw new Error('Quint test output did not report its passing test count');
    record.passed = Number(passed[1]);
  }
  if (args[0] === 'run') {
    record.witnesses = [...output.matchAll(/(\w+) was witnessed in (\d+) trace\(s\) out of (\d+) explored/g)]
      .map(([, name, count, samples]) => ({ name, count: Number(count), samples: Number(samples) }));
    const requiredWitness = args.includes('--step=productiveStep') ? 'bothComplete' : args.includes('--step=browseStep') ? 'completeTraversal' : args.includes('--step=normalizeStep') ? 'normalizedComplete' : args.includes('--step=saveStep') ? 'savedCurrent' : args.includes('--step=recipeSaveStep') ? 'recipeSaved' : args.includes('--step=exportStep') ? 'capturedBeforeDiskChange' : args.includes('--step=historyStep') ? 'historySaved' : args.includes('--step=reloadStep') ? 'favoriteLoaded' : null;
    if (requiredWitness && !record.witnesses.some(w => w.name === requiredWitness && w.count > 0)) {
      process.stderr.write(output);
      throw new Error(`Coverage schedule never reached ${requiredWitness}`);
    }
  }
  console.log(`ok  ${name} ${args.filter(a => !a.startsWith('--verbosity')).join(' ')}`);
  return output;
}
run('openspec', ['validate', '--all', '--strict', '--no-interactive', '--json']);
const modelWitnesses = {
  publication: ['completed', 'collision', 'cancelledDuringWork'],
  quantization: ['bothComplete', 'hasMaskedRegion'],
  palette_discovery: ['filteredNonempty', 'emptyMatch', 'completeTraversal', 'beyondEnd'],
  normalization: ['normalizedComplete', 'swappedAxes', 'rejectedMetadata', 'canceledDuringNormalization'],
  mcp_apps: ['previewAccepted', 'staleDiscarded', 'canceledRequest', 'savedCurrent', 'paletteExtracted', 'recipeSaved', 'randomAccepted'],
  studio_export: ['sourceSizedExport', 'customSizedExport', 'changedInputRejected', 'capturedBeforeDiskChange'],
  style_history: ['hasHistory', 'historyRestored', 'staleIgnored', 'historySaved'],
  favorites: ['persistent', 'favoriteLoaded', 'receiptReported', 'staleRejected'],
};
for (const [model, witnesses] of Object.entries(modelWitnesses)) {
  run('quint', ['typecheck', `spec/${model}.qnt`]);
  run('quint', ['test', `spec/${model}.qnt`, '--backend=typescript']);
  run('quint', ['run', `spec/${model}.qnt`, '--backend=typescript', '--seed=20261003', '--max-samples=10000', '--max-steps=30', '--invariant=safety', '--verbosity=1', '--witnesses', ...witnesses]);
}
run('quint', ['run', 'spec/quantization.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=30', '--invariant=safety', '--step=productiveStep', '--verbosity=1', '--witnesses', 'bothComplete']);
run('quint', ['run', 'spec/palette_discovery.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initBrowse', '--step=browseStep', '--verbosity=1', '--witnesses', 'completeTraversal']);
run('quint', ['run', 'spec/normalization.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initNormalize', '--step=normalizeStep', '--verbosity=1', '--witnesses', 'normalizedComplete']);
run('quint', ['run', 'spec/mcp_apps.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initSave', '--step=saveStep', '--verbosity=1', '--witnesses', 'savedCurrent']);
run('quint', ['run', 'spec/mcp_apps.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initRecipeSave', '--step=recipeSaveStep', '--verbosity=1', '--witnesses', 'recipeSaved']);
run('quint', ['run', 'spec/studio_export.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=20', '--invariant=safety', '--init=initExport', '--step=exportStep', '--verbosity=1', '--witnesses', 'sourceSizedExport', 'capturedBeforeDiskChange']);
run('quint', ['run', 'spec/style_history.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initNavigation', '--step=historyStep', '--verbosity=1', '--witnesses', 'historySaved']);
run('quint', ['run', 'spec/favorites.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initReload', '--step=reloadStep', '--verbosity=1', '--witnesses', 'favoriteLoaded']);
// Remove the no-clobber guard in an isolated temporary copy. The negative test
// must fail because this regression permits replacement of existing outputs.
// A second mutation overlaps pages to challenge ordered complete enumeration.
// A third removes orientation readiness before downstream processing.
// Studio mutations remove the dirty guard, accept stale results, or write on preview.
// Export mutations bypass source/mask guards, reread paths after validation,
// retain invalidated previews, alter options/size, or ignore the output area limit.
const temp = mkdtempSync(path.join(os.tmpdir(), 'dither-spec-mutation-'));
try {
  for (const mutation of [
    { model: 'favorites', before: 'val success = s.disk.get(request.name) == -1', after: 'val success = true' },
    { model: 'favorites', before: 'val report = success and receiptMatches', after: 'val report = success' },
    { model: 'favorites', before: 's.fresh, s.draft == s.accepted,', after: 'true,' },
    { model: 'favorites', before: 'id == state.pending and request.revision == state.revision and request.epoch == state.epoch', after: 'true' },
    { model: 'favorites', before: 'val adopt = active and recipe == request.recipe', after: 'val adopt = active' },
    { model: 'favorites', before: 'Recipes.contains(recipe), not(writing), s.next < 6,', after: 'Recipes.contains(recipe), s.next < 6,' },
    { model: 'favorites', before: 'fresh: false, pending: -1, restarted: true, ...s', after: 'fresh: false, pending: -1, restarted: true, disk: Names.mapBy(_ => -1), ...s' },
    { model: 'favorites', before: 'pending: s.next, next: s.next + 1, fresh: false,', after: 'pending: s.next, next: s.next + 1, fresh: true,' },
    { model: 'style_history', before: 'val current = id == s.pending and request.revision == s.revision and request.epoch == s.epoch', after: 'val current = true' },
    { model: 'style_history', before: 'pure def bounded(items) = items.slice(0, if (items.length() < 3) items.length() else 3)', after: 'pure def bounded(items) = items' },
    { model: 'style_history', before: 'retained(undo, history, request.look)', after: 'bounded(undo)' },
    { model: 'style_history', before: 's.tools, not(s.saving), s.next < 6, Looks.contains(look),', after: 's.tools, s.next < 6, Looks.contains(look),' },
    { model: 'style_history', before: 'accepted: if (adopt) request.look else s.accepted,', after: 'accepted: if (adopt) s.active else s.accepted,' },
    { model: 'style_history', before: 'val history = bounded(List(request.look).concat(s.history.select(look => look != request.look)))', after: 'val history = if (member(s.history, request.look)) s.history else bounded(List(request.look).concat(s.history))' },
    { model: 'style_history', before: 'pending: s.next, next: s.next + 1, fresh: false,', after: 'pending: s.next, next: s.next + 1, fresh: true,' },
    { model: 'publication', before: 's.files.get(p) == -1,', after: 's.files.get(p) >= -1,' },
    { model: 'publication', before: 's.stage.get(j) == "encoding", s.encodedRows.get(j) < 2,', after: 's.encodedRows.get(j) < 2,' },
    { model: 'quantization', before: 'tones.indices().filter(i => weights.nth(i) > 0)', after: 'tones.indices().filter(i => weights.nth(i) > 0).intersect(Set(0, 2))' },
    { model: 'quantization', before: 'val folded = exact.size() > 2', after: 'val folded = false' },
    { model: 'quantization', before: 'sum + tones.nth(i) * weights.nth(i)', after: 'sum + tones.nth(i)' },
    { model: 'quantization', before: 'palette: ids.nth(i), algorithm: algorithm', after: 'palette: ids.nth(0), algorithm: algorithm' },
    { model: 'quantization', before: 'algorithms.length() == 0 and not(sharedPaletteOrRecipe)', after: 'not(sharedPaletteOrRecipe)' },
    { model: 'palette_discovery', before: 'else offset + count', after: 'else offset + count - 1' },
    { model: 'normalization', before: 's.phase == "active", s.orientationRuns == 1, s.colorRuns == 1,', after: 's.phase == "active", s.colorRuns == 1,' },
    { model: 'mcp_apps', before: 's.accepted >= 0, s.acceptedRevision == s.revision', after: 's.accepted >= 0' },
    { model: 'mcp_apps', before: 'val current = id == s.pending and s.requested.get(id) == s.revision', after: 'val current = true' },
    { model: 'mcp_apps', before: 'val extractionCurrent = id == s.pending and s.requested.get(id) == s.revision', after: 'val extractionCurrent = true' },
    { model: 'mcp_apps', before: 'val styleCurrent = id == s.pending and s.requested.get(id) == s.revision', after: 'val styleCurrent = true' },
    { model: 'mcp_apps', before: 's.connected, s.tools, not(s.saving), s.next < 3,', after: 's.connected, s.tools, s.next < 3,' },
    { model: 'mcp_apps', before: 'recipe: if (accept) s.requested.get(id) else s.recipe,', after: 'recipe: if (accept) s.requested.get(id) else s.recipe, writes: s.writes + 1,' },
    { model: 'mcp_apps', before: 'val canSaveRecipe = s.recipeOutput and saveSettingsReady', after: 'val canSaveRecipe = saveSettingsReady' },
    { model: 'mcp_apps', before: 'previewId: s.accepted, style: s.style,\n        explicit: explicit, output: s.recipeOutput', after: 'previewId: s.accepted, style: 0,\n        explicit: explicit, output: s.recipeOutput' },
    { model: 'mcp_apps', before: 'tools: s.tools, kind: "recipe", width: 2', after: 'tools: s.tools, kind: "recipe", width: s.exportWidth' },
    { model: 'mcp_apps', before: 'recipeReports: s.recipeReports + (if (success and receiptMatches) 1 else 0)', after: 'recipeReports: s.recipeReports + (if (success) 1 else 0)' },
    { model: 'studio_export', before: 'val matched = s.readSource == s.request.source and s.readMask == s.request.mask', after: 'val matched = s.readMask == s.request.mask' },
    { model: 'studio_export', before: 'val matched = s.readSource == s.request.source and s.readMask == s.request.mask', after: 'val matched = s.readSource == s.request.source' },
    { model: 'studio_export', before: 'decodedSource: s.readSource,', after: 'decodedSource: s.source,' },
    { model: 'studio_export', before: 'decodedMask: s.readMask, ...s', after: 'decodedMask: if (s.request.masked) s.mask else -1, ...s' },
    { model: 'studio_export', before: 'accepted: if (matched) s.accepted else false,', after: 'accepted: s.accepted,' },
    { model: 'studio_export', before: 'recipe: s.recipe, acceptedRecipe: s.recipe,', after: 'recipe: s.recipe + 1, acceptedRecipe: s.recipe,' },
    { model: 'studio_export', before: 'else if (s.mode == "source") SourceWidth else s.customWidth', after: 'else if (s.mode == "source") PreviewWidth else s.customWidth' },
    { model: 'studio_export', before: 'width * height <= MaxPixels', after: 'true' },
  ]) {
    const source = readFileSync(path.join(root, `spec/${mutation.model}.qnt`), 'utf8');
    if (!source.includes(mutation.before)) throw new Error(`Mutation anchor missing in ${mutation.model}`);
    const mutated = path.join(temp, `${mutation.model}.qnt`);
    writeFileSync(mutated, source.replace(mutation.before, mutation.after));
    run('quint', ['test', mutated, '--backend=typescript'], 1);
    records.at(-1).args[1] = `<temporary mutation of ${mutation.model}.qnt>`;
  }
} finally { rmSync(temp, { recursive: true, force: true }); }
const hashes = Object.fromEntries(Object.keys(modelWitnesses).map(model => `${model}.qnt`).map(name => [name,
  createHash('sha256').update(readFileSync(path.join(root, 'spec', name))).digest('hex')]));
const report = {
  scope: `Strict OpenSpec validation, finite Quint run tests, seeded bounded simulation, and ${records.filter(r => r.expected !== 0).length} negative mutation checks. Not exhaustive verification or a proof of Go or UI refinement.`,
  versions: { openspec: '1.14.0', quint: '0.33.0' }, seed: '20261003', source_sha256: hashes, checks: records,
  totals: {
    openspec_items: records.find(r => r.tool === 'openspec').passed,
    quint_models: Object.keys(modelWitnesses).length,
    quint_run_tests: records.filter(r => r.tool === 'quint' && r.args[0] === 'test' && r.expected === 0).reduce((sum, r) => sum + r.passed, 0),
    simulation_traces: records.filter(r => r.args[0] === 'run').reduce((sum, r) => sum + Number(r.args.find(a => a.startsWith('--max-samples=')).split('=')[1]), 0),
    mutation_checks: records.filter(r => r.expected !== 0).length,
  },
};
mkdirSync(path.join(root, '.spec-results'), { recursive: true });
writeFileSync(path.join(root, '.spec-results/latest.json'), JSON.stringify(report, null, 2) + '\n');
console.log('All specification checks passed. Report: .spec-results/latest.json');
