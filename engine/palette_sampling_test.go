package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"reflect"
	"testing"
)

func TestExtractPaletteSparseVisiblePixels(t *testing.T) {
	// These images have sampling strides of two, three, and four pixels. Both
	// visible colors fall between samples, including one at the end. The middle
	// size also makes stride blocks cross row boundaries.
	for _, size := range []image.Point{{1024, 512}, {1025, 513}, {1024, 1024}} {
		for _, origin := range []image.Point{{}, {13, -7}} {
			img := image.NewNRGBA(image.Rectangle{Min: origin, Max: origin.Add(size)})
			red, white := color.NRGBA{255, 0, 0, 255}, color.NRGBA{255, 255, 255, 255}
			img.SetNRGBA(origin.X+1, origin.Y, red)
			img.SetNRGBA(img.Rect.Max.X-1, img.Rect.Max.Y-1, white)
			want := []color.NRGBA{red, white}
			for attempt := 0; attempt < 2; attempt++ {
				got, err := ExtractPalette(context.Background(), img, 4)
				if err != nil {
					t.Fatalf("bounds %v, attempt %d: %v", img.Rect, attempt, err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("bounds %v: got %v, want %v", img.Rect, got, want)
				}
			}
		}
	}
}

func TestExtractPaletteSparseAlphaWeighting(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	img.SetNRGBA(1, 0, color.NRGBA{0, 0, 0, 255})
	img.SetNRGBA(3, 0, color.NRGBA{120, 0, 0, 85})
	img.SetNRGBA(5, 0, color.NRGBA{255, 255, 255, 255})
	got, err := ExtractPalette(context.Background(), img, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Median cut merges black and red; the red channel is (120*85)/(255+85).
	want := []color.NRGBA{{30, 0, 0, 255}, {255, 255, 255, 255}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want alpha-weighted %v", got, want)
	}
}

func TestSparseVisibleGIFPaletteOverflow(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	for i := 0; i < 256; i++ {
		img.SetNRGBA(i*2+1, 0, color.NRGBA{uint8(i), uint8(i ^ 77), uint8(i ^ 137), 255})
	}
	// Transparency plus 256 visible colors forces the GIF median-cut path.
	var encoded bytes.Buffer
	if err := Encode("gif", &encoded, img); err != nil {
		t.Fatal(err)
	}
	decoded, err := gif.Decode(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	palette := decoded.(*image.Paletted).Palette
	if len(palette) > 256 || len(palette) < 2 {
		t.Fatalf("invalid GIF palette size %d", len(palette))
	}
	for x := 0; x < 512; x++ {
		_, _, _, alpha := decoded.At(x, 0).RGBA()
		if (alpha != 0) != (x%2 == 1) {
			t.Fatalf("pixel %d: sparse GIF alpha changed to %d", x, alpha)
		}
	}
}

type samplingCancelImage struct {
	image.Image
	cancel context.CancelFunc
	calls  int
	atCall int
}

func (img *samplingCancelImage) At(x, y int) color.Color {
	img.calls++
	if img.calls == img.atCall {
		img.cancel()
	}
	return img.Image.At(x, y)
}

func TestExtractPaletteSparseFallbackCancellation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	img.SetNRGBA(1, 0, color.NRGBA{255, 0, 0, 255})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Let the entire regular sample finish, then cancel during the fallback.
	source := &samplingCancelImage{Image: img, cancel: cancel, atCall: 262144 + 32}
	got, err := ExtractPalette(ctx, source, 4)
	if !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("got %v, error %v; want cancellation without a palette", got, err)
	}
	if source.calls > 262144+1024 {
		t.Fatalf("fallback did not stop by its next row: %d pixel reads", source.calls)
	}
}

func TestExtractPaletteTransparentFallbackReadsEachPixelOnce(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1025, 513))
	source := &samplingCancelImage{Image: img, atCall: -1}
	if got, err := ExtractPalette(context.Background(), source, 4); err == nil || got != nil {
		t.Fatalf("got %v, error %v; transparent extraction should fail", got, err)
	}
	if want := img.Rect.Dx() * img.Rect.Dy(); source.calls != want {
		t.Fatalf("read %d pixels, want every pixel exactly once (%d)", source.calls, want)
	}
}

func TestExtractPaletteKeepsNonemptyStrideSample(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	red, white := color.NRGBA{255, 0, 0, 255}, color.NRGBA{255, 255, 255, 255}
	img.SetNRGBA(0, 0, red)
	img.SetNRGBA(2, 0, white)
	img.SetNRGBA(1, 0, color.NRGBA{0, 0, 255, 255})
	source := &samplingCancelImage{Image: img, atCall: -1}
	got, err := ExtractPalette(context.Background(), source, 4)
	if err != nil {
		t.Fatal(err)
	}
	if want := []color.NRGBA{red, white}; !reflect.DeepEqual(got, want) {
		t.Fatalf("changed existing stride palette: got %v, want %v", got, want)
	}
	if source.calls != 262144 {
		t.Fatalf("nonempty sample unexpectedly rescanned image: %d reads", source.calls)
	}
}
