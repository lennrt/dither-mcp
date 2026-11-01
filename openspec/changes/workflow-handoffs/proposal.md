## Why

Comparison selection requires callers to reconstruct recipe settings. Conflicting
CLI options can silently change a render. Canceled batches omit later item outcomes.

## What Changes

- Return an exact reusable version-1 recipe for every comparison cell.
- Reject engine settings supplied through both explicit flags and `--options`.
- Return one indexed outcome per admitted batch item, including canceled work.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `agent-workflows`: exact comparison handoff, unambiguous CLI options, and complete batch outcomes.

## Impact

Shared application and CLI code, behavioral tests, API and CLI guides, and a
publication-model regression. No new dependency, tool, UI, or publication authority.
