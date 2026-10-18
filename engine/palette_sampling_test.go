package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

func TestExtractPaletteFullScanCancellation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	img.SetNRGBA(1, 0, color.NRGBA{255, 0, 0, 255})
	for _, atCall := range []int{32, 262144 + 32, 1024 * 512} {
		ctx, cancel := context.WithCancel(context.Background())
		source := &samplingCancelImage{Image: img, cancel: cancel, atCall: atCall}
		got, err := ExtractPalette(ctx, source, 4)
		cancel()
		if !errors.Is(err, context.Canceled) || got != nil {
			t.Fatalf("at read %d: got %v, error %v; want cancellation without a palette", atCall, got, err)
		}
		if source.calls < atCall || source.calls > (atCall+1023)/1024*1024 {
			t.Fatalf("did not stop by the next row after read %d: %d pixel reads", atCall, source.calls)
		}
	}
}

func TestExtractPaletteTransparentReadsEachPixelOnce(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1025, 513))
	source := &samplingCancelImage{Image: img, atCall: -1}
	if got, err := ExtractPalette(context.Background(), source, 4); err == nil || got != nil {
		t.Fatalf("got %v, error %v; transparent extraction should fail", got, err)
	}
	if want := img.Rect.Dx() * img.Rect.Dy(); source.calls != want {
		t.Fatalf("read %d pixels, want every pixel exactly once (%d)", source.calls, want)
	}
}

func TestExtractPaletteIncludesAllVisibleColors(t *testing.T) {
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
	if want := []color.NRGBA{{0, 0, 255, 255}, red, white}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want every distinct visible color %v", got, want)
	}
	if source.calls != img.Rect.Dx()*img.Rect.Dy() {
		t.Fatalf("did not read every pixel exactly once: %d reads", source.calls)
	}
}

func TestExtractPaletteSmallImageCompatibility(t *testing.T) {
	// Captured before replacing fixed-stride sampling. Both images were read
	// in full previously, including the old 262,144-pixel sampling boundary.
	for _, tc := range []struct {
		size image.Point
		want []color.NRGBA
	}{
		{image.Pt(13, 7), []color.NRGBA{{37, 73, 41, 255}, {92, 72, 80, 255}, {43, 135, 59, 255}, {145, 81, 119, 255}, {100, 142, 100, 255}, {191, 89, 153, 255}, {146, 149, 134, 255}, {199, 147, 171, 255}}},
		{image.Pt(512, 512), []color.NRGBA{{63, 63, 64, 255}, {63, 64, 192, 255}, {191, 64, 63, 255}, {192, 64, 191, 255}, {64, 191, 64, 255}, {64, 192, 191, 255}, {192, 192, 64, 255}, {192, 191, 192, 255}}},
	} {
		img := image.NewNRGBA(image.Rect(17, -11, 17+tc.size.X, -11+tc.size.Y))
		for y := 0; y < tc.size.Y; y++ {
			for x := 0; x < tc.size.X; x++ {
				img.SetNRGBA(17+x, -11+y, color.NRGBA{uint8(x*17 + y*3), uint8(x*5 + y*23), uint8(x*13 + y*7), uint8(x*11 + y*31)})
			}
		}
		got, err := ExtractPalette(context.Background(), img, 8)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("size %v: got %v, error %v; want %v", tc.size, got, err, tc.want)
		}
	}
}

func TestExtractPaletteIncludesRareColorsOutsideNonemptyStride(t *testing.T) {
	for _, size := range []image.Point{{1024, 512}, {1025, 513}, {1024, 1024}} {
		for _, origin := range []image.Point{{}, {13, -7}} {
			img := image.NewNRGBA(image.Rectangle{Min: origin, Max: origin.Add(size)})
			black, white := color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 255, 255, 255}
			for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
				for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
					img.SetNRGBA(x, y, black)
				}
			}
			// A single accent falls between regular stride samples even though
			// those samples contain opaque background pixels.
			img.SetNRGBA(origin.X+1, origin.Y, white)
			got, err := ExtractPalette(context.Background(), img, 2)
			if err != nil {
				t.Fatal(err)
			}
			if want := []color.NRGBA{black, white}; !reflect.DeepEqual(got, want) {
				t.Errorf("bounds %v: got %v, want singleton accent %v", img.Rect, got, want)
			}
		}
	}
}

func TestExtractPaletteIncludesMinorityStripes(t *testing.T) {
	img := image.NewNRGBA(image.Rect(7, -11, 1031, 1013))
	black, red := color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 0, 0, 255}
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			c := black
			if (x-img.Rect.Min.X)%8 == 1 {
				c = red
			}
			img.SetNRGBA(x, y, c)
		}
	}
	got, err := ExtractPalette(context.Background(), img, 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := []color.NRGBA{black, red}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want minority stripe %v", got, want)
	}
}

func BenchmarkExtractPalette(b *testing.B) {
	for _, side := range []int{1024, 4096} {
		for _, colorful := range []bool{false, true} {
			b.Run(fmt.Sprintf("%d/colorful=%t", side, colorful), func(b *testing.B) {
				img := image.NewNRGBA(image.Rect(0, 0, side, side))
				for i := 0; i < side*side; i++ {
					key := i
					if !colorful {
						key = (i % 16) * 0x101010
					}
					img.SetNRGBA(i%side, i/side, color.NRGBA{uint8(key >> 16), uint8(key >> 8), uint8(key), 255})
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := ExtractPalette(context.Background(), img, 16); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
