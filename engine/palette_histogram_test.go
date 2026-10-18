package engine

import (
	"context"
	"errors"
	"image"
	"image/color"
	"reflect"
	"testing"
)

func histogramFixture(visible int, alpha bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(19, -7, 19+1024, -7+(visible+1023)/1024))
	for i := 0; i < visible; i++ {
		c := color.NRGBA{uint8(i >> 16), uint8(i >> 8), uint8(i), 255}
		if alpha {
			c.A = uint8(i%255 + 1)
		}
		img.SetNRGBA(19+i%1024, -7+i/1024, c)
	}
	return img
}

func TestPaletteHistogramExactLimitAndOverflow(t *testing.T) {
	const limit = 262144
	for _, visible := range []int{limit, limit + 1} {
		img := histogramFixture(visible, true)
		got, err := paletteHistogram(context.Background(), img)
		if err != nil {
			t.Fatal(err)
		}
		wantSize := limit
		if visible > limit {
			// RGBs 0..262143 occupy a 4×256×256 volume: 4096 six-bit
			// bins. The next RGB (4,0,0) adds the 4097th bin.
			wantSize = 4097
		}
		if len(got) != wantSize || len(got) > limit {
			t.Fatalf("%d colors: histogram size %d, want %d", visible, len(got), wantSize)
		}
		type totals struct {
			weight int64
			sum    [3]int64
		}
		want := map[[3]uint8]totals{}
		for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				channels := [3]uint8{c.R, c.G, c.B}
				key := channels
				if visible > limit {
					for i := range key {
						key[i] /= 4
					}
				}
				value := want[key]
				value.weight += int64(c.A)
				for i := range channels {
					value.sum[i] += int64(channels[i]) * int64(c.A)
				}
				want[key] = value
			}
		}
		for _, entry := range got {
			key := [3]uint8{entry.c.R, entry.c.G, entry.c.B}
			if visible > limit {
				for i := range key {
					key[i] /= 4
				}
			}
			expected, exists := want[key]
			if !exists || entry.weight != expected.weight || entry.sums != expected.sum {
				t.Fatalf("%d colors, bin %v: got %+v, want %+v", visible, key, entry, expected)
			}
			delete(want, key)
		}
		if len(want) != 0 {
			t.Fatalf("%d color bins never contributed", len(want))
		}
	}
}

func TestExtractPaletteFoldedCentroidsKeepOriginalSums(t *testing.T) {
	img := histogramFixture(262145, false)
	// With count=2 the green-axis split separates G=0..127 plus singleton
	// (4,0,0), from G=128..255. The singleton takes the lower green and blue
	// means just below their half-integer rounding boundaries. Averaging
	// rounded bin centroids would incorrectly yield (2,64,128) for that box.
	want := []color.NRGBA{{2, 63, 127, 255}, {2, 192, 128, 255}}
	for _, reverse := range []bool{false, true} {
		if reverse {
			for a, b := 0, len(img.Pix)-4; a < b; a, b = a+4, b-4 {
				for channel := 0; channel < 4; channel++ {
					img.Pix[a+channel], img.Pix[b+channel] = img.Pix[b+channel], img.Pix[a+channel]
				}
			}
		}
		got, err := ExtractPalette(context.Background(), img, 2)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("reverse=%t: got %v, error %v; want original weighted centroids %v", reverse, got, err, want)
		}
	}
}

func TestPaletteHistogramMaximumImageWeight(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4096, 4096))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	got, err := paletteHistogram(context.Background(), img)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d colors, error %v", len(got), err)
	}
	// The largest legal opaque image exceeds the signed 32-bit weight range.
	const weight int64 = 4096 * 4096 * 255
	if got[0].weight != weight || got[0].sums != [3]int64{weight * 255, weight * 255, weight * 255} {
		t.Fatalf("maximum-image totals overflowed: %+v", got[0])
	}
}

type histogramCancelContext struct {
	context.Context
	cancel context.CancelFunc
	calls  int
	atCall int
}

func (ctx *histogramCancelContext) Err() error {
	ctx.calls++
	if ctx.calls == ctx.atCall {
		ctx.cancel()
	}
	return ctx.Context.Err()
}

func TestPaletteHistogramCancellationDuringFold(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	// The first 257 checks start source rows; the next check starts the fold.
	// Cancel on a subsequent fold checkpoint, before any more pixels are read.
	ctx := &histogramCancelContext{Context: parent, cancel: cancel, atCall: 259}
	source := &samplingCancelImage{Image: histogramFixture(262145, false), atCall: -1}
	got, err := paletteHistogram(ctx, source)
	if !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("got %d colors, error %v; want cancellation during histogram folding", len(got), err)
	}
	if source.calls != 262145 || ctx.calls != ctx.atCall {
		t.Fatalf("cancellation read %d pixels and checked context %d times", source.calls, ctx.calls)
	}
}

func TestExtractPaletteNonemptySampleAlphaWeighting(t *testing.T) {
	img := image.NewNRGBA(image.Rect(-3, 9, 1021, 521))
	img.SetNRGBA(-3, 9, color.NRGBA{0, 0, 0, 255})
	img.SetNRGBA(-2, 9, color.NRGBA{120, 0, 0, 85})
	img.SetNRGBA(-1, 9, color.NRGBA{255, 255, 255, 255})
	got, err := ExtractPalette(context.Background(), img, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []color.NRGBA{{30, 0, 0, 255}, {255, 255, 255, 255}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want alpha-weighted %v", got, want)
	}
}
