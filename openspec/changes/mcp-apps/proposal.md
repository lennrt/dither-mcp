# An interactive dither studio for MCP hosts

## Why

Image choices benefit from direct visual comparison. An MCP Apps host can present a local source, processing controls, and a bounded preview together. The same operation must remain useful through ordinary MCP clients.

## What changes

- Add `dither_studio`, a public read-only tool that processes a source and returns an in-memory PNG with its replayable recipe.
- Associate the tool with a bundled HTML resource through the stable MCP Apps extension.
- Supply algorithm and palette controls from the same catalog that agents use.
- Apply edits only when the user requests a preview. Save through the existing `dither_render` tool after an explicit user action.
- Keep the current preview, pending work, and changed controls distinct. Reject stale results and disable saves for unpreviewed settings.
- Add an executable Quint interaction model, concrete protocol tests, UI checks, and documented verification limits.

## Capabilities

### New capabilities

None. The change extends the existing agent workflow capability.

### Modified capabilities

- `agent-workflows`: negotiated MCP Apps presentation, bounded processed previews, and explicit saves of successfully previewed settings.

## Impact

The server adds one public tool and one `ui://` resource. Hosts without MCP Apps receive the same typed tool and useful PNG/JSON results. Existing rendering, CLI workflows, rooted paths, and publication rules continue to apply.

The Go binary embeds the complete HTML resource. Building the UI requires pinned JavaScript development dependencies. Running the released binary requires no package manager, web server, CDN, or network connection.
