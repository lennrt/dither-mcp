package engine

import (
	"context"
	"image"
	"image/color"
)

type areaSample struct{ r, g, b, a float64 }

// resampleArea integrates exact box coverage in source-pixel coordinates.
// Integer endpoints avoid rounding errors at shared footprint edges. The smaller
// separable intermediate controls axis order. Two scanline buffers bound storage.
func resampleArea(ctx context.Context, src image.Image, r image.Rectangle, w, h int) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	read := pixelReader(src)
	sw, sh := r.Dx(), r.Dy()
	transpose := w*sh > h*sw
	if transpose {
		sw, sh, w, h = sh, sw, h, w
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
						weight := float64(min(right, (sx+1)*w)-max(left, sx*w)) * float64(c.A) / float64(sw)
						sum.a += weight
						sum.r += weight * float64(c.R)
						sum.g += weight * float64(c.G)
						sum.b += weight * float64(c.B)
					}
					horizontal[x] = sum
				}
				cachedY = sy
			}
			weight := float64(min(bottom, (sy+1)*h)-max(top, sy*h)) / float64(sh)
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
				c = color.NRGBA{byteOf(p.r / p.a / 255), byteOf(p.g / p.a / 255), byteOf(p.b / p.a / 255), byteOf(p.a / 255)}
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
