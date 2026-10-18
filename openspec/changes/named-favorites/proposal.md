## Why

Session history disappears when the studio closes. Users need named looks they can discover and reuse from another studio session or MCP client without remembering recipe paths.

## What Changes

- Store immutable named recipes inside the configured workspace.
- Add typed favorites save, list and load tools with bounded, rooted storage.
- Add persistent favorites controls and fresh-preview restoration to the studio.
- Preserve explicit current-preview save authority in the embedding host.

## Capabilities

### New Capabilities
- None; extend existing agent workflows.

### Modified Capabilities
- `agent-workflows`: persistent named favorites shared by MCP and the studio.

## Impact

Shared application service, MCP schemas, studio controller and bundle, restricted embedding host, tests, documentation and finite favorites contracts. No dependency, network service, overwrite, delete or source-file snapshot.
