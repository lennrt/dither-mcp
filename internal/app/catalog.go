package app

import (
	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/colorprofile"
	"github.com/lennrt/dither-mcp/internal/imagemeta"
)

// CatalogResult describes accepted formats and limits for each media operation.
// Existing top-level format lists describe still images. Media adds explicit roles.
type CatalogResult struct {
	Version           string             `json:"version"`
	Algorithms        []engine.Algorithm `json:"algorithms"`
	PaletteCount      int                `json:"palette_count"`
	PaletteCategories []PaletteCategory  `json:"palette_categories"`
	PaletteDiscovery  PaletteDiscovery   `json:"palette_discovery"`
	InputFormats      []string           `json:"input_formats" jsonschema:"These formats provide still-image inputs. Render uses the first image or GIF frame."`
	OutputFormats     []string           `json:"output_formats" jsonschema:"These formats provide still-image outputs. Media lists animation, video, and print outputs separately."`
	AnimationEffects  []string           `json:"animation_effects"`
	VideoEnabled      bool               `json:"video_enabled" jsonschema:"This reports the startup opt-in setting. Video also requires installed ffmpeg and ffprobe executables."`
	Limits            CatalogLimits      `json:"limits"`
	Defaults          Recipe             `json:"defaults"`
	Paths             string             `json:"paths"`
	Media             MediaCapabilities  `json:"media" jsonschema:"Each section lists accepted inputs, output roles, defaults, and limits for a media operation."`
	Studio            StudioCapabilities `json:"studio" jsonschema:"This read-only preview operation can open an optional MCP Apps interface in supporting hosts."`
}

type StudioCapabilities struct {
	Tool                string `json:"tool"`
	DefaultMaxDimension int    `json:"default_max_dimension"`
	MaxDimension        int    `json:"max_dimension"`
	MaxPNGBytes         int    `json:"max_png_bytes"`
	WritesFiles         bool   `json:"writes_files"`
	SaveTool            string `json:"save_tool"`
}

type PaletteDiscovery struct {
	DefaultLimit int    `json:"default_limit"`
	MaxLimit     int    `json:"max_limit"`
	Order        string `json:"order"`
	Query        string `json:"query"`
}

type CatalogLimits struct {
	MaxFileBytes   int64 `json:"max_file_bytes"`
	MaxImagePixels int   `json:"max_image_pixels"`
	MaxFramePixels int   `json:"max_frame_pixels"`
	MaxFrames      int   `json:"max_frames"`
	MaxBatch       int   `json:"max_batch"`
	MaxCompare     int   `json:"max_compare"`
	TimeoutSeconds int   `json:"timeout_seconds"`
}

type MediaCapabilities struct {
	Still     StillCapabilities     `json:"still"`
	Animation AnimationCapabilities `json:"animation"`
	Video     VideoCapabilities     `json:"video"`
	Print     PrintCapabilities     `json:"print"`
}

type FormatRole struct {
	Format   string `json:"format"`
	MIMEType string `json:"mime_type"`
	Role     string `json:"role" jsonschema:"This describes what the format stores and how the operation uses it."`
	Alpha    string `json:"alpha" jsonschema:"This describes transparency handling for this output format."`
}

type StillCapabilities struct {
	InputFormats  []string                  `json:"input_formats"`
	OutputFormats []string                  `json:"output_formats"`
	FramePolicy   string                    `json:"frame_policy"`
	FormatRoles   []FormatRole              `json:"format_roles"`
	Normalization NormalizationCapabilities `json:"normalization"`
}

type NormalizationCapabilities struct {
	OrientationFormats []string `json:"orientation_formats"`
	ICCFormats         []string `json:"icc_formats"`
	WorkingSpace       string   `json:"working_space"`
	Profiles           string   `json:"profiles"`
	Order              string   `json:"order"`
	Untagged           string   `json:"untagged"`
	PNGPrecedence      []string `json:"png_precedence"`
	OutputMetadata     string   `json:"output_metadata"`
	MaxICCBytes        int      `json:"max_icc_bytes"`
	MaxEXIFBytes       int      `json:"max_exif_bytes"`
	MaxICCTags         int      `json:"max_icc_tags"`
	MaxCurveSamples    int      `json:"max_curve_samples"`
}

type AnimationCapabilities struct {
	InputFormats    []string     `json:"input_formats"`
	OutputFormats   []string     `json:"output_formats"`
	Effects         []string     `json:"effects"`
	DefaultFrames   int          `json:"default_frames"`
	DefaultFPS      int          `json:"default_fps"`
	DefaultWidth    int          `json:"default_width" jsonschema:"This width applies when both requested dimensions are omitted or zero."`
	MaxFrames       int          `json:"max_frames"`
	MaxFPS          int          `json:"max_fps"`
	SourceGIFTiming bool         `json:"source_gif_timing" jsonschema:"The source effect preserves GIF frame delays and loop count. Source frame count replaces the requested frame count."`
	FormatRoles     []FormatRole `json:"format_roles"`
}

type VideoCapabilities struct {
	Enabled             bool         `json:"enabled"`
	RequiredExecutables []string     `json:"required_executables"`
	InputContainers     []string     `json:"input_containers"`
	OutputFormats       []string     `json:"output_formats"`
	DefaultFrames       int          `json:"default_frames"`
	DefaultFPS          int          `json:"default_fps"`
	DefaultWidth        int          `json:"default_width" jsonschema:"This width applies when both requested dimensions are omitted or zero."`
	MaxFrames           int          `json:"max_frames"`
	MaxFPS              int          `json:"max_fps"`
	MaxStartSeconds     int          `json:"max_start_seconds"`
	Stream              string       `json:"stream"`
	Audio               bool         `json:"audio"`
	CodecAvailability   string       `json:"codec_availability"`
	InputRequirements   string       `json:"input_requirements"`
	FormatRoles         []FormatRole `json:"format_roles"`
}

type PrintCapabilities struct {
	OutputFormat    string `json:"output_format"`
	PlateFormat     string `json:"plate_format"`
	MaxInks         int    `json:"max_inks"`
	DefaultDPI      int    `json:"default_dpi" jsonschema:"Zero means unspecified. The service adds no DPI metadata unless the request supplies a value."`
	MinDPI          int    `json:"min_dpi"`
	MaxDPI          int    `json:"max_dpi"`
	Masks           bool   `json:"masks"`
	PostEffects     bool   `json:"post_effects"`
	PressCalibrated bool   `json:"press_calibrated"`
	Role            string `json:"role"`
}

func (s *Service) Catalog() CatalogResult {
	input := []string{"png", "jpeg", "gif", "webp", "bmp", "tiff"}
	output := []string{"png", "jpeg", "gif", "svg", "pbm", "ascii"}
	effects := []string{"source", "wave", "orbit", "pulse", "noise", "palette-cycle"}
	palettes := engine.Palettes()
	return CatalogResult{
		Version: Version, Algorithms: engine.Catalog(), PaletteCount: len(palettes), PaletteCategories: paletteCategories(palettes),
		PaletteDiscovery: PaletteDiscovery{DefaultPaletteLimit, MaxPaletteLimit, "id-ascending", "case-insensitive whitespace-separated AND terms"},
		InputFormats:     input, OutputFormats: output, AnimationEffects: effects, VideoEnabled: s.allowVideo,
		Limits:   CatalogLimits{MaxBytes, engine.MaxPixels, MaxFramePixels, MaxFrames, 32, 12, 120},
		Defaults: Recipe{Version: 1, Palette: "mono", Options: engine.DefaultConfig()},
		Paths:    "Paths are relative to the configured root. The service rejects URLs, absolute paths, and parent traversal. Outputs use new paths without overwriting files.",
		Studio:   StudioCapabilities{Tool: "dither_studio", DefaultMaxDimension: StudioDefaultDimension, MaxDimension: StudioMaxDimension, MaxPNGBytes: StudioMaxPNGBytes, WritesFiles: false, SaveTool: "dither_render"},
		Media: MediaCapabilities{
			Still: StillCapabilities{
				InputFormats: append([]string(nil), input...), OutputFormats: append([]string(nil), output...),
				FramePolicy: "Render uses the first decoded image. GIF uses its first frame on the logical canvas. TIFF uses its first page. PNG and APNG use the default image. Animated WebP processing is unsupported.",
				Normalization: NormalizationCapabilities{
					OrientationFormats: []string{"jpeg", "png", "webp", "tiff"},
					ICCFormats:         []string{"jpeg", "png", "webp", "tiff", "bmp-v5"},
					WorkingSpace:       "srgb",
					Profiles:           "ICC v2/v4 RGB matrix/TRC with XYZ PCS, curv or para types 0 through 4. Input, display, and color-space classes are supported. LUT, CMYK, grayscale ICC, Lab, and device-link profiles return errors. BMP linked and calibrated profiles and GIF ICC extensions are unsupported.",
					Order:              "Normalize all eight EXIF orientations and convert supported input color declarations before crop, resize, palette extraction, and masks. Inspection dimensions use upright coordinates.",
					Untagged:           "RGB and grayscale samples without a color declaration assume sRGB. CMYK input requires prior conversion to sRGB.",
					PNGPrecedence:      []string{"cICP (sRGB 1/13/0/1 only)", "iCCP", "sRGB", "gAMA/cHRM", "assumed-srgb"},
					OutputMetadata:     "Output colors use sRGB values. Source EXIF and ICC metadata are omitted. PNG and JPEG outputs have no embedded output profile.",
					MaxICCBytes:        colorprofile.MaxProfileBytes, MaxEXIFBytes: imagemeta.MaxEXIFBytes,
					MaxICCTags: colorprofile.MaxTags, MaxCurveSamples: colorprofile.MaxCurveSamples,
				},
				FormatRoles: []FormatRole{
					{"png", "image/png", "Lossless raster image. This format also supports optional print DPI metadata.", "Preserves full alpha."},
					{"jpeg", "image/jpeg", "Lossy raster image at quality 95. The jpg name is an alias.", "Composites against white."},
					{"gif", "image/gif", "Indexed raster image with at most 256 palette slots.", "Alpha below 128 becomes transparent. Other alpha becomes opaque."},
					{"svg", "image/svg+xml", "Vector paths preserve pixel runs with crisp edges. This stores raster coverage rather than smooth image tracing.", "Preserves full alpha."},
					{"pbm", "image/x-portable-bitmap", "Binary P4 bilevel raster. Luminance below 0.5 becomes black.", "Composites against white."},
					{"ascii", "text/plain", "Text uses one luminance symbol per pixel without terminal aspect-ratio correction.", "Composites against white."},
				},
			},
			Animation: AnimationCapabilities{
				InputFormats: append([]string(nil), input...), OutputFormats: []string{"gif", "png"}, Effects: append([]string(nil), effects...),
				DefaultFrames: 24, DefaultFPS: 12, DefaultWidth: 480, MaxFrames: MaxFrames, MaxFPS: 50, SourceGIFTiming: true,
				FormatRoles: []FormatRole{
					{"gif", "image/gif", "Animated GIF. The source effect preserves input GIF delays and loop count.", "Uses binary alpha."},
					{"png", "image/png", "A static sprite sheet stores each processed frame in a grid.", "Preserves full alpha."},
				},
			},
			Video: VideoCapabilities{
				Enabled: s.allowVideo, RequiredExecutables: []string{"ffmpeg", "ffprobe"},
				InputContainers: []string{"mp4", "mov", "webm", "matroska", "avi"}, OutputFormats: []string{"mp4", "webm", "gif", "png"},
				DefaultFrames: 48, DefaultFPS: 12, DefaultWidth: 480, MaxFrames: MaxFrames, MaxFPS: 50, MaxStartSeconds: 86400,
				Stream: "first-video-stream", Audio: false,
				CodecAvailability: "Input codecs depend on the installed FFmpeg build. MP4 output requires libx264. WebM output requires libvpx-vp9.",
				InputRequirements: "MP4 and MOV inputs require an ftyp header. WebM and Matroska require an EBML header. AVI requires a RIFF AVI header.",
				FormatRoles: []FormatRole{
					{"mp4", "video/mp4", "H.264 video uses libx264, CRF 18, and yuv420p. Odd dimensions pad to even dimensions. Audio is omitted.", "Does not preserve alpha."},
					{"webm", "video/webm", "VP9 video uses libvpx-vp9, CRF 24, and yuv420p. Odd dimensions pad to even dimensions. Audio is omitted.", "Does not preserve alpha."},
					{"gif", "image/gif", "Animated GIF stores the processed video frames at the requested frame rate.", "Uses binary alpha."},
					{"png", "image/png", "A static sprite sheet stores the processed video frames in a grid.", "Preserves full alpha."},
				},
			},
			Print: PrintCapabilities{
				OutputFormat: "zip", PlateFormat: "png", MaxInks: 16, DefaultDPI: 0, MinDPI: 36, MaxDPI: 2400,
				Masks: false, PostEffects: false, PressCalibrated: false,
				Role: "The ZIP contains one black-on-white PNG coverage plate per palette ink and a JSON manifest. Plates do not include press calibration.",
			},
		},
	}
}
