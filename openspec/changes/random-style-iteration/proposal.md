## Why

Trying combinations of 41 algorithms and 256 palettes requires repeated manual choices. A repeatable random-style tool and one-click studio preview make creative iteration accessible immediately.

## What Changes

- Add a read-only, seedable `dither_random_style` MCP tool returning a validated style recipe.
- Put Random style above the studio preview and automatically preview each generated style.
- Preserve geometry, cancel superseded read-only work, and accept only the latest complete preview.
- Extend the embedding host narrowly and document reproducible API and image examples.

## Capabilities

### New Capabilities
- None; extend existing agent workflows.

### Modified Capabilities
- `agent-workflows`: reproducible random-style generation and rapid studio iteration.

## Impact

Shared Go service, MCP schema, studio bundle/controller, embedding host, tests, finite app model and documentation. No dependencies, file writes by generation, releases or deployments.
