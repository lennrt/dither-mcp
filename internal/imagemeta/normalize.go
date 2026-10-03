// Package imagemeta normalizes local still images before image operations.
// It reads only embedded metadata. It never opens an external profile or URL.
package imagemeta

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/lennrt/dither-mcp/internal/colorprofile"
)

const MaxEXIFBytes = 4 << 20
const MaxIFDEntries = 4096

// Info describes the normalization that precedes crop, resize, and sampling.
type Info struct {
	StoredWidth        int    `json:"stored_width" jsonschema:"Width in the encoded image before EXIF orientation."`
	StoredHeight       int    `json:"stored_height" jsonschema:"Height in the encoded image before EXIF orientation."`
	EXIFOrientation    int    `json:"exif_orientation" jsonschema:"Source orientation from 1 through 8. Missing orientation is 1."`
	OrientationApplied bool   `json:"orientation_applied"`
	ColorSource        string `json:"color_source" jsonschema:"The selected source color declaration, or assumed-srgb when none is present."`
	WorkingSpace       string `json:"working_space" jsonschema:"Normalized pixels use srgb. Palette colors also use srgb."`
	ColorConverted     bool   `json:"color_converted" jsonschema:"True when a color transform ran before image operations."`
	ICCSHA256          string `json:"icc_sha256,omitempty" jsonschema:"SHA-256 of the selected embedded ICC profile. The service does not return profile text or other EXIF fields."`
}

// Metadata holds a validated transform and orientation for one image.
type Metadata struct {
	orientation int
	hasEXIF     bool
	icc         []byte
	srgb        bool
	cicp        bool
	gamma       float64
	chromas     *[8]float64
	source      string
	transform   *colorprofile.Transform
}

// Parse validates recognized metadata before full pixel decoding.
func Parse(ctx context.Context, b []byte, format string) (*Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m := &Metadata{orientation: 1, source: "assumed-srgb"}
	var err error
	switch format {
	case "jpeg":
		err = m.jpeg(ctx, b)
	case "png":
		err = m.png(ctx, b)
	case "webp":
		err = m.webp(ctx, b)
	case "tiff":
		err = m.tiff(b, true)
	case "bmp":
		err = m.bmp(b)
	case "gif":
		err = checkGIF(ctx, b)
	default:
		err = errors.New("unsupported metadata container")
	}
	if err != nil {
		return nil, fmt.Errorf("%s metadata: %w", format, err)
	}
	if m.cicp {
		m.source = "png-cicp-srgb"
		m.icc = nil
	} else if len(m.icc) != 0 {
		m.source = "embedded-icc"
		m.transform, err = colorprofile.Parse(m.icc)
	} else if m.srgb {
		m.source = format + "-srgb"
	} else if m.gamma != 0 || m.chromas != nil {
		m.source = "png-gamma-chromaticities"
		m.transform, err = colorprofile.FromPNG(m.gamma, m.chromas)
	}
	if err != nil {
		return nil, fmt.Errorf("input color profile: %w. Convert the source to sRGB with a color-managed editor", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// Apply preserves source alpha and leaves untagged RGB sample values unchanged.
// The returned image uses display coordinates with its origin at zero.
func (m *Metadata) Apply(ctx context.Context, src image.Image) (image.Image, Info, error) {
	info := Info{
		StoredWidth: src.Bounds().Dx(), StoredHeight: src.Bounds().Dy(),
		EXIFOrientation: m.orientation, OrientationApplied: m.orientation != 1,
		ColorSource: m.source, WorkingSpace: "srgb", ColorConverted: m.transform != nil,
	}
	if err := ctx.Err(); err != nil {
		return nil, info, err
	}
	if _, ok := src.(*image.CMYK); ok {
		return nil, info, errors.New("CMYK input is unsupported. Convert the source to sRGB with a color-managed editor")
	}
	if len(m.icc) != 0 {
		switch src.(type) {
		case *image.Gray, *image.Gray16:
			return nil, info, errors.New("an RGB ICC profile cannot describe a grayscale image")
		}
		h := sha256.Sum256(m.icc)
		info.ICCSHA256 = hex.EncodeToString(h[:])
	}
	var err error
	if m.transform != nil {
		src, err = m.transform.Apply(ctx, src)
		if err != nil {
			return nil, info, err
		}
	}
	if m.orientation != 1 || src.Bounds().Min != (image.Point{}) {
		w, h := src.Bounds().Dx(), src.Bounds().Dy()
		if m.orientation >= 5 {
			w, h = h, w
		}
		src = &oriented{src: src, orientation: m.orientation, bounds: image.Rect(0, 0, w, h)}
	}
	return src, info, ctx.Err()
}

// An oriented view avoids allocating another full image for lossless geometry.
type oriented struct {
	src         image.Image
	orientation int
	bounds      image.Rectangle
}

func (o *oriented) Bounds() image.Rectangle { return o.bounds }
func (o *oriented) ColorModel() color.Model { return o.src.ColorModel() }
func (o *oriented) At(x, y int) color.Color {
	if !image.Pt(x, y).In(o.bounds) {
		return color.NRGBA64{}
	}
	w, h := o.src.Bounds().Dx(), o.src.Bounds().Dy()
	switch o.orientation {
	case 2:
		x = w - 1 - x
	case 3:
		x, y = w-1-x, h-1-y
	case 4:
		y = h - 1 - y
	case 5:
		x, y = y, x
	case 6:
		x, y = y, h-1-x
	case 7:
		x, y = w-1-y, h-1-x
	case 8:
		x, y = w-1-y, x
	}
	return o.src.At(x+o.src.Bounds().Min.X, y+o.src.Bounds().Min.Y)
}
