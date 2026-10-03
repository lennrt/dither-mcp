package colorprofile

import (
	"context"
	"image"
	"image/color"
	"math"
	"reflect"
)

// Apply converts unassociated source RGB to 16-bit sRGB and preserves alpha.
// It preserves image bounds and checks cancellation before allocation and per row.
// The caller handles images without color metadata before calling this method.
func (t *Transform) Apply(ctx context.Context, src image.Image) (image.Image, error) {
	if t == nil || !t.valid {
		return nil, fail("use Parse or FromPNG to construct a transform")
	}
	if ctx == nil {
		return nil, fail("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if src == nil || (reflect.ValueOf(src).Kind() == reflect.Pointer && reflect.ValueOf(src).IsNil()) {
		return nil, fail("source image is required")
	}
	bounds := src.Bounds()
	if bounds.Max.X <= bounds.Min.X || bounds.Max.Y <= bounds.Min.Y {
		return nil, fail("source image must have positive dimensions")
	}
	// Unsigned subtraction also handles extreme int coordinates without overflow.
	w, h := uint64(bounds.Max.X)-uint64(bounds.Min.X), uint64(bounds.Max.Y)-uint64(bounds.Min.Y)
	if w > MaxImagePixels || h > MaxImagePixels || w*h > MaxImagePixels {
		return nil, fail("source image exceeds %d pixels", MaxImagePixels)
	}
	out := image.NewNRGBA64(bounds)
	sample := sourcePixels(src)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if (x-bounds.Min.X)&1023 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			c := sample(x, y)
			linear := t.matrix.vector([3]float64{t.curves[0][c.R], t.curves[1][c.G], t.curves[2][c.B]})
			out.SetNRGBA64(x, y, color.NRGBA64{R: channel(linear[0]), G: channel(linear[1]), B: channel(linear[2]), A: c.A})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func sourcePixels(src image.Image) func(int, int) color.NRGBA64 {
	switch img := src.(type) {
	case *image.NRGBA64:
		return img.NRGBA64At
	case *image.NRGBA:
		return func(x, y int) color.NRGBA64 {
			v := img.NRGBAAt(x, y)
			// Preserve hidden RGB and small-alpha channel values exactly.
			return color.NRGBA64{R: uint16(v.R) * 257, G: uint16(v.G) * 257, B: uint16(v.B) * 257, A: uint16(v.A) * 257}
		}
	case image.RGBA64Image:
		// Standard image types expose channels without boxing each pixel.
		return func(x, y int) color.NRGBA64 { return unassociate(img.RGBA64At(x, y)) }
	default:
		return func(x, y int) color.NRGBA64 {
			switch v := src.At(x, y).(type) {
			case color.NRGBA64:
				return v
			case color.NRGBA:
				return color.NRGBA64{R: uint16(v.R) * 257, G: uint16(v.G) * 257, B: uint16(v.B) * 257, A: uint16(v.A) * 257}
			default:
				return color.NRGBA64Model.Convert(v).(color.NRGBA64)
			}
		}
	}
}

func unassociate(v color.RGBA64) color.NRGBA64 {
	if v.A == 0 {
		return color.NRGBA64{}
	}
	if v.A == 65535 {
		return color.NRGBA64{R: v.R, G: v.G, B: v.B, A: v.A}
	}
	return color.NRGBA64{
		R: uint16(uint32(v.R) * 65535 / uint32(v.A)),
		G: uint16(uint32(v.G) * 65535 / uint32(v.A)),
		B: uint16(uint32(v.B) * 65535 / uint32(v.A)), A: v.A,
	}
}

func channel(linear float64) uint16 {
	return uint16(math.Round(clamp(srgbEncode(linear)) * 65535))
}
