package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"reflect"
	"strings"
	"testing"
)

func fixture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if (x+y)%13 == 0 {
				a = 0
			} else if (x+y)%7 == 0 {
				a = 96
			}
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 255 / max(1, w-1)), uint8(y * 255 / max(1, h-1)), uint8((x*17 + y*31) % 256), a})
		}
	}
	return img
}
func mustPalette(t *testing.T, name string) []color.NRGBA {
	t.Helper()
	p, err := ResolvePalette(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBayerReference(t *testing.T) {
	expected := [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}
	for y, row := range expected {
		for x, want := range row {
			if got := bayer(4, x, y); got != want {
				t.Fatalf("Bayer[%d,%d]=%d, want %d", x, y, got, want)
			}
		}
	}
	for _, n := range []int{2, 4, 8, 16, 32} {
		seen := map[int]bool{}
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				rank := bayer(n, x, y)
				if rank < 0 || rank >= n*n || seen[rank] {
					t.Fatalf("invalid rank %d for Bayer %d", rank, n)
				}
				seen[rank] = true
			}
		}
	}
}

func TestDiffusionReferences(t *testing.T) {
	// Published kernels produce these fixed results on an eight-step ramp.
	// These detect coefficient, scan-order, carry-row, and Atkinson divisor errors.
	ramp := []uint8{0, 32, 64, 96, 128, 160, 192, 255}
	img := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x, v := range ramp {
			img.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	cases := []struct {
		id   string
		rows []string
	}{{"floyd-steinberg", []string{"00010111", "00001011", "00101101", "00010111"}}, {"atkinson", []string{"00001111", "00001011", "00011011", "00000111"}}}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			out, err := Process(context.Background(), img, Config{Algorithm: tc.id})
			if err != nil {
				t.Fatal(err)
			}
			for y, row := range tc.rows {
				for x, want := range row {
					got := byte('0')
					if out.NRGBAAt(x, y).R == 255 {
						got = '1'
					}
					if got != byte(want) {
						t.Fatalf("(%d,%d)=%c, want %c", x, y, got, want)
					}
				}
			}
		})
	}
}
func TestKernelConservation(t *testing.T) {
	for id, k := range kernels {
		total := 0.0
		for _, e := range k {
			if e.y < 0 || (e.y == 0 && e.x <= 0) || e.weight <= 0 {
				t.Fatalf("invalid causal weight in %s", id)
			}
			total += e.weight
		}
		want := 1.0
		if id == "atkinson" {
			want = .75
		}
		if id == "steven-pigeon" {
			want = 6.0 / 7
		}
		if math.Abs(total-want) > 1e-12 {
			t.Errorf("%s sum %g want %g", id, total, want)
		}
	}
}
func TestCatalogDeterminismPaletteAlphaAndPurity(t *testing.T) {
	img := fixture(31, 27)
	original := append([]byte(nil), img.Pix...)
	palette := mustPalette(t, "pico-8")
	membership := map[[3]uint8]bool{}
	for _, p := range palette {
		membership[[3]uint8{p.R, p.G, p.B}] = true
	}
	seen := map[string]bool{}
	for _, algorithm := range Catalog() {
		t.Run(algorithm.ID, func(t *testing.T) {
			if seen[algorithm.ID] {
				t.Fatal("duplicate algorithm")
			}
			seen[algorithm.ID] = true
			cfg := Config{Algorithm: algorithm.ID, Palette: palette, Seed: 42, Serpentine: true}
			a, err := Process(context.Background(), img, cfg)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Process(context.Background(), img, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a.Pix, b.Pix) {
				t.Fatal("non-deterministic output")
			}
			for y := 0; y < img.Rect.Dy(); y++ {
				for x := 0; x < img.Rect.Dx(); x++ {
					c, src := a.NRGBAAt(x, y), img.NRGBAAt(x, y)
					if c.A != src.A {
						t.Fatalf("alpha changed at %d,%d", x, y)
					}
					if c.A == 0 {
						if c != (color.NRGBA{}) {
							t.Fatal("transparent pixel has hidden color")
						}
					} else if !membership[[3]uint8{c.R, c.G, c.B}] {
						t.Fatalf("out-of-palette pixel %#v", c)
					}
				}
			}
		})
	}
	if len(seen) != 41 {
		t.Fatalf("catalog has %d algorithms, want 41", len(seen))
	}
	if !bytes.Equal(original, img.Pix) {
		t.Fatal("source mutated")
	}
	copyCatalog := Catalog()
	copyCatalog[0].ID = "broken"
	if Catalog()[0].ID == "broken" {
		t.Fatal("catalog exposes shared memory")
	}
}
func TestZeroStrengthAndSeed(t *testing.T) {
	img := fixture(48, 33)
	reference, err := Process(context.Background(), img, Config{Algorithm: "threshold"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"floyd-steinberg", "riemersma", "bayer-8", "random", "blue-noise"} {
		out, err := Process(context.Background(), img, Config{Algorithm: id, Strength: Float(0)})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.Pix, reference.Pix) {
			t.Errorf("zero strength differs for %s", id)
		}
	}
	for _, id := range []string{"random", "blue-noise"} {
		a, _ := Process(context.Background(), img, Config{Algorithm: id, Seed: 1})
		b, _ := Process(context.Background(), img, Config{Algorithm: id, Seed: 2})
		if bytes.Equal(a.Pix, b.Pix) {
			t.Errorf("seed did not affect %s", id)
		}
	}
}
func TestBlueNoiseRankPermutation(t *testing.T) {
	tile, err := blueNoiseTile(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[float64]bool{}
	for _, v := range tile.values {
		if v <= 0 || v >= 1 || seen[v] {
			t.Fatalf("invalid or repeated threshold %g", v)
		}
		seen[v] = true
	}
	if len(seen) != 256 {
		t.Fatal("wrong tile size")
	}
}
func TestRiemersmaVisitsRectangles(t *testing.T) {
	for _, dims := range [][2]int{{1, 129}, {129, 1}, {3, 17}, {31, 32}, {33, 31}} {
		img := image.NewNRGBA(image.Rect(0, 0, dims[0], dims[1]))
		for y := 0; y < dims[1]; y++ {
			for x := 0; x < dims[0]; x++ {
				img.SetNRGBA(x, y, color.NRGBA{99, 117, 137, 255})
			}
		}
		out, err := Process(context.Background(), img, Config{Algorithm: "riemersma"})
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < dims[1]; y++ {
			for x := 0; x < dims[0]; x++ {
				c := out.NRGBAAt(x, y)
				if c.R != 0 && c.R != 255 {
					t.Fatalf("unvisited %d,%d in %v: %#v", x, y, dims, c)
				}
			}
		}
	}
}
func TestGeometryMasksAndNoCrossGap(t *testing.T) {
	source := fixture(20, 10)
	out, err := Process(context.Background(), source, Config{Algorithm: "atkinson", Crop: &Rect{2, 2, 10, 6}, Width: 15, PixelScale: 4})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != image.Rect(0, 0, 15, 9) {
		t.Fatalf("bounds %v", out.Bounds())
	}
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{100, 110, 120, 255})
		}
	}
	masks := []*Mask{{Shape: "rectangle", X: 2, Y: 2, Width: 3, Height: 3}, {Shape: "circle", X: 4, Y: 4, Radius: 2}, {Shape: "circle", X: 4, Y: 4, Radius: 2, Invert: true}}
	for _, mask := range masks {
		out, err := Process(context.Background(), img, Config{Mask: mask})
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				c := out.NRGBAAt(x, y)
				if !maskContains(mask, x, y, 8, 8) {
					if c != img.NRGBAAt(x, y) {
						t.Fatal("unselected source changed")
					}
				} else if c.R != 0 && c.R != 255 {
					t.Fatal("selected pixel unquantized")
				}
			}
		}
	}
	for _, id := range []string{"atkinson", "jarvis-judice-ninke", "stevenson-arce", "riemersma"} {
		a, b := image.NewNRGBA(image.Rect(0, 0, 17, 11)), image.NewNRGBA(image.Rect(0, 0, 17, 11))
		for y := 0; y < 11; y++ {
			for x := 0; x < 17; x++ {
				c := color.NRGBA{123, 123, 123, 255}
				if x == 7 {
					c = color.NRGBA{}
				}
				a.SetNRGBA(x, y, c)
				if x < 7 {
					c = color.NRGBA{199, 199, 199, 255}
				}
				b.SetNRGBA(x, y, c)
			}
		}
		aa, _ := Process(context.Background(), a, Config{Algorithm: id})
		bb, _ := Process(context.Background(), b, Config{Algorithm: id})
		for y := 0; y < 11; y++ {
			for x := 8; x < 17; x++ {
				if aa.NRGBAAt(x, y) != bb.NRGBAAt(x, y) {
					t.Fatalf("%s diffuses across transparent gap at %d,%d", id, x, y)
				}
			}
		}
	}
}
func TestAlphaCorrectSamplingAndImageMask(t *testing.T) {
	img := image.NewNRGBA(image.Rect(4, 5, 6, 6))
	img.SetNRGBA(4, 5, color.NRGBA{255, 0, 0, 0})
	img.SetNRGBA(5, 5, color.NRGBA{0, 0, 255, 255})
	out, err := resample(context.Background(), img, img.Bounds(), 4, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	for x := 0; x < 4; x++ {
		c := out.NRGBAAt(x, 0)
		if c.A > 0 && (c.R != 0 || c.B != 255) {
			t.Fatalf("transparent red leaked into bilinear sample %#v", c)
		}
	}
	maskImage := image.NewGray(image.Rect(0, 0, 2, 1))
	maskImage.SetGray(1, 0, color.Gray{255})
	mask := &Mask{Shape: "image", Image: maskImage}
	if maskContains(mask, 0, 0, 4, 1) || !maskContains(mask, 3, 0, 4, 1) {
		t.Fatal("image mask sampled incorrectly")
	}
}

func TestValidation(t *testing.T) {
	cases := []Config{{Algorithm: "unknown"}, {Brightness: math.NaN()}, {Gamma: Float(0)}, {Contrast: Float(-1)}, {Saturation: Float(math.Inf(1))}, {Strength: Float(3)}, {Threshold: Float(-.1)}, {Width: MaxDimension + 1}, {Width: MaxDimension, Height: MaxDimension}, {PixelScale: 257}, {Crop: &Rect{Width: 0, Height: 2}}, {Mask: &Mask{Shape: "triangle"}}, {Mask: &Mask{Shape: "circle", Radius: 0}}, {ResizeFilter: "cubic"}, {ColorSpace: "lab"}, {Effects: Effects{Noise: 2}}, {Palette: []color.NRGBA{{A: 255}, {A: 128}}}}
	for i, cfg := range cases {
		if err := ValidateConfig(cfg); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	for _, cfg := range []Config{{}, {Contrast: Float(0), Saturation: Float(0), Strength: Float(0), Threshold: Float(0)}} {
		if err := ValidateConfig(cfg); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Process(context.Background(), fixture(2, 2), Config{Crop: &Rect{X: 1, Y: 1, Width: 2, Height: 2}}); err == nil {
		t.Fatal("out-of-bounds crop accepted")
	}
	if _, err := Process(context.Background(), fixture(16, 1), Config{Height: MaxDimension}); err == nil {
		t.Fatal("aspect ratio exceeded allocation bounds")
	}
	var nilImage *image.NRGBA
	if _, err := Process(context.Background(), nilImage, Config{}); err == nil {
		t.Fatal("typed nil image accepted")
	}
}

type cancelImage struct {
	image.Image
	cancel context.CancelFunc
	calls  int
}

func (i *cancelImage) At(x, y int) color.Color {
	i.calls++
	if i.calls == 32 {
		i.cancel()
	}
	return i.Image.At(x, y)
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Process(ctx, fixture(10, 10), Config{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	src := &cancelImage{Image: fixture(64, 64), cancel: cancel}
	out, err := Process(ctx, src, Config{})
	if !errors.Is(err, context.Canceled) || out != nil {
		t.Fatalf("partial output %#v error %v", out, err)
	}
	if _, err := ExtractPalette(ctx, fixture(10, 10), 4); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestPaletteParsingAndExtraction(t *testing.T) {
	p, err := ResolvePalette("", []string{"#abc", "000000"})
	if err != nil {
		t.Fatal(err)
	}
	if p[0] != (color.NRGBA{170, 187, 204, 255}) {
		t.Fatal(p)
	}
	for _, args := range []struct {
		name string
		hex  []string
	}{{"missing", nil}, {"mono", []string{"#fff", "#000"}}, {"", []string{"#000", "#000"}}, {"", []string{"#z00", "#fff"}}} {
		if _, err := ResolvePalette(args.name, args.hex); err == nil {
			t.Fatal("invalid palette accepted")
		}
	}
	catalogs := Palettes()
	for _, p := range catalogs {
		if _, err := ResolvePalette(p.ID, nil); err != nil {
			t.Fatal(err)
		}
	}
	catalogs[0].Colors[0] = "#ff0000"
	if Palettes()[0].Colors[0] == "#ff0000" {
		t.Fatal("palette catalog aliases memory")
	}
	img := fixture(64, 47)
	a, err := ExtractPalette(context.Background(), img, 8)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ExtractPalette(context.Background(), img, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || len(a) != 8 {
		t.Fatalf("palette results %v %v", a, b)
	}
	clear(img.Pix)
	if _, err := ExtractPalette(context.Background(), img, 4); err == nil {
		t.Fatal("transparent image palette accepted")
	}
}
func TestEffectsDeterminismAndAlpha(t *testing.T) {
	img := fixture(43, 29)
	cfg := Config{Algorithm: "atkinson", Seed: 7, Effects: Effects{Scanlines: .3, CRT: .4, Noise: .2, Glitch: 8, PixelSort: true}}
	a, err := Process(context.Background(), img, cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Process(context.Background(), img, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("effects non-deterministic")
	}
	// Moving effects must conserve the multiset of alpha values in each row.
	for y := 0; y < img.Rect.Dy(); y++ {
		before, after := map[uint8]int{}, map[uint8]int{}
		for x := 0; x < img.Rect.Dx(); x++ {
			before[img.NRGBAAt(x, y).A]++
			after[a.NRGBAAt(x, y).A]++
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("effects changed alpha values")
		}
	}
}
func TestEncoders(t *testing.T) {
	img := fixture(11, 7)
	for _, format := range Formats() {
		t.Run(format, func(t *testing.T) {
			var b bytes.Buffer
			if err := Encode(format, &b, img); err != nil {
				t.Fatal(err)
			}
			if b.Len() == 0 {
				t.Fatal("empty export")
			}
			if format == "png" {
				decoded, err := png.Decode(&b)
				if err != nil {
					t.Fatal(err)
				}
				for y := 0; y < 7; y++ {
					for x := 0; x < 11; x++ {
						if color.NRGBAModel.Convert(decoded.At(x, y)) != img.At(x, y) {
							t.Fatal("PNG changed pixels")
						}
					}
				}
			}
			if format == "svg" && !strings.Contains(b.String(), "fill-opacity=") {
				t.Fatal("SVG omitted partial alpha")
			}
		})
	}
	if err := Encode("invalid", &bytes.Buffer{}, img); err == nil {
		t.Fatal("unknown format accepted")
	}
	binary := image.NewNRGBA(image.Rect(0, 0, 9, 1))
	for x := 0; x < 9; x++ {
		v := uint8(0)
		if x%2 == 1 {
			v = 255
		}
		binary.SetNRGBA(x, 0, color.NRGBA{v, v, v, 255})
	}
	var b bytes.Buffer
	if err := Encode("pbm", &b, binary); err != nil {
		t.Fatal(err)
	}
	want := append([]byte("P4\n9 1\n"), 0xaa, 0x80)
	if !bytes.Equal(b.Bytes(), want) {
		t.Fatalf("PBM %v, want %v", b.Bytes(), want)
	}
}
func TestGIFExactPaletteAndBinaryAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 1))
	cs := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 127}, {15, 26, 37, 128}, {255, 255, 255, 0}}
	for x, c := range cs {
		img.SetNRGBA(x, 0, c)
	}
	var b bytes.Buffer
	if err := Encode("gif", &b, img); err != nil {
		t.Fatal(err)
	}
	out, err := gif.Decode(&b)
	if err != nil {
		t.Fatal(err)
	}
	for x, c := range cs {
		got := color.NRGBAModel.Convert(out.At(x, 0)).(color.NRGBA)
		if c.A < 128 {
			if got.A != 0 {
				t.Fatal("GIF transparency lost")
			}
		} else if got.R != c.R || got.G != c.G || got.B != c.B || got.A != 255 {
			t.Fatalf("GIF changed fitting palette color %v to %v", c, got)
		}
	}
	// A transparent slot plus 256 visible colors exceeds the GIF palette limit.
	many := image.NewNRGBA(image.Rect(0, 0, 257, 1))
	for x := 0; x < 256; x++ {
		many.SetNRGBA(x, 0, color.NRGBA{uint8(x), uint8(x ^ 77), uint8(x ^ 137), 255})
	}
	p, err := Palettize(many)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Palette) > 256 {
		t.Fatal("GIF palette overflow")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("writer failure") }
func TestEncoderPropagatesWriterErrors(t *testing.T) {
	for _, format := range Formats() {
		if err := Encode(format, failingWriter{}, fixture(3, 3)); err == nil {
			t.Errorf("%s swallowed writer error", format)
		}
	}
}

func BenchmarkFloydSteinberg(b *testing.B) {
	img := fixture(1024, 768)
	palette, _ := ResolvePalette("pico-8", nil)
	cfg := Config{Palette: palette, Serpentine: true}
	b.ReportAllocs()
	b.SetBytes(int64(len(img.Pix)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Process(context.Background(), img, cfg); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkBayer8(b *testing.B) {
	img := fixture(1024, 768)
	palette, _ := ResolvePalette("pico-8", nil)
	cfg := Config{Algorithm: "bayer-8", Palette: palette}
	b.ReportAllocs()
	b.SetBytes(int64(len(img.Pix)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Process(context.Background(), img, cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func TestPortableImageMaskRecipeValidation(t *testing.T) {
	cfg := Config{Mask: &Mask{Shape: "image"}}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("portable recipe rejected: %v", err)
	}
	if _, err := Process(context.Background(), fixture(8, 8), cfg); err == nil {
		t.Fatal("runtime mask omission accepted")
	}
}

func TestDirectPixelReadersMatchColorModel(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(2, 3, 18, 9))
	gray := image.NewGray(image.Rect(2, 3, 18, 9))
	nrgba64 := image.NewNRGBA64(image.Rect(2, 3, 18, 9))
	ycbcr := image.NewYCbCr(image.Rect(2, 3, 18, 9), image.YCbCrSubsampleRatio444)
	for y := 3; y < 9; y++ {
		for x := 2; x < 18; x++ {
			a := uint8((x*17 + y*13) % 256)
			c := color.NRGBA{uint8(x * 13), uint8(y * 29), uint8((x + y) * 11), a}
			rgba.Set(x, y, c)
			gray.Set(x, y, c)
			nrgba64.Set(x, y, c)
			i := ycbcr.YOffset(x, y)
			ycbcr.Y[i] = uint8(x * 13)
			j := ycbcr.COffset(x, y)
			ycbcr.Cb[j] = uint8(y * 29)
			ycbcr.Cr[j] = uint8((x + y) * 11)
		}
	}
	for _, img := range []image.Image{rgba, gray, nrgba64, ycbcr} {
		read := pixelReader(img)
		for y := 3; y < 9; y++ {
			for x := 2; x < 18; x++ {
				want := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				// NRGBA64 is already unassociated. Truncate its stored channels
				// directly. An RGBA interface round trip loses precision at low alpha.
				if src, ok := img.(*image.NRGBA64); ok {
					c := src.NRGBA64At(x, y)
					want = color.NRGBA{uint8(c.R >> 8), uint8(c.G >> 8), uint8(c.B >> 8), uint8(c.A >> 8)}
				}
				if got := read(x, y); got != want {
					t.Fatalf("%T pixel %d,%d %#v want %#v", img, x, y, got, want)
				}
			}
		}
	}
}

func TestCatalogChoicesProduceDistinctScreens(t *testing.T) {
	// A broad grayscale ramp includes every coverage level and several repeats of
	// the largest tile. This guards against accidentally counting aliases.
	img := image.NewNRGBA(image.Rect(0, 0, 128, 96))
	for y := 0; y < 96; y++ {
		for x := 0; x < 128; x++ {
			v := uint8((x*2 + y*3) % 256)
			img.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	seen := map[string]string{}
	for _, a := range Catalog() {
		out, err := Process(context.Background(), img, Config{Algorithm: a.ID, Seed: 41})
		if err != nil {
			t.Fatal(err)
		}
		fingerprint := string(out.Pix)
		if previous, ok := seen[fingerprint]; ok {
			t.Fatalf("%s and %s have identical screens", previous, a.ID)
		}
		seen[fingerprint] = a.ID
	}
}

func TestDiffusionFeedbackBoundAndLongHighStrengthImage(t *testing.T) {
	// Each corrected channel is in [0,1], as is its chosen palette channel.
	// At strength 2, the error magnitude is therefore <=2. Each incoming causal
	// stencil sums to <=1. Its next correction stays within 2 regardless of the
	// number of pixels traversed. Check this bound for every real kernel.
	for id, k := range kernels {
		sum := 0.0
		for _, e := range k {
			sum += e.weight
		}
		bound := 2 * sum
		if !finite(bound) || bound > 2+1e-12 {
			t.Fatalf("%s feedback bound %g", id, bound)
		}
		for _, src := range []float64{0, 1} {
			for _, incoming := range []float64{-bound, bound} {
				for _, bias := range []float64{-.5, .5} {
					v := clamp(src + incoming + bias)
					for _, chosen := range []float64{0, 1} {
						err := 2 * (v - chosen)
						if !finite(err) || math.Abs(err) > 2 {
							t.Fatalf("%s unbounded corrected error %g", id, err)
						}
					}
				}
			}
		}
	}
	// Wide, high-strength scans must retain both tones to the final rows. Raw
	// amplified feedback would eventually dominate the source and collapse them.
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 1024; x++ {
			img.SetNRGBA(x, y, color.NRGBA{127, 127, 127, 255})
		}
	}
	for _, id := range []string{"floyd-steinberg", "shiau-fan-2", "simple-2d"} {
		out, err := Process(context.Background(), img, Config{Algorithm: id, Strength: Float(2)})
		if err != nil {
			t.Fatal(err)
		}
		white := 0
		total := 0
		for y := 224; y < 256; y++ {
			for x := 0; x < 1024; x++ {
				c := out.NRGBAAt(x, y)
				if c != (color.NRGBA{0, 0, 0, 255}) && c != (color.NRGBA{255, 255, 255, 255}) {
					t.Fatal("invalid long-image pixel")
				}
				if c.R == 255 {
					white++
				}
				total++
			}
		}
		ratio := float64(white) / float64(total)
		if ratio < .25 || ratio > .75 {
			t.Fatalf("%s amplified feedback collapsed tail coverage to %g", id, ratio)
		}
	}
}

func TestGIFInvisibleRGBDoesNotBiasOverflowPalette(t *testing.T) {
	a, b := image.NewNRGBA(image.Rect(0, 0, 512, 4)), image.NewNRGBA(image.Rect(0, 0, 512, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 512; x++ {
			b.SetNRGBA(x, y, color.NRGBA{255, 0, 0, 127})
		}
	}
	for x := 0; x < 256; x++ {
		c := color.NRGBA{uint8(x), uint8(x ^ 77), uint8(x ^ 137), 255}
		a.SetNRGBA(x, 0, c)
		b.SetNRGBA(x, 0, c)
	}
	// 256 visible colors plus transparency forces median cut in both cases.
	var aa, bb bytes.Buffer
	if err := Encode("gif", &aa, a); err != nil {
		t.Fatal(err)
	}
	if err := Encode("gif", &bb, b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(aa.Bytes(), bb.Bytes()) {
		t.Fatal("hidden RGB biased GIF overflow palette")
	}
}
