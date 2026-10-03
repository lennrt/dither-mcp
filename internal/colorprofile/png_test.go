package colorprofile

import (
	"context"
	"image"
	"image/color"
	"math"
	"testing"
)

func TestPNGGammaAndChromaticities(t *testing.T) {
	linear, err := FromPNG(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeRGB(t, applyPixel(t, linear, [3]uint16{32768, 32768, 32768}), [3]uint16{48192, 48192, 48192}, 1)
	p3 := [8]float64{.3127, .3290, .68, .32, .265, .69, .15, .06}
	transform, err := FromPNG(0, &p3)
	if err != nil {
		t.Fatal(err)
	}
	closeRGB(t, applyPixel(t, transform, [3]uint16{28385, 32839, 24871}), [3]uint16{27254, 33008, 24027}, 3)
	closeRGB(t, applyPixel(t, transform, [3]uint16{65535, 0, 0}), [3]uint16{65535, 0, 0}, 0)
	for _, gamma := range []float64{0, 1, .45455} {
		for _, chroma := range []*[8]float64{nil, &srgbChromaticities} {
			transform, err := FromPNG(gamma, chroma)
			if err != nil {
				t.Fatal(err)
			}
			img := image.NewGray16(image.Rect(0, 0, 65536, 1))
			for x := range 65536 {
				img.SetGray16(x, 0, color.Gray16{Y: uint16(x)})
			}
			out, err := transform.Apply(context.Background(), img)
			if err != nil {
				t.Fatal(err)
			}
			for x := range 65536 {
				c := out.(*image.NRGBA64).NRGBA64At(x, 0)
				if c.R != c.G || c.G != c.B || c.A != 65535 {
					t.Fatalf("PNG neutral is not neutral at %d: %v", x, c)
				}
				if gamma == 0 && c.R != uint16(x) {
					t.Fatalf("sRGB gray precision changed at %d: %v", x, c)
				}
			}
		}
	}
}

func TestInvalidPNGMetadata(t *testing.T) {
	for _, gamma := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1), math.SmallestNonzeroFloat64} {
		if _, err := FromPNG(gamma, nil); err == nil {
			t.Fatalf("invalid gamma accepted: %g", gamma)
		}
	}
	for i := range 8 {
		for _, value := range []float64{-1, math.NaN(), math.Inf(1)} {
			chroma := srgbChromaticities
			chroma[i] = value
			if _, err := FromPNG(0, &chroma); err == nil {
				t.Fatalf("invalid chromaticity accepted at %d: %g", i, value)
			}
		}
	}
	for _, chroma := range [][8]float64{
		{.3127, 0, .64, .33, .30, .60, .15, .06},
		{.3127, .329, .64, .33, .64, .33, .15, .06},
		{.9, .1, .64, .33, .30, .60, .15, .06},
		{.3127, .329, .8, .6, .30, .60, .15, .06},
		{.3127, .329, .64, 1e-300, .30, .60, .15, .06},
	} {
		if _, err := FromPNG(0, &chroma); err == nil {
			t.Fatalf("invalid primary matrix accepted: %v", chroma)
		}
	}
}
