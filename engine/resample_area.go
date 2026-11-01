package engine

import (
	"context"
	"image"
	"image/color"
)

type areaSample struct{ r, g, b, a uint64 }

// resampleArea integrates exact box coverage in source-pixel coordinates.
// Integer endpoints and numerators preserve coverage until final byte rounding.
// Validated source geometry bounds every numerator by MaxPixels*255*255, which
// fits uint64. The smaller separable intermediate controls axis order, and two
// scanline buffers bound storage.
func resampleArea(ctx context.Context, src image.Image, r image.Rectangle, w, h int) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	read := pixelReader(src)
	sw, sh := r.Dx(), r.Dy()
	transpose := w*sh > h*sw
	if transpose {
		sw, sh, w, h = sh, sw, h, w
	}
	area := uint64(sw) * uint64(sh)
	round := func(numerator, denominator uint64) uint8 {
		return uint8((numerator + denominator/2) / denominator)
	}
	get := func(x, y int) color.NRGBA {
		if transpose {
			return read(r.Min.X+y, r.Min.Y+x)
		}
		return read(r.Min.X+x, r.Min.Y+y)
	}
	horizontal := make([]areaSample, w)
	accumulated := make([]areaSample, w)
	cachedY := -1
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		clear(accumulated)
		top, bottom := y*sh, (y+1)*sh
		for sy := top / h; sy < (bottom+h-1)/h; sy++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if cachedY != sy {
				for x := 0; x < w; x++ {
					left, right := x*sw, (x+1)*sw
					var sum areaSample
					for sx := left / w; sx < (right+w-1)/w; sx++ {
						c := get(sx, sy)
						weight := uint64(min(right, (sx+1)*w)-max(left, sx*w)) * uint64(c.A)
						sum.a += weight
						sum.r += weight * uint64(c.R)
						sum.g += weight * uint64(c.G)
						sum.b += weight * uint64(c.B)
					}
					horizontal[x] = sum
				}
				cachedY = sy
			}
			weight := uint64(min(bottom, (sy+1)*h) - max(top, sy*h))
			for x, p := range horizontal {
				accumulated[x].a += p.a * weight
				accumulated[x].r += p.r * weight
				accumulated[x].g += p.g * weight
				accumulated[x].b += p.b * weight
			}
		}
		for x, p := range accumulated {
			c := color.NRGBA{}
			if p.a > 0 {
				c = color.NRGBA{round(p.r, p.a), round(p.g, p.a), round(p.b, p.a), round(p.a, area)}
				if c.A == 0 {
					c = color.NRGBA{}
				}
			}
			if transpose {
				out.SetNRGBA(y, x, c)
			} else {
				out.SetNRGBA(x, y, c)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
