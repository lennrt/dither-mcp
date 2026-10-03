package colorprofile

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func FuzzICCProfile(f *testing.F) {
	for _, name := range []string{"srgb", "display-p3", "adobe-rgb"} {
		f.Add(fixture(f, name))
	}
	f.Add([]byte{})
	f.Add([]byte("acsp"))
	f.Fuzz(func(t *testing.T, data []byte) {
		transform, err := Parse(data)
		if err != nil {
			return
		}
		img := image.NewNRGBA64(image.Rect(0, 0, 3, 1))
		for i, c := range []color.NRGBA64{{R: 12345, G: 23456, B: 34567, A: 0}, {R: 65535, G: 128, B: 32768, A: 1}, {R: 32768, G: 16384, B: 49152, A: 65535}} {
			img.SetNRGBA64(i, 0, c)
		}
		out, err := transform.Apply(context.Background(), img)
		if err != nil {
			t.Fatal(err)
		}
		for x := range 3 {
			if out.(*image.NRGBA64).NRGBA64At(x, 0).A != img.NRGBA64At(x, 0).A {
				t.Fatal("valid transform changed source alpha")
			}
		}
		for _, curve := range transform.curves {
			for _, value := range curve {
				if !finite(value) || value < 0 || value > 1 {
					t.Fatal("valid curve contains an invalid value")
				}
			}
		}
	})
}

func FuzzPNGTransform(f *testing.F) {
	f.Add(0.0, .3127, .329, .64, .33, .30, .60, .15, .06)
	f.Add(1.0, .3127, .329, .68, .32, .265, .69, .15, .06)
	f.Fuzz(func(t *testing.T, gamma, wx, wy, rx, ry, gx, gy, bx, by float64) {
		chroma := [8]float64{wx, wy, rx, ry, gx, gy, bx, by}
		transform, err := FromPNG(gamma, &chroma)
		if err != nil {
			return
		}
		actual := applyPixel(t, transform, [3]uint16{12345, 23456, 34567})
		if actual.A != 43210 {
			t.Fatal("PNG transform changed alpha")
		}
		for _, row := range transform.matrix {
			for _, value := range row {
				if !finite(value) {
					t.Fatal("PNG transform contains an invalid matrix")
				}
			}
		}
	})
}
