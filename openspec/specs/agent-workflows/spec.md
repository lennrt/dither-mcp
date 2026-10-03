# Agent workflows specification

## Purpose

Make creative iteration inspectable, repeatable, and scriptable through MCP and the CLI.

## Requirements

### Requirement: Typed stdio MCP

The application SHALL implement MCP initialization and stdio tool discovery with explicit input schemas and useful structured success/error results. Stdout in server mode SHALL contain only protocol messages. Diagnostics SHALL use stderr. Tool annotations SHALL accurately distinguish reads from file writes.

#### Scenario: Agent connects
- **WHEN** a client initializes the stdio server and lists tools
- **THEN** it receives discoverable image-workflow tools with schemas

#### Scenario: Invalid tool arguments
- **WHEN** a client calls a tool with invalid fields or types
- **THEN** the tool returns an actionable error without terminating the server

### Requirement: Inspection and catalog context

The suite SHALL expose image metadata and catalogs without modifying the source. It SHALL supply agent-readable usage context through documented resources or prompts, including local-root rules and recommended discover-inspect-compare-render workflows.

#### Scenario: Plan a render
- **WHEN** an agent inspects a source and reads algorithm/palette context
- **THEN** it can choose valid settings and destination dimensions before rendering

#### Scenario: Discover supported media
- **WHEN** an agent reads the capability catalog before choosing a media operation
- **THEN** the catalog identifies still formats, animation outputs, video containers, video outputs, enablement, dependencies, and limits

### Requirement: Bounded image previews

The suite SHALL provide a read-only PNG preview with a default requested width of 512 pixels. The preview SHALL not exceed 1,024 pixels per axis or 2 MiB of encoded content. It SHALL deliver the preview as MCP image content without creating a file.

#### Scenario: Inspect a rendered candidate
- **WHEN** an agent requests a preview of a supported local image
- **THEN** it receives a bounded PNG image and dimensions without a new filesystem artifact

### Requirement: Comparable candidates

The suite SHALL render a bounded set of candidate configurations from one input and generate a comparison artifact. Candidate ordering and labels SHALL be deterministic and traceable to the supplied configurations.

#### Scenario: Compare three algorithms
- **WHEN** an agent submits three valid algorithm choices
- **THEN** it receives a comparison with one corresponding candidate per choice in request order

### Requirement: Versioned recipes

The suite SHALL accept versioned JSON recipes that represent reproducible processing options. It SHALL reject unsupported recipe versions. Recipes SHALL not grant access outside the configured root or bypass validation and limits.

#### Scenario: Replay a saved look
- **WHEN** an agent reuses a recipe with the same source and seed
- **THEN** the same normalized processing options produce the same pixels

#### Scenario: Unsupported recipe version
- **WHEN** a recipe declares an unsupported version
- **THEN** the suite rejects it with a version error

### Requirement: Explicit batch outcomes

The suite SHALL process bounded batches with per-item results. It SHALL describe whether processing stops or continues after an item fails. It SHALL never claim transaction-wide rollback after publishing earlier items.

#### Scenario: One item fails
- **WHEN** a batch contains a valid render and an invalid input
- **THEN** the result identifies artifacts the suite published and the item that failed according to the documented failure policy

### Requirement: CLI shares application semantics

The CLI SHALL expose the principal discovery, inspection, rendering, and workflow operations through the same service used by MCP. It SHALL provide useful help, a version command, nonzero failure exit status, and machine-readable output where documented.

#### Scenario: Reproduce a tool call locally
- **WHEN** an agent submits equivalent validated configuration through CLI and MCP
- **THEN** both paths apply the same engine options and filesystem policies
