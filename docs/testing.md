# Verification

The repository separates executable contracts from implementation checks. Building the server needs Go 1.27. The full verification suite and specifications need Node >=20.19 and the pinned pnpm development dependencies. The built server needs neither Node nor pnpm. Terminal recordings need VHS, ttyd, and FFmpeg. Optional video integration tests also need FFmpeg/ffprobe.

```sh
make spec-install    # install pinned UI and specification development dependencies
make verify          # Go checks, MCP session, docs/schema drift, UI build/state/bridge checks
make spec-check      # strict OpenSpec + Quint typechecks/tests/simulations/mutation
make fuzz            # four bounded five-second fuzz smoke runs
make bench           # engine benchmarks with allocation counts
make vuln            # current advisory database. Requires network.
make video-test      # optional FFmpeg/ffprobe integration tests
pnpm exec playwright install chromium
make ui-browser      # optional real-browser checks with an isolated local host
make embedding-test  # supported host example: stdio and boundary tests
make embedding-browser # reusable embedding example in Chromium
```

`make verify` does not run the optional native video test or specification
installation implicitly. Run these checks locally. Hosted CI setup is outside
the current repository scope.

## Test coverage

- Engine tests cover all 41 algorithms, deterministic seeds, palette membership, source alpha, and distinct pattern fingerprints. They check reference diffusion pixels, Bayer ranks, blue-noise ranks, and Hilbert traversal. They also check adjustment validation, mask/transparent diffusion barriers, cancellation, six encoder formats, and GIF palette fidelity. Sparse visible-pixel tests challenge missed extraction samples. Context-aware encoder tests cancel during scanning, training, mapping, and writes; six baseline hashes check unchanged successful output bytes.
- Filesystem and service tests cover traversal and URL rejection, symlink escapes, and named special files. They check two independent services that race for one destination, immutable destinations, and temporary-file cleanup. They also check argument validation, input/output budgets, canceled writes, and portable recipes with external masks.
- Workflow tests cover algorithm and palette comparison cell pixels and metadata, partial batch success, versioned recipe replay, print plates, and PNG DPI CRCs. They check animation determinism, sheet frames, source-GIF background/disposal/timing/loop, and 12 fps timing quantization. They also check rendering with extracted palettes and bounded previews.
- MCP and CLI tests cover initialization, discovery, input/output schemas, native image results, and recoverable tool errors. They check protocol-only stdout, message framing limits, JSON CLI output, unsupported flags including explicit zero/false, an actual subprocess demo, and overwrite rejection.
- Studio tests check memory-only previews, default and inferred dimensions, crop, EXIF/color normalization, exact recipe replay, image masks, safe integer seeds, the 2 MiB PNG budget, and cancellation. Guard tests replace files with equal-size, equal-timestamp content; remove inputs; and replace paths during decoding. They verify the exact captured bytes and inherited render workflows. MCP tests check negotiated UI linkage, fallback JSON/PNG, resource MIME/CSP, and unchanged shared descriptors.
- JavaScript tests cover the delivered bundle syntax, concrete controller cases, and two official SDK bridge cases. A third bridge test connects to the real Go stdio server, changes a preview, saves identical PNG bytes, verifies overwrite rejection, checks source/custom export dimensions, and rejects changed source bytes. The optional Playwright suite renders the actual embedded app in Chromium. It checks random generation/preview, exact recipe saves, extraction races, mixed advanced JSON/control edits, negative-number typing, invalid JSON preservation, export-size limits, source-change recovery, keyboard saving without sandbox form permission, validation, cancellation, host capabilities, theme changes, and narrow layouts. These checks do not establish compatibility with every host.
- Palette tests render every catalog entry and check legacy ID/color compatibility, unique color sets, and combined category/term/color-count filters. They check stable pagination through every result, empty and out-of-range pages, argument bounds, and MCP/CLI parity.
- Normalization tests cover all eight EXIF orientations, both byte orders, source alpha precision, and crop ordering. They check metadata across JPEG, PNG, WebP, TIFF, and supported BMP containers. They also check split JPEG profiles, metadata between scans, PNG precedence, malformed directories, external-profile rejection, and decompression limits. Application tests exercise the shared loader through inspection, preview, rendering, extraction, comparison, animation, separations, and masks.
- Color tests compare 45 independent LittleCMS reference vectors across sRGB, Display P3, and Adobe RGB profiles. Original binary fixtures and recorded vectors keep these tests independent of native libraries. Published CSS Color 4 vectors provide another numerical check. Tests also cover malformed profiles, alpha precision, cancellation, and concurrent transforms. The runtime and normal test suite gain no native color-management dependency.
- Video tests cover MP4/H.264, MOV/H.264, WebM/VP9, Matroska/FFV1, and AVI/MPEG-4 inputs. They also check content detection for an MP4 file with a `.bin` extension. The tests export GIF, sprite PNG, H.264 MP4, and VP9 WebM. They check codecs, frame counts, dimensions, even padding, and silent output from a source with AAC audio. These tests are opt-in so pure-Go tests remain portable.

`make fuzz` runs `FuzzProcess`, `FuzzStrictJSON`, `FuzzMetadata`, and `FuzzICCProfile` for five seconds each. These smoke runs challenge engine options, structured requests, metadata containers, and ICC parsing. They provide bounded exploration rather than exhaustive malformed-input coverage.

Tests use finite timeouts and small synthetic inputs. They do not demonstrate
large-scale throughput, every malformed decoder input, power-loss persistence,
all operating systems, or byte compatibility across future codec versions.

The embedding example has separate tests for its five-tool allowlist, selected source, rooted paths, real stdio results, byte-identical PNG saves, and recipe save/load/replay. Its optional browser suite checks the loopback HTTP boundary, session token, cross-origin sandbox, six out-of-order random clicks, recipe authority, theme changes, and mobile layout. [Host compatibility](hosts.md) records external-client checks separately.

## Formal checks

There are 55 OpenSpec requirements across six capability specifications and nine
implementation changes. Strict validation checks all fifteen items for structure
and scenario coverage. Six Quint models cover artifact publication, quantization,
palette discovery, normalization, MCP Apps interactions, and guarded exports:

- Six typechecks and 119 concrete model tests.
- 60,000 seeded safety traces, each with at most 30 transitions.
- Six completion schedules add 1,000 traces each, for 66,000 traces overall. Each requires a completion witness.
- 27 mutations challenge output collisions, cancellation, histogram coverage/bounds/weighted sums, palette identity, pagination, normalization, stale previews/styles, explicit save authority, receipt validation, and guarded export behavior. The tests must detect every mutation.

The seed is 20261003. Exact command results, source hashes, durations, and limits
are in [spec/verification.json](../spec/verification.json). The finite models do
not implement every diffusion kernel or system call, and the simulations are not
exhaustive model checking. The palette model uses six abstract records to exercise combined filters and pagination. The Go tests check the actual 256-entry catalog.
The normalization model checks six pixel identities and exact alpha transport through all eight orientations. Its abstract color transform does not establish numerical ICC accuracy. Independent implementation vectors check that separate claim.
The export model uses abstract byte identities and fixed geometry. It checks accepted-buffer use despite independent disk changes; it does not establish SHA-256 collision resistance, actual codec behavior, host compatibility, or browser enforcement.
Read [the model guide](../spec/README.md) for mappings between model requirements
and Go regression tests.

## Local evidence

[verification-2026-10-18.json](verification-2026-10-18.json) records random styles, studio recipes, full-source color extraction, and earlier request-reliability verification outcomes. It includes the environment and performance tradeoffs. The earlier [verification.json](verification.json) retains historical host and showcase checks. These are local snapshots, not assertions that a hosted CI system passed. Rerun the commands on your deployment platform before release.

The showcase uses deterministic original scenes and actual engine outputs.
Four VHS tapes record real CLI, palette discovery and MCP workflows. The site was
checked at desktop and mobile dimensions, including palette filtering, algorithm
selection, slider keyboard control, and overflow in the earlier showcase verification. The MCP Apps update also passed Chromium browser checks at desktop and mobile sizes. The README and site show actual studio screenshots captured by Playwright. A labeled workflow illustration remains available. See [showcase reproduction](showcase.md).

The October 8, 2026 dependency audit found one advisory in the OpenSpec development
tool chain: [GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)
affects `braces` 3.0.3 when it parses deeply nested patterns. The advisory names
3.0.4 as the fix, but that version was unavailable from the package registry
during verification. This dependency is absent from the Go server and embedded
studio bundle. The Quint archive dependency is pinned to patched `adm-zip`
0.6.1 in `pnpm-workspace.yaml`. Rerun `pnpm audit` when updating development tools.

## Full-source extraction cost

The histogram now reads every visible pixel. This fixes missed accents and
periodic colors, but increases scan work on large, low-color sources. These
local measurements compare the previous nonempty fixed-stride path with the
full-source histogram, requesting 16 colors. Values are medians of three runs
on Apple M3 Pro with Go 1.27.0; fixture creation is outside the timed region.

| Source | Previous time | Full-source time | Previous allocated bytes/op | Full-source allocated bytes/op |
|---|---:|---:|---:|---:|
| 1,024², 16-color pattern | 2.183 ms | 11.590 ms | 824 | 5,800 |
| 1,024², high color diversity | 208.720 ms | 28.789 ms | 24,163,432 | 27,965,040 |
| 4,096², 16-color pattern | 4.124 ms | 188.502 ms | 96 | 5,800 |
| 4,096², high color diversity | 176.237 ms | 289.848 ms | 24,163,416 | 37,795,328 |

The old sampler saw only four of the 16 pattern colors at 1,024² and one at
4,096². The new histogram includes them all. High-diversity overflow can reduce
median-cut work by combining nearby colors, while requiring a complete scan.
These measurements describe these fixtures, not a general throughput promise.

Allocated bytes are total allocations per operation, not peak resident memory.
The active histogram has at most 262,144 distinct keys. During overflow, the old
map, fixed 8 MiB bin accumulator, and output storage can temporarily coexist.
Tests independently check the exact-cap transition, original weighted sums,
64-bit weight bounds, cancellation, and unchanged small-image encoder output.
