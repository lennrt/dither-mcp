## ADDED Requirements

### Requirement: Verifiable host setup and reusable embedding

The project SHALL document the local stdio transport, MCP Apps extension and supported MIME type, app-to-host tool-call requirement, fallback behavior, and installation steps. Compatibility guidance SHALL distinguish cited host documentation from locally executed checks and unverified product behavior, with verification dates. The project SHALL include a runnable embedding example using the official Apps host bridge and the real stdio server through a loopback-only development adapter. The example SHALL document its sandbox, workspace-root configuration, lifecycle, reuse points, and development-only boundary. It SHALL not claim that this creates a production remote MCP endpoint.

#### Scenario: Reader chooses a host
- **WHEN** a reader follows the compatibility and installation guide
- **THEN** they can determine the separate transport, Apps rendering, and tool-call capabilities required for their setup
- **AND** each compatibility statement identifies its evidence or unverified boundary

#### Scenario: Developer embeds the studio
- **WHEN** a developer runs the documented embedding example with a valid workspace and built server
- **THEN** the host loads the bundled resource, supplies the official bridge, and supports preview and explicit save calls against the real stdio server
- **AND** stopping the example closes its development listener and child server
