package engine

import (
	"context"
	"image"
	"image/color"
	"math"
	"sort"
)

func luminance(c color.NRGBA) float64 {
	return (.2126*float64(c.R) + .7152*float64(c.G) + .0722*float64(c.B)) / 255
}
func applyEffects(ctx context.Context, img *image.NRGBA, e Effects, seed int64) error {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	spacing := e.ScanlineSpacing
	if spacing == 0 {
		spacing = 2
	}
	if e.Scanlines > 0 || e.CRT > 0 || e.Noise > 0 {
		for y := 0; y < h; y++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			for x := 0; x < w; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				v := vec3{float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255}
				if e.Scanlines > 0 && y%spacing == spacing-1 {
					v = v.mul(1 - e.Scanlines)
				}
				if e.CRT > 0 {
					nx, ny := 2*(float64(x)+.5)/float64(w)-1, 2*(float64(y)+.5)/float64(h)-1
					vignette := 1 - e.CRT*.35*min(1, (nx*nx+ny*ny)/2)
					for k := range v {
						channel := 1.0
						if k != x%3 {
							channel = 1 - .45*e.CRT
						}
						v[k] *= vignette * channel
					}
				}
				if e.Noise > 0 {
					noise := (coordinateNoise(x, y, seed^0x23a1) - .5) * e.Noise
					for k := range v {
						v[k] += noise
					}
				}
				img.SetNRGBA(x, y, color.NRGBA{byteOf(v[0]), byteOf(v[1]), byteOf(v[2]), c.A})
			}
		}
	}
	if e.Glitch > 0 {
		// Copy one row at a time: memory does not grow with total image size.
		row := make([]byte, w*4)
		for y := 0; y < h; y++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			if coordinateNoise(0, y/4, seed^0x6c8f) > .35 {
				continue
			}
			shift := int(math.Round((coordinateNoise(1, y/4, seed^0x6c8f)*2 - 1) * float64(e.Glitch)))
			offset := y * img.Stride
			copy(row, img.Pix[offset:offset+w*4])
			for x := 0; x < w; x++ {
				source := (x - shift) % w
				if source < 0 {
					source += w
				}
				copy(img.Pix[offset+x*4:offset+x*4+4], row[source*4:source*4+4])
			}
		}
	}
	if e.PixelSort {
		threshold := scalar(e.PixelSortThreshold, .25)
		run := make([]color.NRGBA, 0, w)
		for y := 0; y < h; y++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			for x := 0; x < w; {
				c := img.NRGBAAt(x, y)
				if c.A == 0 || luminance(c) < threshold {
					x++
					continue
				}
				start := x
				run = run[:0]
				for x < w {
					c = img.NRGBAAt(x, y)
					if c.A == 0 || luminance(c) < threshold {
						break
					}
					run = append(run, c)
					x++
				}
				sort.SliceStable(run, func(i, j int) bool { return luminance(run[i]) < luminance(run[j]) })
				for i, c := range run {
					img.SetNRGBA(start+i, y, c)
				}
			}
		}
	}
	return nil
}
