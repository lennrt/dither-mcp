package engine

import (
	"image"
	"image/color"
)

// pixelReader avoids boxing a color interface for every sample of common Go
// image types. This matters when bilinear interpolation reads four pixels each.
func pixelReader(img image.Image) func(int, int) color.NRGBA {
	switch src := img.(type) {
	case *gifPaletteImage:
		return src.NRGBAAt
	case *image.NRGBA:
		return src.NRGBAAt
	case *image.RGBA:
		return func(x, y int) color.NRGBA {
			c := src.RGBAAt(x, y)
			if c.A == 255 {
				return color.NRGBA{c.R, c.G, c.B, 255}
			}
			if c.A == 0 {
				return color.NRGBA{}
			}
			a := uint32(c.A) * 257
			return color.NRGBA{uint8((uint32(c.R) * 257 * 65535 / a) >> 8), uint8((uint32(c.G) * 257 * 65535 / a) >> 8), uint8((uint32(c.B) * 257 * 65535 / a) >> 8), c.A}
		}
	case *image.NRGBA64:
		return func(x, y int) color.NRGBA {
			c := src.NRGBA64At(x, y)
			return color.NRGBA{uint8(c.R >> 8), uint8(c.G >> 8), uint8(c.B >> 8), uint8(c.A >> 8)}
		}
	case *image.Gray:
		return func(x, y int) color.NRGBA { v := src.GrayAt(x, y).Y; return color.NRGBA{v, v, v, 255} }
	case *image.Gray16:
		return func(x, y int) color.NRGBA { v := uint8(src.Gray16At(x, y).Y >> 8); return color.NRGBA{v, v, v, 255} }
	case *image.YCbCr:
		return func(x, y int) color.NRGBA {
			c := src.YCbCrAt(x, y)
			r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
			return color.NRGBA{r, g, b, 255}
		}
	case *image.Paletted:
		palette := make([]color.NRGBA, len(src.Palette))
		for i, c := range src.Palette {
			palette[i] = color.NRGBAModel.Convert(c).(color.NRGBA)
		}
		return func(x, y int) color.NRGBA { return palette[src.ColorIndexAt(x, y)] }
	case *image.Uniform:
		c := color.NRGBAModel.Convert(src.C).(color.NRGBA)
		return func(int, int) color.NRGBA { return c }
	default:
		return func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) }
	}
}
