## ADDED Requirements

### Requirement: Bounded studio look history

The studio SHALL retain at most 20 successful canonical preview requests in session memory, including initial, manual and random previews. It SHALL deduplicate equal requests using stable look identities and keep recent selections available through most-recently-used ordering. The history SHALL store settings rather than image data or save receipts. New host input SHALL clear history and navigation.

#### Scenario: Recover a completed random look
- **WHEN** several random previews succeed
- **THEN** the user can identify and select a previous look by its algorithm, palette and seed
- **AND** canceled, failed and superseded requests create no history entry

#### Scenario: Keep discoveries after branching
- **WHEN** a user restores an earlier look and generates another
- **THEN** previously discovered looks remain selectable within the history capacity
- **AND** Undo returns to the look displayed immediately before the new preview

#### Scenario: Bound history and navigation
- **WHEN** accepted looks exceed the 20-entry capacity
- **THEN** the least recently used look is evicted and unavailable navigation IDs are pruned
- **AND** restoring an old look before generating a new one preserves that immediate undo target

#### Scenario: Avoid duplicate navigation and reset source scope
- **WHEN** the accepted settings match the active look
- **THEN** the studio creates no duplicate entry or undo loop and preserves Redo
- **WHEN** new host input arrives
- **THEN** the studio clears earlier history and rejects its pending completions

### Requirement: Fresh preview history restoration

The studio SHALL offer explicit Undo, Redo and direct history restoration near Random style. Restoration SHALL call `dither_studio` with the exact stored request, preserving all accepted settings. Only a validated current success SHALL change active history, navigation and controls, and grant save authority. Navigation SHALL require tool capability and idle operation state. Restoration SHALL create no file.

#### Scenario: Restore and save exact settings
- **WHEN** a user restores a look on unchanged source and mask files
- **THEN** the fresh preview reproduces its pixels and refreshes the recipe available for explicit saving
- **AND** history restoration makes no random-style generation call

#### Scenario: Re-read changed source files
- **WHEN** source or mask bytes change before restoration
- **THEN** the existing preview tool reads current files and returns fresh fingerprints or an error
- **AND** history does not silently reuse previous image data or authority

#### Scenario: Retain work until a successful restore
- **WHEN** current controls or advanced text contain unapplied changes and a restore is requested
- **THEN** those changes remain until the restored preview succeeds
- **AND** failed, canceled, malformed, edited or superseded responses do not move navigation or discard current text

#### Scenario: Preserve exclusive writes and current request identity
- **WHEN** another operation is pending or host tools are unavailable
- **THEN** Undo, Redo and history restoration make no tool call
- **WHEN** Random style supersedes a pending restoration
- **THEN** the old response cannot change history or authorize saving
