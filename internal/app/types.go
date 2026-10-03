// Package app implements the bounded operations that the MCP server and CLI
// share. It resolves paths within a workspace root. The engine is independent
// of this package.
package app

import (
	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/imagemeta"
)

const Version = "0.1.0-dev"
const MaxBytes int64 = 32 << 20
const MaxFramePixels = 64 * 1024 * 1024
const MaxFrames = 120

type Empty struct{}
type InputRequest struct {
	Input string `json:"input" jsonschema:"Read this local file relative to the configured workspace. Recipe loading expects JSON. Image operations accept formats from dither_catalog."`
}
type ExtractRequest struct {
	Input string `json:"input" jsonschema:"Read this image relative to the configured workspace. Extraction uses its first decoded image."`
	Count int    `json:"count,omitempty" jsonschema:"Request 2 through 256 colors. Omission or zero selects 8. An image with fewer distinct colors returns fewer colors."`
}

// Recipe uses a versioned envelope. The version prevents future changes from
// silently reinterpreting an existing recipe. It contains no input or output paths.
type Recipe struct {
	Version int           `json:"version" jsonschema:"The recipe version must be 1."`
	Palette string        `json:"palette,omitempty" jsonschema:"Select a built-in palette ID from dither_palettes. Omit colors when this field is present. Omission selects mono unless colors supplies a palette."`
	Colors  []string      `json:"colors,omitempty" jsonschema:"Supply 2 through 256 unique opaque hex colors instead of palette. Each color accepts RGB or RRGGBB with an optional leading #."`
	Options engine.Config `json:"options" jsonschema:"These engine settings define a reusable processing recipe without input or output paths."`
}
type RecipeSaveRequest struct {
	Output string `json:"output" jsonschema:"Create this JSON file relative to the workspace. The path must not exist."`
	Recipe Recipe `json:"recipe" jsonschema:"Validate and save this version-1 recipe. The service preserves image masks without storing their source paths."`
}
type RenderRequest struct {
	Input     string        `json:"input" jsonschema:"Read this local source relative to the workspace. Video expects a supported container. Other image operations use dither_catalog input_formats."`
	Output    string        `json:"output" jsonschema:"Create this output relative to the workspace. The path must not exist. Its extension selects the format unless format overrides it."`
	Recipe    string        `json:"recipe,omitempty" jsonschema:"Load this saved JSON recipe relative to the workspace. A saved recipe must be used without inline options, palette, or colors."`
	Palette   string        `json:"palette,omitempty" jsonschema:"Choose a built-in palette ID. Use dither_palettes to discover IDs. Omit colors when this field is present."`
	Colors    []string      `json:"colors,omitempty" jsonschema:"Supply 2 through 256 unique opaque hex colors instead of palette. Each color accepts RGB or RRGGBB with an optional leading #."`
	Options   engine.Config `json:"options,omitempty" jsonschema:"Apply these engine settings. A saved recipe cannot be combined with inline options."`
	MaskInput string        `json:"mask_input,omitempty" jsonschema:"Read and normalize this local mask relative to the workspace. EXIF orientation and supported color declarations apply before luminance times alpha >= 0.5 selects pixels on the reduced grid."`
	Format    string        `json:"format,omitempty" jsonschema:"Override the output extension. Still output supports png, jpeg, gif, svg, pbm, or ascii. Animation supports gif or png. Video supports mp4, webm, gif, or png. Separations use zip."`
	DPI       int           `json:"dpi,omitempty" jsonschema:"Set PNG print metadata from 36 through 2400 DPI. Omission or zero leaves metadata unspecified. Animation and video require zero."`
}
type CompareRequest struct {
	RenderRequest
	Algorithms []string `json:"algorithms" jsonschema:"Compare 1 through 12 algorithm IDs from dither_catalog. Each cell uses the same source, palette, and engine settings."`
	Columns    int      `json:"columns,omitempty" jsonschema:"Use 1 through 12 columns in the contact sheet. Omission or zero selects 3."`
}
type BatchRequest struct {
	Items []RenderRequest `json:"items" jsonschema:"Process 1 through 32 render requests in sequence. Each successful item creates its own output. The result reports individual failures."`
}
type AnimateRequest struct {
	RenderRequest
	Effect    string  `json:"effect,omitempty" jsonschema:"Use source, wave, orbit, pulse, noise, or palette-cycle. Omission selects source for GIF input and wave for other images."`
	Frames    int     `json:"frames,omitempty" jsonschema:"Generate 1 through 120 frames. Omission or zero selects 24. The source effect uses every input GIF frame instead of this value."`
	FPS       int     `json:"fps,omitempty" jsonschema:"Use 1 through 50 frames per second. Omission or zero selects 12. The source effect preserves input GIF delays."`
	Amplitude float64 `json:"amplitude,omitempty" jsonschema:"Set motion strength from 0 through 1. Omission or zero selects 0.15."`
	Columns   int     `json:"columns,omitempty" jsonschema:"Use 1 through 16 columns for PNG sprite sheets. Omission or zero selects 6. GIF output ignores this value."`
}
type VideoRequest struct {
	RenderRequest
	FPS    int     `json:"fps,omitempty" jsonschema:"Sample 1 through 50 frames per second. Omission or zero selects 12."`
	Frames int     `json:"frames,omitempty" jsonschema:"Set a limit of 1 through 120 processed frames. Omission or zero selects 48. A shorter input segment can return fewer frames."`
	Start  float64 `json:"start,omitempty" jsonschema:"Start the local video segment at this time in seconds, from 0 through 86400. Omission starts at zero."`
}
type Artifact struct {
	Operation     string         `json:"operation,omitempty"`
	Parameters    map[string]any `json:"parameters,omitempty"`
	EngineVersion string         `json:"engine_version,omitempty"`
	Path          string         `json:"path"`
	Format        string         `json:"format"`
	Width         int            `json:"width,omitempty"`
	Height        int            `json:"height,omitempty"`
	Frames        int            `json:"frames,omitempty"`
	Bytes         int            `json:"bytes"`
	SHA256        string         `json:"sha256"`
	Recipe        *Recipe        `json:"recipe,omitempty"`
}
type Inspection struct {
	Normalization imagemeta.Info `json:"normalization" jsonschema:"Input orientation and color handling applied before image operations. Width and height report the upright image."`
	Path          string         `json:"path"`
	Format        string         `json:"format"`
	Width         int            `json:"width"`
	Height        int            `json:"height"`
	Frames        int            `json:"frames"`
	Bytes         int            `json:"bytes"`
	SHA256        string         `json:"sha256"`
	HasAlpha      bool           `json:"has_alpha"`
}
type BatchItem struct {
	Index    int       `json:"index"`
	Artifact *Artifact `json:"artifact,omitempty"`
	Error    string    `json:"error,omitempty"`
}
type BatchResult struct {
	Items []BatchItem `json:"items"`
}
type CompareResult struct {
	Artifact Artifact `json:"artifact"`
	Cells    []Cell   `json:"cells"`
}
type Cell struct {
	Algorithm string `json:"algorithm"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}
