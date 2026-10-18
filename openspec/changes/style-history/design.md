## History and navigation

Keep at most 20 canonical accepted requests in memory. Include the initial preview, manual refinements and random previews. Deep-clone complete settings, including custom colors, crop, masks, dimensions, pixel scale and effects. Store no PNG, source bytes, fingerprint or publication receipt. Equal requests reuse a stable look ID. Present labels identifying the look, algorithm, palette and seed.

The gallery uses most-recently-used order. Successful restoration promotes an entry, so selecting an old look and generating another preserves the immediate undo target. Separate Undo and Redo stacks hold at most 20 IDs each. New previews or explicit history selections push the prior active ID and clear Redo, while keeping the gallery. Undo and Redo transfer IDs only after successful preview. Re-selecting the active settings does not create navigation loops or clear Redo. Eviction prunes unavailable IDs. New host input resets history and navigation.

This separates a discovery collection from navigation: a new experiment after Undo must not delete all later discoveries. Persistent favorites and cross-session storage are deferred; Save recipe already provides durable reuse.

## Restore through the existing preview tool

History restoration sends the exact stored request to `dither_studio`. It does not regenerate a seed or reuse a cached image. The fresh preview renews source/mask fingerprints and the embedding host's accepted recipe authority. Changed source bytes may change the result; history preserves settings, not historical source files.

Navigation requires host tools and no pending operation. Random style may supersede restoration as another read-only preview. Restores use the existing operation identity and edit revision checks. Failed, canceled, malformed, superseded or edited requests cannot change the active look, stacks, gallery, controls or raw pending text. Saving remains invalid until a current preview succeeds.

Explicit restoration can replace dirty controls or invalid JSON, but only after success. Pending work preserves the displayed image and draft. Image/recipe writes cannot be interrupted, and restoration itself publishes no artifact. No host allowlist or admission expansion is needed.

## Verification

State tests cover branching, deduplication, capacity and stable IDs, cloning, source reset, errors/cancellation, dirty text and exact request restoration. Real stdio/browser tests compare restored image and saved recipe bytes, challenge host authority, and inspect keyboard/mobile/dark presentation. The finite history model reduces the gallery and stacks to three entries and represents complete requests as opaque look IDs. It checks sequencing and bounds, not actual pixel rendering, browser storage or a machine-checked implementation refinement.
