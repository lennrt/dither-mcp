## ADDED Requirements

### Requirement: Cancellable encoding

The engine SHALL provide context-aware encoding and palette mapping while retaining its existing background-context entry points. Handwritten encoder image loops SHALL observe cancellation between rows. GIF palette extraction SHALL receive the caller's context. Application encoding SHALL retain the request context through encoded writes and report observed cancellation without publication. Standard codecs MAY finish work between writer calls. Successful uncanceled encoding SHALL retain existing format, color, and alpha semantics.

#### Scenario: Cancel GIF palette mapping
- **WHEN** cancellation occurs during a GIF palette scan, extraction, or mapping pass
- **THEN** the encoder stops cooperatively and returns a context error
- **AND** the application creates no final artifact from that encode

#### Scenario: Cancel an encoded write
- **WHEN** the encoding writer observes request cancellation
- **THEN** it stops forwarding bytes and reports cancellation

#### Scenario: Preserve successful exports
- **WHEN** an uncanceled request uses the context-aware encoder
- **THEN** it produces the same encoded bytes as the compatible background entry point
