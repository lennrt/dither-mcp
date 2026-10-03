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
  if (args[0] === 'run') {
    record.witnesses = [...output.matchAll(/(\w+) was witnessed in (\d+) trace\(s\) out of (\d+) explored/g)]
      .map(([, name, count, samples]) => ({ name, count: Number(count), samples: Number(samples) }));
    const requiredWitness = args.includes('--step=productiveStep') ? 'bothComplete' : args.includes('--step=browseStep') ? 'completeTraversal' : args.includes('--step=normalizeStep') ? 'normalizedComplete' : null;
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
};
for (const [model, witnesses] of Object.entries(modelWitnesses)) {
  run('quint', ['typecheck', `spec/${model}.qnt`]);
  run('quint', ['test', `spec/${model}.qnt`, '--backend=typescript']);
  run('quint', ['run', `spec/${model}.qnt`, '--backend=typescript', '--seed=20261003', '--max-samples=10000', '--max-steps=30', '--invariant=safety', '--verbosity=1', '--witnesses', ...witnesses]);
}
run('quint', ['run', 'spec/quantization.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=30', '--invariant=safety', '--step=productiveStep', '--verbosity=1', '--witnesses', 'bothComplete']);
run('quint', ['run', 'spec/palette_discovery.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initBrowse', '--step=browseStep', '--verbosity=1', '--witnesses', 'completeTraversal']);
run('quint', ['run', 'spec/normalization.qnt', '--backend=typescript', '--seed=20261003', '--max-samples=1000', '--max-steps=12', '--invariant=safety', '--init=initNormalize', '--step=normalizeStep', '--verbosity=1', '--witnesses', 'normalizedComplete']);
// Remove the no-clobber guard in an isolated temporary copy. The negative test
// must fail because this regression permits replacement of existing outputs.
// A second mutation overlaps pages to challenge ordered complete enumeration.
// A third removes orientation readiness before downstream processing.
const temp = mkdtempSync(path.join(os.tmpdir(), 'dither-spec-mutation-'));
try {
  for (const mutation of [
    { model: 'publication', before: 's.files.get(p) == -1,', after: 's.files.get(p) >= -1,' },
    { model: 'palette_discovery', before: 'else offset + count', after: 'else offset + count - 1' },
    { model: 'normalization', before: 's.phase == "active", s.orientationRuns == 1, s.colorRuns == 1,', after: 's.phase == "active", s.colorRuns == 1,' },
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
  scope: 'Strict OpenSpec validation, finite Quint run tests, seeded bounded simulation, and three negative mutation checks. Not exhaustive verification or a proof of Go refinement.',
  versions: { openspec: '1.14.0', quint: '0.33.0' }, seed: '20261003', source_sha256: hashes, checks: records,
};
mkdirSync(path.join(root, '.spec-results'), { recursive: true });
writeFileSync(path.join(root, '.spec-results/latest.json'), JSON.stringify(report, null, 2) + '\n');
console.log('All specification checks passed. Report: .spec-results/latest.json');
