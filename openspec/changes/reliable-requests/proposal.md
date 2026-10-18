# Stop canceled encodes and reject ignored CLI flags

## Why

GIF palette scans and text encoders continue after cancellation, holding the service's serial admission slot. The CLI also accepts flags that a selected command silently ignores. Both behaviors can mislead users about what a request is doing.

## What changes

- Add context-aware encoding entry points while retaining compatible background wrappers.
- Propagate request cancellation through handwritten encoder loops, palette training, and encoded writes.
- Reject explicitly supplied CLI flags that the selected command does not consume.
- Preserve later studio control edits when advanced JSON settings are pending, and reject explicit null options on Apply.
- Add original-failure regressions, compatibility checks, and finite cancellation scenarios.

## Capabilities

### New capabilities

None.

### Modified capabilities

- `local-workspace`: cancellation during encoding and its limits.
- `agent-workflows`: command-specific explicit flag admission and mixed studio edits.

## Impact

Successful encoding keeps its current bytes. The service retains its final no-clobber publication check. Standard codecs remain cooperative at writer calls. Scripts that supplied ignored flags now receive an error and must remove those flags.
