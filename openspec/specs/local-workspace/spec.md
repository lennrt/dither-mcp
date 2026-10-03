# Local workspace specification

## Purpose

Keep image operations local, bounded, cancelable, and safe to repeat.

## Requirements

### Requirement: Root-confined local inputs

The service SHALL accept only relative input, output, recipe, and palette file paths. It SHALL resolve them within a configured workspace root through a race-resistant rooted filesystem API. It SHALL reject absolute paths, remote URLs, traversal outside the root, and symlink escapes. Runtime image processing SHALL not issue network requests.

#### Scenario: Local file under the root
- **WHEN** an agent requests inspection of an existing regular image file inside the root
- **THEN** the service returns image metadata without uploading the file

#### Scenario: Escaping or remote input
- **WHEN** a request references `../outside.png`, an absolute path outside the root, a symlink leaving the root, or an HTTP URL
- **THEN** the request fails before image processing or output publication

### Requirement: Bounded processing

The service SHALL impose these limits:

- 32 MiB per input and encoded output.
- 16,777,216 source/destination pixels.
- 16,384 pixels per dimension.
- 120 frames and 67,108,864 cumulative animation pixels.
- 32 batch items and 12 comparison items.
- A two-minute operation timeout, with documented optional video duration bounds.

It SHALL reject excessive dimensions before allocating destination image buffers. It SHALL inspect image headers before decoding where the format permits.

#### Scenario: Oversized image header
- **WHEN** an image header describes dimensions above the configured pixel limit
- **THEN** the service returns a limit error before decoding a full image buffer

#### Scenario: Excessive aggregate work
- **WHEN** a request exceeds the documented count or work limit for a batch, animation, or comparison
- **THEN** it fails with a useful bound in the error

### Requirement: Atomic publication and immutable destinations

The service SHALL publish only a completely encoded artifact and SHALL always refuse to replace an existing destination. There SHALL be no overwrite option. A failed encode or canceled request SHALL not publish a partial destination.

#### Scenario: Destination exists
- **WHEN** a render names an existing destination
- **THEN** the service reports a conflict and preserves its contents

#### Scenario: Encoding fails
- **WHEN** encoding or temporary-file finalization fails
- **THEN** an incomplete destination is not reported as a successful artifact

#### Scenario: Concurrent collision
- **WHEN** two requests concurrently publish to the same absent destination
- **THEN** at most one publishes successfully and the winner remains a complete artifact

### Requirement: Cooperative cancellation

The service SHALL propagate request cancellation to image loops and child processes. It SHALL release resources after cancellation. It SHALL check cancellation before publishing a destination. Cancellation observed before the commit SHALL stop publication. A cancellation that arrives after the final check MAY race a complete atomic commit.

#### Scenario: Cancel an active render
- **WHEN** the service observes cancellation of a long-running operation before its final publication check
- **THEN** work stops cooperatively and that request publishes no new final destination
