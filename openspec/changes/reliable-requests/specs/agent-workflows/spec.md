## ADDED Requirements

### Requirement: Explicit CLI flag admission

The CLI SHALL reject explicitly supplied flags that its selected command does not consume. It SHALL identify the unsupported flag and command before opening the workspace or creating an output. Supplying zero or false SHALL NOT make an unsupported flag valid. Supported command flags, common startup flags, help, and generic tool-call JSON SHALL retain their documented behavior.

#### Scenario: Inspection receives render settings
- **WHEN** a caller supplies `inspect --width 100` or an output path flag
- **THEN** the CLI returns an actionable error instead of silently ignoring those flags

#### Scenario: Zero-valued irrelevant flag
- **WHEN** a caller supplies `render --frames 0`
- **THEN** the CLI rejects the unsupported frame flag

#### Scenario: Valid palette comparison
- **WHEN** a caller supplies the supported palette comparison flags
- **THEN** the CLI forwards the intended comparison through the shared service

### Requirement: Preserve mixed studio option edits

When an engine control changes after pending valid advanced JSON, the studio SHALL merge the latest control field into those parsed options. It SHALL preserve all other options. If the JSON is malformed or not an object, the studio SHALL retain the raw text and draft, restore the control, and report the correction needed. Neither case SHALL call a tool without an explicit preview request. Direct Apply SHALL reject malformed or nonobject advanced JSON, including explicit `null`, without replacing the draft or calling a tool. Omitted host options SHALL continue to use defaults.

#### Scenario: Adjust tone after advanced settings
- **WHEN** a user enters valid JSON with an effect and then changes brightness
- **THEN** the next preview receives both the effect and the latest brightness value

#### Scenario: Control changes while JSON is invalid
- **WHEN** a user changes a control while advanced JSON is malformed or not an object
- **THEN** the studio preserves the JSON text and previous draft and explains the error
- **AND** it makes no tool call

#### Scenario: Apply explicit null options
- **WHEN** a user enters `null` in advanced JSON and selects Apply preview
- **THEN** the studio retains the text and existing draft, reports that options must be an object, and makes no tool call
- **AND** correcting the JSON allows preview again
