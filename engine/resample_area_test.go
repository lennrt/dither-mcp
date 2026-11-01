package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestAreaReferenceCoverage(t *testing.T) {
	// Three unit-width pixels reduced to two boxes of width 1.5.
	// Coverage is [1, .5, 0] and [0, .5, 1], respectively.
	src := image.NewNRGBA(image.Rect(5, -2, 8, -1))
	for i, c := range []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 128}, {0, 0, 255, 0}} {
		src.SetNRGBA(5+i, -2, c)
	}
	got, err := resampleArea(context.Background(), src, src.Bounds(), 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []color.NRGBA{{204, 51, 0, 213}, {0, 255, 0, 43}} {
		if c := got.NRGBAAt(x, 0); c != want {
			t.Fatalf("pixel %d=%v, want %v", x, c, want)
		}
	}
}

func TestAreaStripeAndCrop(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 20, 22))
	// Eight-pixel crop: six white, two central black. Bilinear sees only black.
	for y := 20; y < 22; y++ {
		for x := 10; x < 20; x++ {
			c := color.NRGBA{255, 255, 255, 255}
			if x == 14 || x == 15 {
				c = color.NRGBA{0, 0, 0, 255}
			}
			src.SetNRGBA(x, y, c)
		}
	}
	r := image.Rect(11, 20, 19, 22)
	got, err := resampleArea(context.Background(), src, r, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if c := got.NRGBAAt(0, 0); c != (color.NRGBA{191, 191, 191, 255}) {
		t.Fatalf("coverage average=%v", c)
	}
	// Process integration: an inverted empty selection preserves adjusted samples.
	out, err := Process(context.Background(), src, Config{ResizeFilter: "area", Width: 1, Height: 1, Crop: &Rect{X: 1, Width: 8, Height: 2}, Mask: &Mask{Shape: "rectangle", Width: 1, Height: 1, Invert: true}})
	if err != nil || out.NRGBAAt(0, 0) != got.NRGBAAt(0, 0) {
		t.Fatalf("pipeline area/crop result=%v err=%v", out, err)
	}
}

func TestAreaMixedAxesAndTransparentBlack(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	src.SetNRGBA(0, 1, color.NRGBA{0, 0, 255, 255})
	src.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 0})
	src.SetNRGBA(1, 1, color.NRGBA{255, 255, 0, 0})
	got, err := resampleArea(context.Background(), src, src.Bounds(), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []color.NRGBA{{128, 0, 128, 255}, {128, 0, 128, 255}, {}, {}} {
		if c := got.NRGBAAt(x, 0); c != want {
			t.Fatalf("mixed pixel %d=%v want %v", x, c, want)
		}
	}
}

func TestAreaCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	src := &encodeCancelImage{Image: image.NewNRGBA(image.Rect(0, 0, 8, 8)), cancel: cancel, atCall: 1}
	got, err := resampleArea(ctx, src, src.Bounds(), 1, 1)
	if got != nil || !errors.Is(err, context.Canceled) || src.calls > 8 {
		t.Fatalf("got=%v err=%v reads=%d", got, err, src.calls)
	}
}

func TestAreaFractionalEnlargement(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	src.SetNRGBA(1, 0, color.NRGBA{0, 0, 255, 255})
	got, err := resampleArea(context.Background(), src, src.Bounds(), 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []color.NRGBA{{255, 0, 0, 255}, {128, 0, 128, 255}, {0, 0, 255, 255}} {
		if c := got.NRGBAAt(x, 0); c != want {
			t.Fatalf("fractional enlargement pixel %d=%v want %v", x, c, want)
		}
	}
}

func TestAreaHalfAlphaRemainsVisibleInGIF(t *testing.T) {
	for _, transpose := range []bool{false, true} {
		w, h := 7, 2
		ow, oh, px, py := 5, 1, 2, 0
		if transpose {
			w, h, ow, oh, px, py = h, w, oh, ow, py, px
		}
		src := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < 2; y++ {
			for x, alpha := range []uint8{0, 0, 160, 114, 162, 0, 0} {
				if x == 4 && y == 1 {
					alpha = 163
				}
				sx, sy := x, y
				if transpose {
					sx, sy = y, x
				}
				src.SetNRGBA(sx, sy, color.NRGBA{255, 255, 255, alpha})
			}
		}
		// The central box covers columns 2, 3, 4 in proportions 1:5:1.
		// Across both rows its alpha is (160 + 5*114 + 162.5)/7 = 127.5.
		out, err := Process(context.Background(), src, Config{Algorithm: "threshold", ResizeFilter: "area", Width: ow, Height: oh})
		if err != nil {
			t.Fatal(err)
		}
		if got := out.NRGBAAt(px, py); got != (color.NRGBA{255, 255, 255, 128}) {
			t.Fatalf("transpose=%v: exact half-alpha sample=%v", transpose, got)
		}
		var encoded bytes.Buffer
		if err := EncodeContext(context.Background(), "gif", &encoded, out); err != nil {
			t.Fatal(err)
		}
		decoded, err := gif.Decode(&encoded)
		if err != nil {
			t.Fatal(err)
		}
		if got := color.NRGBAModel.Convert(decoded.At(px, py)).(color.NRGBA); got != (color.NRGBA{255, 255, 255, 255}) {
			t.Fatalf("transpose=%v: GIF discarded alpha128 pixel: %v", transpose, got)
		}
	}
}

// areaLatticeReference subdivides each source pixel into w*h unit cells. Every
// output footprint then contains exactly r.Dx()*r.Dy() cells. Enumerating those
// cells is an independent, deliberately slow reference: it uses neither overlap
// calculations nor separable scanlines.
func areaLatticeReference(src *image.NRGBA, r image.Rectangle, w, h, x, y int) color.NRGBA {
	var alpha, red, green, blue uint64
	for dy := 0; dy < r.Dy(); dy++ {
		for dx := 0; dx < r.Dx(); dx++ {
			sx := r.Min.X + (x*r.Dx()+dx)/w
			sy := r.Min.Y + (y*r.Dy()+dy)/h
			c := src.NRGBAAt(sx, sy)
			a := uint64(c.A)
			alpha += a
			red += uint64(c.R) * a
			green += uint64(c.G) * a
			blue += uint64(c.B) * a
		}
	}
	// Quotient/remainder rounding is exact, including odd denominators.
	round := func(numerator, denominator uint64) uint8 {
		q, remainder := numerator/denominator, numerator%denominator
		if 2*remainder >= denominator {
			q++
		}
		return uint8(q)
	}
	if alpha == 0 {
		return color.NRGBA{}
	}
	c := color.NRGBA{round(red, alpha), round(green, alpha), round(blue, alpha), round(alpha, uint64(r.Dx()*r.Dy()))}
	if c.A == 0 {
		return color.NRGBA{}
	}
	return c
}

func areaReferenceFixture(bounds image.Rectangle) *image.NRGBA {
	src := image.NewNRGBA(bounds)
	alphas := []uint8{0, 1, 63, 127, 128, 129, 254, 255}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx, dy := x-bounds.Min.X, y-bounds.Min.Y
			src.SetNRGBA(x, y, color.NRGBA{uint8(dx*67 + dy*23), uint8(dx*11 + dy*41), uint8(dx*31 + dy*17), alphas[(dx*3+dy*5)%len(alphas)]})
		}
	}
	return src
}

func TestAreaMatchesExactLatticeReference(t *testing.T) {
	for _, sw := range []int{1, 2, 3, 5, 7} {
		for _, sh := range []int{1, 2, 3, 5, 7} {
			src := areaReferenceFixture(image.Rect(7, -11, 7+sw+2, -11+sh+2))
			r := image.Rect(8, -10, 8+sw, -10+sh)
			for _, w := range []int{1, 2, 3, 4, 8} {
				for _, h := range []int{1, 2, 3, 4, 8} {
					out, err := resampleArea(context.Background(), src, r, w, h)
					if err != nil {
						t.Fatal(err)
					}
					for y := 0; y < h; y++ {
						for x := 0; x < w; x++ {
							want := areaLatticeReference(src, r, w, h, x, y)
							if got := out.NRGBAAt(x, y); got != want {
								t.Fatalf("%dx%d -> %dx%d pixel(%d,%d)=%v want %v", sw, sh, w, h, x, y, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestAreaCropMaskAndPixelScaleReference(t *testing.T) {
	src := areaReferenceFixture(image.Rect(10, -3, 21, 7))
	crop := Rect{X: 1, Y: 1, Width: 9, Height: 7}
	r := image.Rect(11, -2, 20, 5)
	for _, dimensions := range [][3]int{{7, 5, 1}, {11, 8, 3}, {3, 13, 2}, {13, 3, 2}} {
		w, h, scale := dimensions[0], dimensions[1], dimensions[2]
		gw, gh := (w+scale-1)/scale, (h+scale-1)/scale
		cfg := Config{ResizeFilter: "area", Width: w, Height: h, PixelScale: scale, Crop: &crop,
			Mask: &Mask{Shape: "rectangle", Width: gw, Height: gh, Invert: true}}
		out, err := Process(context.Background(), src, cfg)
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				gx, gy := (2*x+1)*gw/(2*w), (2*y+1)*gh/(2*h)
				want := areaLatticeReference(src, r, gw, gh, gx, gy)
				if got := out.NRGBAAt(x, y); got != want {
					t.Fatalf("%dx%d scale%d pixel(%d,%d)=%v want %v", w, h, scale, x, y, got, want)
				}
			}
		}
	}
}

func TestAreaMaximumCoverageNumerator(t *testing.T) {
	// A single output sample reaches MaxPixels*255*255 for each RGB numerator.
	// This exceeds uint32 even though both input geometry limits are satisfied.
	src := image.NewGray(image.Rect(0, 0, MaxDimension, MaxPixels/MaxDimension))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	out, err := resampleArea(context.Background(), src, src.Bounds(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.NRGBAAt(0, 0); got != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("maximum numerator overflowed: %v", got)
	}
}
