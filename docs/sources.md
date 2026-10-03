# Sources and inspiration

We researched the initial specification on 2026-10-03. Project descriptions can evolve. The implemented catalog and [feature matrix](feature-matrix.md) define this project's scope. We do not present reference UIs or sample artwork as our own. We do not imply affiliation or endorsement.

## Creative references

| Source | What informed the design | Evidence used |
|---|---|---|
| [sergedoub/dither-studio](https://github.com/sergedoub/dither-studio) | Broad algorithm catalog, local processing, palettes/extraction, DPI, ink masks, batch and portable settings | Public repository README, fetched directly |
| [Dither Studio by zed2101](https://zed2101.github.io/dither-studio/) | Approachable retro palette and classic dither vocabulary | Project description. Site contents were unavailable to the research browser |
| [Dither Studio by kcvete](https://kcvete.github.io/dither-studio/) | Video and region-specific processing as creative workflows | Project description. Camera/tracking controls are explicitly outside our release |
| [Noise by kjellr](https://kjellr.github.io/noise/) | Classic/modern algorithm exploration | Project description. No independently verified parity claim |
| [Dither Studio by Jonas Sigrist](https://dithering.sigrist.dev/) | Ordered/diffusion algorithms and custom palettes | Live entry page plus project description |
| [Laser Dither Studio](https://laperiut.github.io/dither-studio/dither-studio.html) | Local print/laser preparation | Project description. This project does not control laser hardware |
| [ditherit](https://ditherit-rho.vercel.app/) | Threshold, contrast, gamma, error strength, serpentine traversal, SVG/ASCII workflows | Live page controls inspected. This release excludes learned background removal, pointer repulsion, and code export |
| [Animated Dither Studio](https://dither-studio-pi.vercel.app/) | Still-to-loop, GIF/video/sprite output vocabulary | Project description. Independently implemented finite animation modes |
| [gyng/ditherer](https://github.com/gyng/ditherer) / [live tool](https://gyng.github.io/ditherer/) | Filter-chain ideas, retro effects, palettes, finite video export | Repository README fetched directly. Its much broader live audio/3D/GPU catalog is not claimed as parity |
| [Dither Studios](https://dither-studios.vercel.app/) | Simple upload-adjust-export creative workflow | Project description. No independent quality/performance equivalence claim |
| [Turbo Dither palettes](https://www.turbodither.com/palettes) | Retro color-library reference for the subsequent palette expansion | Live palette index inspected on 2026-10-03. 15 listed entries informed the library breadth target |

The project descriptions provided inspiration. They do not establish that the named sites share an implementation or current features. The sergedoub README describes limitations and distinguishes approximations from exact replacements. That clarity informed this project's documentation.

The palette expansion uses a separate [OpenSpec change](../openspec/changes/palette-library/proposal.md). Its 256 presets span 16 categories: 239 original designs, seven references, and ten historical/display approximations. We authored the new original palettes for this library. Referenced and approximate historical palettes carry their source and origin in the embedded catalog.

The [palette guide](palettes.md) exposes that provenance alongside every color array. Catalog size measures breadth. Validation also checks unique unordered color sets, complete metadata, and unchanged colors for the original 22 presets.

## Quality references

[TempestKeep](https://github.com/lennrt/tempestkeep) informed the one-binary stdio MCP presentation, structured capabilities, usable local CLI, realistic synthetic demonstration, and explicit operational limits. [trial-lang](https://github.com/lennrt/trial-lang) informed the cohesive visual identity, executable examples, recorded demos, and attention to specifying behavior before presenting it as reliable. We fetched both READMEs directly from their public repositories.

The visual style draws on [ka2a.dev](https://ka2a.dev): warm paper surfaces, serif typography, dark ink, and a restrained clay accent. The dither-mcp banner and image treatments use this project's original artwork. The website bundles its fonts with their [license notices](../THIRD_PARTY_NOTICES.md).

## Technical primary sources

- [OpenSpec repository and workflow](https://github.com/Fission-AI/OpenSpec): proposal/design/tasks, baseline specifications, change deltas, strict validation. `package.json` and `pnpm-lock.yaml` pin the checked tool version.
- [Quint getting started](https://quint.sh/docs/getting-started) and [source repository](https://github.com/quint-co/quint): executable action specifications, run tests, invariant simulation, and the distinction between simulation and model checking.
- [MCP tools specification](https://modelcontextprotocol.io/specification/2026-07-28/server/tools): tool discovery, schemas, structured results, image content, and annotations. The [MCP guide](mcp.md) identifies the tested current and legacy protocol paths.
- [mcp-go](https://github.com/mark3labs/mcp-go): the Go MCP SDK used by this implementation. This is the `mark3labs/mcp-go` project, selected for the requested Go MCP implementation.
- [Go traversal-resistant file APIs](https://go.dev/blog/osroot): `os.Root`, symlink/traversal races, and native-platform limitations.
- [Go image/gif package](https://pkg.go.dev/image/gif): GIF data structures, timing, palette and disposal fields.
- [Go image/color/palette package](https://pkg.go.dev/image/color/palette): exact Plan 9 and WebSafe palette references. The Go tests compare the embedded entries with those standard-library tables.
- [GIF89a specification](https://giflib.sourceforge.net/gifstandard/GIF89a.html): logical screen, global background, graphic-control transparency, and disposal semantics.
- [Charmbracelet VHS](https://github.com/charmbracelet/vhs): reproducible terminal recordings through tape scripts.

See the [engine source notes](../engine/SOURCES.md) for kernel matrices, screen construction, historical naming, palette provenance, and measured performance. Algorithm identities describe our documented implementations. Historical labels alone do not guarantee agreement with every other renderer.
