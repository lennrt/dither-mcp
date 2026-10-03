// Package engine implements deterministic, in-memory image dithering.
// The engine does not read files, start processes, or access the network.
package engine

import (
	"context"
	"image"
	"image/color"
	"io"
)

// MaxPixels bounds both input and output dimensions before any image-sized
// allocation. The service may impose a lower limit for untrusted input.
const MaxPixels = 16_777_216
const MaxDimension = 16_384

// Config is a complete, reproducible processing recipe.
// Nil adjustment pointers select neutral defaults. Explicit zero remains meaningful.
// Width and Height specify output dimensions before pixel-grid reduction.
//
// PixelScale reduces the image to ceil(width/scale) x ceil(height/scale).
// The engine dithers this smaller grid. Nearest-neighbor sampling then restores
// the exact requested dimensions. Mask coordinates refer to the smaller grid.
type Config struct {
	Algorithm    string        `json:"algorithm,omitempty" jsonschema:"Choose an algorithm ID from dither_catalog. Omission selects floyd-steinberg."`
	Palette      []color.NRGBA `json:"-"`
	Width        int           `json:"width,omitempty" jsonschema:"Set output width from 0 through 16384 pixels. Zero preserves source size or the aspect ratio when height is specified."`
	Height       int           `json:"height,omitempty" jsonschema:"Set output height from 0 through 16384 pixels. Zero preserves source size or the aspect ratio when width is specified."`
	PixelScale   int           `json:"pixel_scale,omitempty" jsonschema:"Set grid scale from 0 through 256. Zero or one keeps the full grid. Larger values reduce the grid before dithering and enlarge it afterward."`
	ResizeFilter string        `json:"resize_filter,omitempty" jsonschema:"Use bilinear or nearest sampling. Omission selects bilinear interpolation with correct alpha handling."`
	Brightness   float64       `json:"brightness,omitempty" jsonschema:"Add an sRGB channel offset from -1 through 1. The default is 0."`
	Contrast     *float64      `json:"contrast,omitempty" jsonschema:"Set contrast from 0 through 4 around the channel midpoint 0.5. Omission selects 1. Zero produces a flat midpoint."`
	Gamma        *float64      `json:"gamma,omitempty" jsonschema:"Set gamma from 0.1 through 8. The engine applies channel^(1/gamma). Omission selects 1."`
	Saturation   *float64      `json:"saturation,omitempty" jsonschema:"Set saturation from 0 through 4. Omission selects 1. Zero produces grayscale."`
	Threshold    *float64      `json:"threshold,omitempty" jsonschema:"Set threshold from 0 through 1. Omission selects 0.5. Higher values darken the quantization vector."`
	Strength     *float64      `json:"strength,omitempty" jsonschema:"Set dither strength from 0 through 2. Omission selects 1. Zero selects nearest-color quantization for ordered patterns."`
	Seed         int64         `json:"seed,omitempty" jsonschema:"Use this integer seed for reproducible noise and moving effects. The default is 0."`
	Serpentine   bool          `json:"serpentine,omitempty" jsonschema:"Reverse diffusion scan direction on alternate rows. The default is false."`
	Invert       bool          `json:"invert,omitempty" jsonschema:"Invert RGB channels before dithering. Alpha remains unchanged."`
	Grayscale    bool          `json:"grayscale,omitempty" jsonschema:"Convert RGB to grayscale before dithering. Alpha remains unchanged."`
	ColorSpace   string        `json:"color_space,omitempty" jsonschema:"Use srgb or linear-rgb for palette distance and diffusion error. Omission selects srgb."`
	Crop         *Rect         `json:"crop,omitempty" jsonschema:"Crop within source bounds before resizing. Service inputs use upright coordinates after EXIF orientation, with a top-left origin."`
	Mask         *Mask         `json:"mask,omitempty" jsonschema:"Dither only selected pixels on the reduced grid. Pixels outside the selection retain their adjusted source values."`
	Effects      Effects       `json:"effects,omitempty" jsonschema:"Apply these effects after dithering. Shading and noise may produce colors outside the palette."`
}

// Rect uses source-image coordinates, relative to its Bounds().Min.
type Rect struct {
	X      int `json:"x" jsonschema:"Set the nonnegative horizontal crop origin within the source."`
	Y      int `json:"y" jsonschema:"Set the nonnegative vertical crop origin within the source."`
	Width  int `json:"width" jsonschema:"Set a positive crop width that fits wholly within the source."`
	Height int `json:"height" jsonschema:"Set a positive crop height that fits wholly within the source."`
}

// Mask selects pixels to dither. Rectangle coordinates use a top-left origin.
// Circle X and Y specify its center. The engine scales image masks to the
// processing grid with nearest-neighbor sampling. These masks select pixels
// whose luminance times alpha is >= 0.5. Pixels outside the selection retain
// their adjusted source values.
//
// Diffusion remains within the selection and cannot cross transparent pixels.
// The service supplies Mask.Image in memory. Recipes cannot serialize this field.
type Mask struct {
	Shape  string      `json:"shape" jsonschema:"Use rectangle, circle, or image. Image masks require a separate mask_input path in the request."`
	X      int         `json:"x,omitempty" jsonschema:"Set the rectangle left edge or circle center on the reduced grid."`
	Y      int         `json:"y,omitempty" jsonschema:"Set the rectangle top edge or circle center on the reduced grid."`
	Width  int         `json:"width,omitempty" jsonschema:"Set a positive rectangle width in reduced-grid pixels."`
	Height int         `json:"height,omitempty" jsonschema:"Set a positive rectangle height in reduced-grid pixels."`
	Radius int         `json:"radius,omitempty" jsonschema:"Set a circle radius from 1 through 16384 reduced-grid pixels."`
	Invert bool        `json:"invert,omitempty" jsonschema:"Dither the complement of this selection. The default is false."`
	Image  image.Image `json:"-"`
}

// The engine applies Effects after dithering. They may produce colors outside the palette.
// Scanlines/CRT/Noise are strengths in [0,1]. Glitch is a maximum horizontal
// shift in pixels, with seeded row bands. PixelSort sorts contiguous luminance
// runs above PixelSortThreshold (default 0.25), preserving their alpha.
type Effects struct {
	Scanlines          float64  `json:"scanlines,omitempty" jsonschema:"Set scanline shading strength from 0 through 1."`
	ScanlineSpacing    int      `json:"scanline_spacing,omitempty" jsonschema:"Set scanline spacing from 0 through 256 pixels. Omission or zero selects 2."`
	CRT                float64  `json:"crt,omitempty" jsonschema:"Set CRT shading strength from 0 through 1."`
	Noise              float64  `json:"noise,omitempty" jsonschema:"Set seeded additive noise strength from 0 through 1."`
	Glitch             int      `json:"glitch,omitempty" jsonschema:"Set the maximum seeded horizontal row-band shift from 0 through 1024 pixels."`
	PixelSort          bool     `json:"pixel_sort,omitempty" jsonschema:"Sort visible contiguous luminance runs above pixel_sort_threshold."`
	PixelSortThreshold *float64 `json:"pixel_sort_threshold,omitempty" jsonschema:"Set the pixel-sorting threshold from 0 through 1. Omission selects 0.25."`
}

// Algorithm describes a distinct implementation, not an alias.
type Algorithm struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Family      string `json:"family"`
	Description string `json:"description"`
}

// Palette is a named collection of opaque sRGB colors in hexadecimal notation.
type Palette struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Colors      []string `json:"colors"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Origin      string   `json:"origin"`
	Source      string   `json:"source,omitempty"`
}

// Float constructs an optional scalar for Config.
func Float(v float64) *float64 { return &v }

// DefaultConfig returns the default Floyd-Steinberg monochrome recipe.
func DefaultConfig() Config { return Config{Algorithm: "floyd-steinberg"} }

// Encode supports PNG, JPEG, GIF, SVG, PBM, and ASCII.
// The format is a lowercase name without a dot. The names jpeg and jpg are equivalent.
// Encode writes to the caller's writer. The caller remains responsible for closing it.
func Encode(format string, w io.Writer, img image.Image) error { return encode(format, w, img) }

// Process preserves its input and configuration. It returns a new NRGBA image
// with zero-origin bounds and retains source alpha. Fully transparent source
// pixels become transparent black. Process dithers partially transparent pixels
// using their unassociated RGB and retains their exact alpha values.
// Post-effects preserve alpha but may move pixels when glitch or pixel sorting is enabled.
func Process(ctx context.Context, img image.Image, cfg Config) (*image.NRGBA, error) {
	return process(ctx, img, cfg)
}
