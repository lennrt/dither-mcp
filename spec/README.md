# Executable contracts

The behavioral requirements live in [`openspec/specs`](../openspec/specs). The initial [`proposal`](../openspec/changes/initial-suite/proposal.md), [`design`](../openspec/changes/initial-suite/design.md), and [`tasks`](../openspec/changes/initial-suite/tasks.md) record the feature and architecture decisions. We wrote and strictly validated the initial baseline before implementing the application.

Quint models four small contracts covering publication, quantization, palette discovery, and source normalization. They make sequencing and invariants executable. Concrete Go tests challenge the same claims. They are not a formal proof of the Go program.

## Run the checks

The specification-only command requires Node.js 20.19 or later and pnpm. It does not require Go.

```sh
pnpm install --frozen-lockfile --ignore-scripts
pnpm spec:check
```

The lockfile pins OpenSpec 1.14.0 and Quint 0.33.0. The harness invokes the TypeScript Quint backend, so it needs neither Java/Apalache nor a downloaded native simulator. The harness disables OpenSpec telemetry. Node and these packages are development dependencies only. The runtime remains Go.

The command performs:

1. Strict validation of six baseline OpenSpec capabilities and three changes.
2. Typechecking of all four Quint modules.
3. Forty-three executable run tests, including negative precondition scenarios.
4. 10,000 seeded traces of at most 30 transitions for each model.
5. 1,000 additional traces for each completion-focused schedule: quantization, catalog browsing, and normalization. Each schedule requires a completion witness.
6. Three negative mutation checks, each in an independent temporary model copy. They remove the no-clobber guard, overlap pagination, or bypass orientation readiness. The harness confirms that the respective regression tests fail.

The harness writes a machine-readable report to `.spec-results/latest.json`. It includes model SHA-256 hashes, commands, versions, seed, return codes, durations, and witness counts. [`verification.json`](verification.json) is the checked snapshot delivered with this version. Rerunning the command writes a fresh local report without silently updating that snapshot.

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

## Refinement evidence

| Contract | Implementation boundary | Go evidence |
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

The Go tests independently exercise real code. They do not come from Quint traces. They do not establish machine-checked refinement. Review the tests together with the models when changing either boundary.

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
