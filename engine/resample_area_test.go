package engine

import (
	"context"
	"errors"
	"image"
	"image/color"
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
