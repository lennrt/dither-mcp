package engine

import (
	"context"
	"image/color"
	"testing"
)

// FuzzProcess exercises valid bounded recipes across every implementation. It
// checks contract properties rather than duplicating algorithm code.
func FuzzProcess(f *testing.F) {
	f.Add([]byte{0, 16, 16, 3, 1, 3, 4, 8, 42, 11, 17, 7})
	f.Add([]byte{25, 1, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{15, 31, 1, 2, 3, 4, 2, 1, 255, 2, 7, 4})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 12 {
			return
		}
		w, h := int(data[1]%32)+1, int(data[2]%32)+1
		cfg := Config{Algorithm: catalog[int(data[0])%len(catalog)].ID, PixelScale: int(data[3]%8) + 1, Seed: int64(data[8]), Brightness: float64(data[4]%9)/16 - .25, Contrast: Float(float64(data[5]%8) / 4), Saturation: Float(float64(data[6]%8) / 4), Gamma: Float(.5 + float64(data[7]%8)/4), Serpentine: data[9]%2 == 1, Strength: Float(float64(data[10]%9) / 4)}
		if data[9]%3 == 0 {
			cfg.ColorSpace = "linear-rgb"
		}
		if data[11]%4 == 0 {
			cfg.Mask = &Mask{Shape: "circle", X: w / 2, Y: h / 2, Radius: max(1, min(w, h)/3)}
		}
		src := fixture(w, h)
		out, err := Process(context.Background(), src, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if out.Rect.Dx() != w || out.Rect.Dy() != h {
			t.Fatal("dimension contract violated")
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := out.NRGBAAt(x, y)
				if c.A == 0 && c != (color.NRGBA{}) {
					t.Fatal("transparent pixel is not canonical black")
				}
			}
		}
	})
}
func FuzzParseHex(f *testing.F) {
	f.Add("#abc")
	f.Add("000000")
	f.Add("#12xz89")
	f.Fuzz(func(t *testing.T, s string) {
		c, err := ParseHex(s)
		if err == nil {
			if c.A != 255 {
				t.Fatal("nonopaque parsed color")
			}
			again, err := ParseHex(Hex(c))
			if err != nil || again != c {
				t.Fatal("hex roundtrip failed")
			}
		}
	})
}
