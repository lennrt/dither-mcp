## Why

Repeated Random style clicks make exploration easy, but replace the only accepted preview. Users need a way back to a look they liked.

## What Changes

- Retain a bounded, session-only history of successful preview settings.
- Add Undo, Redo and direct look restoration beside the random-style workflow.
- Re-preview exact stored settings before restoring controls and save authority.
- Preserve recent discoveries across new branches without caching image data.

## Capabilities

### New Capabilities
- None; extend existing agent workflows.

### Modified Capabilities
- `agent-workflows`: bounded history, navigation and current-preview restoration.

## Impact

Studio state, controls, embedded bundle, browser/bridge tests, documentation and a finite history model. No new MCP tool, dependency, persistent browser storage, or host permission.
