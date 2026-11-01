## ADDED Requirements

### Requirement: Studio draft recovery and input preservation
The studio SHALL offer local draft restoration from its last accepted preview. Restoration SHALL preserve preview bytes, preview validity, history, source identity, and export choices. It SHALL make no tool call and SHALL remain unavailable during pending work or before the current source has an accepted preview. Selecting the active palette mode SHALL preserve draft settings and pending editor text.

#### Scenario: Discard invalid editor text
- **WHEN** a user discards malformed JSON or unfinished custom colors
- **THEN** the studio restores accepted controls and editor text without a tool call
- **AND** an invalidated preview still requires a successful new preview before saving

#### Scenario: Preserve custom draft colors
- **WHEN** a user selects the already active custom-color mode
- **THEN** the studio retains the custom colors and pending editor text

### Requirement: Studio multiline preview shortcut
Ctrl+Enter and Cmd+Enter in the advanced JSON and custom-color editors SHALL invoke the same explicit preview action as Apply preview. Ordinary Enter SHALL remain available for newlines. The shortcut SHALL preserve all existing validation and pending-operation checks.

#### Scenario: Apply advanced JSON from the keyboard
- **WHEN** a user presses Ctrl+Enter or Cmd+Enter in a multiline editor
- **THEN** the studio validates and previews the complete draft through the normal Apply action

### Requirement: Studio custom export aspect ratio
On first selecting custom export size for each host input, the studio SHALL initialize dimensions from the accepted preview. Later mode changes SHALL preserve intentionally empty or invalid fields. It SHALL offer an aspect-ratio lock based on that preview. With the lock enabled, editing either positive integer dimension SHALL update the other dimension using nearest-integer rounding with a minimum of one pixel. When accepted preview geometry changes, a locked valid custom export SHALL retain its width and recalculate its height using the new ratio. Invalid or unlocked custom dimensions SHALL remain unchanged. Existing export limits SHALL still disable invalid exports. Export changes SHALL not change preview settings.

#### Scenario: Resize a custom export proportionally
- **WHEN** the user edits one valid custom dimension with the aspect-ratio lock enabled
- **THEN** the other dimension follows the accepted preview ratio and export bounds remain enforced
