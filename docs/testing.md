# Verification

The repository separates executable contracts from implementation checks. Runtime commands need Go 1.27. Specs need Node >=20.19 and the pinned pnpm development toolchain. Terminal recordings need VHS, ttyd, and FFmpeg. Optional video integration tests also need FFmpeg/ffprobe.

```sh
make verify          # format, vet, tests, race, build, docs/schema drift, MCP session
make spec-install    # pinned OpenSpec/Quint. No runtime Node dependency.
make spec-check      # strict OpenSpec + Quint typechecks/tests/simulations/mutation
make fuzz            # four bounded five-second fuzz smoke runs
make bench           # engine benchmarks with allocation counts
make vuln            # current advisory database. Requires network.
make video-test      # optional FFmpeg/ffprobe integration tests
```

`make verify` does not run the optional native video test or specification
installation implicitly. Run these checks locally. Hosted CI setup is outside
the current repository scope.

## Test coverage

- Engine tests cover all 41 algorithms, deterministic seeds, palette membership, source alpha, and distinct pattern fingerprints. They check reference diffusion pixels, Bayer ranks, blue-noise ranks, and Hilbert traversal. They also check adjustment validation, mask/transparent diffusion barriers, cancellation, six encoder formats, and GIF palette fidelity.
- Filesystem and service tests cover traversal and URL rejection, symlink escapes, and named special files. They check two independent services that race for one destination, immutable destinations, and temporary-file cleanup. They also check argument validation, input/output budgets, canceled writes, and portable recipes with external masks.
- Workflow tests cover comparison cell pixels and metadata, partial batch success, versioned recipe replay, print plates, and PNG DPI CRCs. They check animation determinism, sheet frames, source-GIF background/disposal/timing/loop, and 12 fps timing quantization. They also check rendering with extracted palettes and bounded previews.
- MCP and CLI tests cover initialization, discovery, input/output schemas, native image results, and recoverable tool errors. They check protocol-only stdout, message framing limits, JSON CLI output, an actual subprocess demo, and overwrite rejection.
- Palette tests render every catalog entry and check legacy ID/color compatibility, unique color sets, and combined category/term/color-count filters. They check stable pagination through every result, empty and out-of-range pages, argument bounds, and MCP/CLI parity.
- Normalization tests cover all eight EXIF orientations, both byte orders, source alpha precision, and crop ordering. They check metadata across JPEG, PNG, WebP, TIFF, and supported BMP containers. They also check split JPEG profiles, metadata between scans, PNG precedence, malformed directories, external-profile rejection, and decompression limits. Application tests exercise the shared loader through inspection, preview, rendering, extraction, comparison, animation, separations, and masks.
- Color tests compare 45 independent LittleCMS reference vectors across sRGB, Display P3, and Adobe RGB profiles. Original binary fixtures and recorded vectors keep these tests independent of native libraries. Published CSS Color 4 vectors provide another numerical check. Tests also cover malformed profiles, alpha precision, cancellation, and concurrent transforms. The runtime and normal test suite gain no native color-management dependency.
- Video tests cover MP4/H.264, MOV/H.264, WebM/VP9, Matroska/FFV1, and AVI/MPEG-4 inputs. They also check content detection for an MP4 file with a `.bin` extension. The tests export GIF, sprite PNG, H.264 MP4, and VP9 WebM. They check codecs, frame counts, dimensions, even padding, and silent output from a source with AAC audio. These tests are opt-in so pure-Go tests remain portable.

`make fuzz` runs `FuzzProcess`, `FuzzStrictJSON`, `FuzzMetadata`, and `FuzzICCProfile` for five seconds each. These smoke runs challenge engine options, structured requests, metadata containers, and ICC parsing. They provide bounded exploration rather than exhaustive malformed-input coverage.

Tests use finite timeouts and small synthetic inputs. They do not demonstrate
large-scale throughput, every malformed decoder input, power-loss persistence,
all operating systems, or byte compatibility across future codec versions.

## Formal checks

There are 36 OpenSpec requirements across six capability specifications and three
implementation changes. Strict validation checks all nine items for structure and scenario
coverage. Quint models describe artifact publication, quantization, palette
discovery, and source normalization:

- Four typechecks and 43 concrete model tests.
- 40,000 seeded safety traces, each with at most 30 transitions.
- 1,000 additional traces for each completion-focused schedule: quantization, palette browsing, and normalization. Each requires a completion witness, for 43,000 traces overall.
- Three deliberate mutations weaken collision protection, overlap pagination boundaries, or bypass orientation before processing. The corresponding tests must detect all three regressions.

The seed is 20261003. Exact command results, source hashes, durations, and limits
are in [spec/verification.json](../spec/verification.json). The finite models do
not implement every diffusion kernel or system call, and the simulations are not
exhaustive model checking. The palette model uses six abstract records to exercise combined filters and pagination. The Go tests check the actual 256-entry catalog.
The normalization model checks six pixel identities and exact alpha transport through all eight orientations. Its abstract color transform does not establish numerical ICC accuracy. Independent implementation vectors check that separate claim.
Read [the model guide](../spec/README.md) for mappings between model requirements
and Go regression tests.

## Local evidence

[verification.json](verification.json) records the final local verification outcomes and environment. This is a snapshot from this local
implementation, not an assertion that a hosted CI system passed. Rerun the commands on your deployment platform before release.

The showcase uses deterministic original scenes and actual engine outputs.
Four VHS tapes record real CLI, palette discovery and MCP workflows. The site was
checked at desktop and mobile dimensions, including palette filtering, algorithm
selection, slider keyboard control, and overflow. See [showcase reproduction](showcase.md).
