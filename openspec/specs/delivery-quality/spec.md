# Delivery quality specification

## Purpose

Deliver an inspectable project with honest capabilities, reproducible verification, and an attractive local showcase.

## Requirements

### Requirement: Executable specifications

The project SHALL include OpenSpec requirements, executable Quint models, and documented validation commands. It SHALL explicitly explain the finite abstraction and its refinement into implementation tests. It SHALL distinguish randomized simulation from exhaustive proof.

#### Scenario: Contributor verifies contracts
- **WHEN** a contributor installs the pinned development tooling and runs specification checks
- **THEN** OpenSpec validation, Quint typechecking, examples, and bounded seeded simulations execute reproducibly

### Requirement: Reproducible examples and recordings

The project SHALL include original or appropriately attributed example inputs, generated sample outputs, and multiple VHS tape files. It SHALL document how to replay the actual CLI/MCP demonstration commands. Recorded workflows SHALL exercise real application behavior rather than fabricated transcripts.

#### Scenario: Replay a demo
- **WHEN** a contributor installs the documented VHS dependencies and runs a demo target
- **THEN** the real local application produces the documented example artifacts and terminal recording

### Requirement: Static showcase

A separate website directory SHALL contain a static showcase with relative asset paths. The showcase SHALL not require a server-side runtime. External asset requests SHALL not be mandatory. It SHALL support hosting beneath a GitHub Pages project path.

#### Scenario: Host under a path prefix
- **WHEN** a server serves the website directory beneath a non-root path
- **THEN** its styles, images, demos, and internal navigation resolve correctly

### Requirement: Honest coverage and operational documentation

The project SHALL document installation, MCP connection, CLI workflows, and algorithm and palette semantics. It SHALL document limits, security assumptions, reference inspirations, and known exclusions. It SHALL not claim full reference parity or production certification from test counts or formal-model checks alone.

#### Scenario: Evaluate a reference capability
- **WHEN** a reader consults the feature matrix
- **THEN** they can distinguish implemented functionality, optional dependencies, approximations, and deliberate exclusions
