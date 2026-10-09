# Executable contracts

The behavioral requirements live in [`openspec/specs`](../openspec/specs). The initial [`proposal`](../openspec/changes/initial-suite/proposal.md), [`design`](../openspec/changes/initial-suite/design.md), and [`tasks`](../openspec/changes/initial-suite/tasks.md) record the feature and architecture decisions. We wrote and strictly validated the initial baseline before implementing the application.

Quint models six small contracts covering publication, quantization, palette discovery, source normalization, MCP Apps interactions, and guarded studio exports. They make sequencing and invariants executable. Concrete implementation tests challenge the same claims. They are not a formal proof of the Go program or browser UI.

## Run the checks

The specification check requires Node.js and pnpm at the versions declared in `package.json`. It does not require Go.

```sh
pnpm install --frozen-lockfile --ignore-scripts
pnpm spec:check
```

The lockfile pins OpenSpec 1.14.0 and Quint 0.33.0. The harness invokes the TypeScript Quint backend, so it needs neither Java/Apalache nor a downloaded native simulator. The harness disables OpenSpec telemetry. Node and these packages are development dependencies only. The server runtime remains Go. An MCP Apps host executes the embedded studio script in its sandbox.

The command performs:

1. Strict validation of six baseline OpenSpec capabilities and five changes.
2. Typechecking of all six Quint modules.
3. Seventy-six executable run tests, including negative precondition scenarios.
4. 10,000 seeded traces of at most 30 transitions for each model.
5. 1,000 additional traces for each completion-focused schedule: quantization, catalog browsing, normalization, studio saves, and guarded exports. Each schedule requires a completion witness.
6. Fourteen negative mutation checks, each in an independent temporary model copy. Six remove the no-clobber guard, overlap pagination, bypass orientation readiness, permit dirty saves, accept stale studio results, or write during a preview. Eight bypass source/mask checks, reread source/mask paths after checking, retain an invalid preview, alter an accepted option, use the wrong source-size width, or ignore the export pixel limit. The harness confirms that the respective regression tests fail.

The harness writes a machine-readable report to `.spec-results/latest.json`. It includes model SHA-256 hashes, commands, versions, seed, return codes, durations, witness counts, and derived check totals. [`verification.json`](verification.json) is the checked snapshot delivered with this version. Rerunning the command writes a fresh local report without silently updating that snapshot.

## Publication model

[`publication.qnt`](publication.qnt) has two jobs and two destination slots. One destination begins occupied. The model may admit each job with safe/unsafe and within/over-budget inputs. Each job may then render, encode, cancel, fail, or publish atomically. Both jobs can target the same slot.

The model checks that visible files have complete bytes and that unsafe or oversized work never publishes. It checks that existing files are immutable. It also checks that a job canceled before the abstract commit cannot later publish. Concrete runs demonstrate rejection, a collision with exactly one winner, and cancellation after encoding. They also demonstrate a failed batch item that preserves an earlier success.

`safe` and `bounded` are abstract admission predicates, not implementations of path checking or resource estimation. `publish` is one atomic action. Its implementation refinement is a no-clobber filesystem link after complete encoding. A `Stat` followed by an ordinary replacing rename would not refine it.

The `noNetwork` property is true by construction because the model has no network action. Package review establishes runtime network independence. Simulation does not establish it.

## Quantization model

[`quantization.qnt`](quantization.qnt) has four pixels, eight scalar tones, a three-color palette, four seeds, an arbitrary selected-pixel subset, and zero/partial/opaque alpha classes. Two independent workers traverse those pixels in different orders using a coordinate-local seeded quantizer.

The model checks palette membership for selected visible pixels, unchanged unselected source values, and transparent black. It checks equal results at equal seeds. It also checks that workers process no more pixels after observed cancellation. Explicit runs cover each contract and verify that changing the seed changes the modeled noise.

This order-independence applies only to the coordinate-local quantizer in the abstraction. Real error-diffusion algorithms depend on traversal order. The implementation promises equal output for equal complete configurations, including serpentine settings. The model does not encode RGB floating point, diffusion weights, resize arithmetic, blue-noise quality, or effects. It models quantization after adjustments and before post-effects. Masks and effects intentionally limit final palette-membership guarantees.

## Palette discovery model

[`palette_discovery.qnt`](palette_discovery.qnt) specifies filtered discovery over six stable, ID-ordered entries, three categories, small color counts, and abstract metadata-search tokens. Token containment represents the real case-insensitive substring terms combined with AND. A bounded-query Boolean represents the UTF-8 byte-length check. The model reduces real limits of 256/default 32 to 3/default 2. This preserves the pagination protocol and keeps traces small.

The model validates category, count, page, and query admission. It applies all filters before slicing. It records each returned page and the concatenated traversal.

Invariants check filter soundness, effective page bounds, exact slices, and strict ordering. They check for duplicate IDs and incorrect next offsets. They also check that concatenation matches the filtered catalog segment.

Concrete tests cover empty and beyond-end pages, repeated requests, rejected ranges, and complete enumeration. An additional browsing schedule must reach the final page. This provides coverage evidence, not a liveness proof.

The model contains no real palette colors or prose. It does not validate aesthetic quality, historical attribution, UTF-8/string implementation details, or the original 22-entry compatibility fixture. Those properties belong to independent Go tests and catalog review. The [palette-library OpenSpec change](../openspec/changes/palette-library/proposal.md) records the expansion separately from the completed initial suite. Offsets refer to one unchanged query and catalog version. A filter change requires enumeration to restart.

## Source normalization model

[`normalization.qnt`](normalization.qnt) uses an asymmetric 3-by-2 image with six distinct pixel identities. Explicit expected permutations cover all eight EXIF orientations. Alpha values include zero, partial transparency, and values that require 16-bit precision.

Admission represents valid orientation, a supported profile, bounded metadata, and consistent declarations. The model rejects unsupported LUT/CMYK classes, malformed profiles, invalid orientation, and failed metadata predicates. It then permits orientation and pointwise color conversion in either order. Both must finish before a one-pixel crop represents downstream processing.

Invariants check that orientation remains a permutation and preserves pixel count. They check exact alpha transport, working dimensions, color association, and at-most-once operations. Rejected and canceled work cannot process pixels. An independent expected-order table challenges the coordinate formulas. A mutation removes orientation readiness and must fail the crop-order regression test.

Color conversion is an abstract scalar transformation. The model does not establish ICC arithmetic, PNG metadata precedence, parser safety, or perceptual color accuracy. A metadata-validity Boolean abstracts those admission checks. Implementation tests must verify binary layouts, numeric reference values, byte limits, alpha precision, and shared-loader integration.

The [image-normalization change](../openspec/changes/image-normalization/proposal.md) records the supported subset and implementation checklist. A completion-focused schedule checks that valid requests can reach downstream processing. Its witness is coverage evidence rather than a liveness proof.

## MCP Apps interaction model

[`mcp_apps.qnt`](mcp_apps.qnt) represents complete source, palette, options, and mask settings with four revision values. It permits three preview requests and three save attempts. Each request keeps its identity and original revision, including after cancellation. Successful responses become current only while both identity and revision match.

The model separates preview requests from explicit save authorization and publication outcomes. Save requires host tool support, a destination, a current successful preview, and no pending operation. An authorization records the current revision, accepted preview, and replay recipe. Invariants check that these agree. Run tests cover edits during requests, cancellation, late results, failed replacements, missing destinations, and busy-state guards.

Three mutations challenge the studio tests. They permit dirty saves, accept stale responses, or increment the publication count while receiving a preview. A completion-focused schedule must witness a successful save. This supplies coverage evidence rather than a liveness proof.

Revision numbers abstract complete settings. The model does not parse tool data, render controls, implement the host bridge, validate filesystem paths, or compare image pixels. Preview actions have no publication transition by design. Go tests must independently check that the actual preview service writes no files. Controller and browser tests must check that real events enforce the modeled guards. The model alone establishes no UI behavior or machine-checked refinement.

The [MCP Apps change](../openspec/changes/mcp-apps/proposal.md) defines negotiated presentation and fallback. The [studio guide](../docs/mcp-apps.md) describes host requirements, saves, and the runtime boundary.

## Guarded studio export model

[`studio_export.qnt`](studio_export.qnt) separates export size from preview settings. It uses three encoded-byte identities each for source and mask, three opaque recipe tokens for all non-dimensional options, fixed 2-by-2 preview and 6-by-4 source geometry, and custom dimensions from zero through nine. An eight-pixel axis limit and 48-pixel area limit stand in for the real engine bounds. The model permits three accepted previews and three successful publications.

The default mode replays preview dimensions. An explicit source/custom choice changes only the dimensions sent for saving. Admission rejects zero, missing, and over-budget modeled dimensions. The concrete UI additionally checks JavaScript types, fractions, and unsafe integers, which this integer model does not represent.

Independent disk mutations may occur before or after loading. The save path captures buffer identities, checks them against the accepted preview tokens, decodes those captured buffers, then publishes. A mismatch invalidates the preview and requires another preview before saving. Invariants check dimension bounds, explicit authorization, unchanged non-dimensional options, and exact accepted source/mask identity at publication. Seventeen runs cover default/source/custom exports, bounds, source and mask changes, recovery, and files changed after loading or checking. Eight isolated mutations challenge these new contracts. A completion-focused schedule must witness a source-size export using the accepted captured bytes after the source path changes.

The tokens abstract SHA-256 equality; they neither implement hashing nor prove collision resistance. The buffer-loading action does not imply an atomic snapshot across multiple files. The source dimensions are fixed abstract values; actual orientation, cropping, and proportional resizing require implementation tests. This model has no codecs, image pixels, filesystem paths, unreadable-input states, JSON, host lifecycle, UI events, cancellation, or no-clobber publication protocol. Guarded read failures and their error shape require the concrete service/protocol tests. The other models address separate protocol aspects without a machine-checked composition or refinement proof. Bounded seeded traces and completion witnesses provide coverage evidence, not exhaustive verification or liveness.

The [studio-export change](../openspec/changes/studio-export/proposal.md) defines byte guards, independent export sizes, and reproducible host/embedding guidance. Fingerprints are consistency checks over encoded bytes, not authentication or a promise that the on-disk path stays unchanged after it is read.

## Refinement evidence

| Contract | Implementation boundary | Implementation evidence |
|---|---|---|
| Palette compatibility and data quality | Built-in engine catalog | `TestLegacyPaletteCompatibility`, `TestPaletteLibraryQuality`, `TestPaletteLibraryReferences`, `TestEveryPaletteRenders` |
| Filtered discovery and complete pagination | Application discovery shared by MCP/CLI | `TestPaletteDiscoveryPagination`, `TestPaletteDiscoveryFilterComposition`, `TestPaletteDiscoveryInvalidInputs`, `TestPaletteDiscoveryCLI`, `TestProtocolWorkflow` |
| Root-safe admission | `internal/app` path validation and `os.Root` | `TestSecurityRejectsNonlocalPaths`, `TestSecurityRejectsSymlinkEscapes` |
| No-clobber atomic publication | `Service.publish` | `TestSecurityConcurrentPublicationHasOneWinner`, `TestSecurityOutputNeverOverwrites` |
| No publication after observed cancellation | Context propagation and final commit check | `TestSecurityCancellationDoesNotPublish` |
| Bounded inputs and aggregate animation | Byte/header checks, `frameDimensions`, GIF scanner | `TestSecurityLimitsFailBeforePublication`, `TestAnimationPreflightRejectsAggregateBeforeAllocation` |
| Recipe admission | Strict JSON plus engine validation | `TestSecurityStrictArguments`, `TestSecurityRecipeValidation` |
| Quantization/alpha/repeatability | Pure engine pipeline | Engine palette-membership, alpha, deterministic-seed, cancellation, and regression tests |
| GIF composition and timing | Source compositor and frame encoder | `TestGIFCompositesLogicalBackgroundAndPrevious`, `TestGIFDisposalBackgroundRestoresColorOrTransparency`, `TestGIFSourceAnimationPreservesTimingAndLoop` |
| Orientation and source alpha | Shared metadata parser and oriented image view | `TestAllOrientationsPreserveSamples`, `TestEXIFContainersAndBounds`, `TestReviewEXIFDirectoryAdmission` |
| Metadata admission and precedence | Container scanners and selected color transform | `TestICCContainersAndSequence`, `TestPNGPrecedenceAndBoundedMetadata`, `TestReviewBMPProfileAdmission`, `TestReviewCICPPrecedesProfileInternals`, `TestReviewJPEGMetadataBetweenScans` |
| Numerical color conversion and alpha precision | Bounded matrix/TRC converter | `TestIndependentLittleCMSVectors`, `TestPublishedCSSColorVectors`, `TestMalformedProfiles`, `TestAlphaPrecisionBoundsAndConcurrency`, `TestApplyAdmissionAndCancellation` |
| Normalization before shared operations | Application still loader and mask loader | `TestNormalizationAcrossDecodedContainers`, `TestNormalizationSharedWorkflows`, `TestMaskNormalizationAndFailureBeforePublication` |
| Read-only processed previews and exact replay | Shared application studio service | `TestStudioPreviewReplaysWithoutWriting`, `TestStudioGeometryAndAdmission`, `TestStudioImageMaskReplay`, `TestStudioNormalization`, `TestStudioEncodedBudgetAndSafeSeeds` |
| Guarded byte snapshots and source-size geometry | Shared bounded read/decode buffers and studio metadata | `TestGuardedRenderDetectsSourceAndMaskReplacement`, `TestGuardChecksBytesBeforeDecodingAndPreservesNormalRenders`, `TestStudioSourceDimensionsFollowOrientationAndCrop`, `TestStudioAndGuardedRenderUseTheDecodedSnapshot`, `TestEmbeddedRenderRequestsApplySourceGuard` |
| Negotiated MCP Apps linkage and core fallback | Tool filter, resource handler, and result encoder | `TestAppsNegotiationAndFallback`, `TestStdioProtocolEras`, `TestStudioExportGuardsReturnStructuredMCPError` |
| Dirty controls, stale responses, and exact save arguments | Concrete JavaScript state controller | [`ui/state.test.mjs`](../ui/state.test.mjs), including canceled replacements, late host results, custom color preservation, hidden options, invalid seeds, export bounds, exact dimension-only changes, guard forwarding, and mismatch invalidation/recovery |
| Host lifecycle and tool-call capability | Official SDK App and AppBridge over paired transports | [`ui/bridge.test.mjs`](../ui/bridge.test.mjs) checks initial input/result, exact save arguments, theme, cancellation, and a host without tool capability |

The implementation tests independently exercise real code. They do not come from Quint traces. The SDK bridge tests use paired in-memory transports and do not render a browser document. The local browser harness can exercise that separate integration boundary. These checks do not establish machine-checked refinement. Review the tests together with the models when changing either boundary.

## What the checks do not establish

Seeded simulation samples finite traces. It is **not exhaustive model checking**, and it does not prove liveness. Witness counts show that simulation reached a scenario. They do not show that simulation reached every scenario. The checks make no claims about these properties:

- Infinite executions or process crashes during a kernel operation.
- Power-loss durability or malicious codec internals.
- Filesystem behavior outside the native `os.Root` contract.
- Aesthetics, perceptual quality, or absence of every implementation bug.

A cancellation request may race the final publication check. The model's cancellation transition means cancellation observed before the abstract commit. It does not guarantee that no output exists whenever a client stops waiting. A batch is intentionally a series of independent transactions.

## Changing a contract

1. Add or update OpenSpec requirements and scenarios.
2. Revise the relevant Quint abstraction where sequencing changes.
3. Add implementation regression tests.
4. Rerun specification and Go checks.
 Keep unsupported capabilities explicit in the feature matrix. Archive the initial OpenSpec change only when every task is verified. An unarchived proposal alone does not establish that runtime behavior is incomplete.
