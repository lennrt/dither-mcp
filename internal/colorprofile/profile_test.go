package colorprofile

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixture(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".icc"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func applyPixel(t testing.TB, transform *Transform, values [3]uint16) color.NRGBA64 {
	t.Helper()
	img := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	img.SetNRGBA64(0, 0, color.NRGBA64{R: values[0], G: values[1], B: values[2], A: 43210})
	out, err := transform.Apply(context.Background(), img)
	if err != nil {
		t.Fatal(err)
	}
	return out.(*image.NRGBA64).NRGBA64At(0, 0)
}

func closeRGB(t testing.TB, actual color.NRGBA64, expected [3]uint16, tolerance int) {
	t.Helper()
	for i, channel := range []uint16{actual.R, actual.G, actual.B} {
		if difference := int(channel) - int(expected[i]); difference > tolerance || difference < -tolerance {
			t.Errorf("channel %d: got %d, want %d within %d", i, channel, expected[i], tolerance)
		}
	}
	if actual.A != 43210 {
		t.Fatalf("alpha changed to %d", actual.A)
	}
}

func TestIndependentLittleCMSVectors(t *testing.T) {
	var oracle struct {
		Version  int `json:"littlecms_version"`
		Profiles []struct {
			ID      string `json:"id"`
			Vectors []struct {
				Input [3]uint16 `json:"input"`
				SRGB  [3]uint16 `json:"srgb"`
			} `json:"vectors"`
		} `json:"profiles"`
	}
	b, err := os.ReadFile("testdata/littlecms-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &oracle); err != nil {
		t.Fatal(err)
	}
	if oracle.Version != 2190 || len(oracle.Profiles) != 3 {
		t.Fatalf("unexpected oracle provenance: %d/%d", oracle.Version, len(oracle.Profiles))
	}
	for _, profile := range oracle.Profiles {
		t.Run(profile.ID, func(t *testing.T) {
			transform, err := Parse(fixture(t, profile.ID))
			if err != nil {
				t.Fatal(err)
			}
			if len(profile.Vectors) != 15 {
				t.Fatal("oracle vector count changed")
			}
			for i, v := range profile.Vectors {
				t.Run(string(rune('A'+i)), func(t *testing.T) {
					// The two sRGB destination matrices differ by ICC rounding.
					// This bound is less than one eighth of an 8-bit channel step.
					closeRGB(t, applyPixel(t, transform, v.Input), v.SRGB, 32)
				})
			}
		})
	}
}

func TestPublishedCSSColorVectors(t *testing.T) {
	// The CSS Color 4 introductory leaf example gives this common sRGB value.
	// Its P3 and A98 coordinates are independent of this package's matrix math.
	expected := [3]uint16{27254, 33008, 24027}
	for _, tc := range []struct {
		name  string
		input [3]uint16
	}{{"display-p3", [3]uint16{28385, 32839, 24871}}, {"adobe-rgb", [3]uint16{28895, 32749, 24516}}, {"srgb", expected}} {
		t.Run(tc.name, func(t *testing.T) {
			transform, err := Parse(fixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			closeRGB(t, applyPixel(t, transform, tc.input), expected, 16)
		})
	}
}

func TestSystemProfileCompatibility(t *testing.T) {
	// These read-only checks do not distribute any platform profile bytes.
	for _, name := range []string{"Display P3.icc", "AdobeRGB1998.icc", "sRGB Profile.icc"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("/System/Library/ColorSync/Profiles", name))
			if errors.Is(err, os.ErrNotExist) {
				t.Skip("platform profile is not installed")
			}
			if err != nil {
				t.Fatal(err)
			}
			transform, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			actual := applyPixel(t, transform, [3]uint16{32768, 32768, 32768})
			if actual.R < 32000 || actual.R > 34000 || actual.G < 32000 || actual.G > 34000 || actual.B < 32000 || actual.B > 34000 {
				t.Fatalf("neutral conversion: %v", actual)
			}
		})
	}
}

func locate(data []byte, name string) (entry, offset, size int) {
	for i := 0; i < int(u32(data[128:132])); i++ {
		p := 132 + 12*i
		if string(data[p:p+4]) == name {
			return p, int(u32(data[p+4:])), int(u32(data[p+8:]))
		}
	}
	panic("test fixture lacks " + name)
}

func TestMalformedProfiles(t *testing.T) {
	for _, tc := range []struct {
		name   string
		modify func([]byte) []byte
	}{
		{"short", func(p []byte) []byte { return p[:100] }},
		{"oversized", func([]byte) []byte { return make([]byte, MaxProfileBytes+1) }},
		{"size-mismatch", func(p []byte) []byte { p[3] ^= 4; return p }},
		{"signature", func(p []byte) []byte { copy(p[36:40], "bad!"); return p }},
		{"version", func(p []byte) []byte { p[8] = 5; return p }},
		{"version-bcd", func(p []byte) []byte { p[9] = 0xaf; return p }},
		{"device-link", func(p []byte) []byte { copy(p[12:16], "link"); return p }},
		{"cmyk", func(p []byte) []byte { copy(p[16:20], "CMYK"); return p }},
		{"gray", func(p []byte) []byte { copy(p[16:20], "GRAY"); return p }},
		{"lab-pcs", func(p []byte) []byte { copy(p[20:24], "Lab "); return p }},
		{"intent", func(p []byte) []byte { binary.BigEndian.PutUint32(p[64:], 4); return p }},
		{"non-d50", func(p []byte) []byte { binary.BigEndian.PutUint32(p[68:], 32768); return p }},
		{"header-reserved", func(p []byte) []byte { p[100] = 1; return p }},
		{"zero-tags", func(p []byte) []byte { binary.BigEndian.PutUint32(p[128:], 0); return p }},
		{"tag-limit", func(p []byte) []byte { binary.BigEndian.PutUint32(p[128:], MaxTags+1); return p }},
		{"tag-overflow", func(p []byte) []byte { binary.BigEndian.PutUint32(p[136:], 0xfffffffc); return p }},
		{"tag-in-table", func(p []byte) []byte { binary.BigEndian.PutUint32(p[136:], 132); return p }},
		{"tag-unaligned", func(p []byte) []byte { binary.BigEndian.PutUint32(p[136:], u32(p[136:])+1); return p }},
		{"tag-short", func(p []byte) []byte { binary.BigEndian.PutUint32(p[140:], 7); return p }},
		{"duplicate", func(p []byte) []byte { copy(p[144:148], p[132:136]); return p }},
		{"partial-overlap", func(p []byte) []byte {
			a, x, _ := locate(p, "rXYZ")
			b, _, _ := locate(p, "gXYZ")
			_ = a
			binary.BigEndian.PutUint32(p[b+4:], uint32(x+8))
			return p
		}},
		{"missing-matrix", func(p []byte) []byte { e, _, _ := locate(p, "rXYZ"); copy(p[e:e+4], "xxxx"); return p }},
		{"missing-curve", func(p []byte) []byte { e, _, _ := locate(p, "rTRC"); copy(p[e:e+4], "xxxx"); return p }},
		{"wrong-matrix-type", func(p []byte) []byte { _, o, _ := locate(p, "rXYZ"); copy(p[o:o+4], "curv"); return p }},
		{"extra-xyz-values", func(p []byte) []byte { e, _, _ := locate(p, "rXYZ"); binary.BigEndian.PutUint32(p[e+8:], 24); return p }},
		{"zero-matrix", func(p []byte) []byte {
			for _, name := range []string{"rXYZ", "gXYZ", "bXYZ"} {
				_, o, _ := locate(p, name)
				clear(p[o+8 : o+20])
			}
			return p
		}},
		{"curve-reserved", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); p[o+4] = 1; return p }},
		{"curve-kind", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); p[o+9] = 5; return p }},
		{"curve-gamma", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); clear(p[o+12 : o+16]); return p }},
		{"curve-scale", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); clear(p[o+16 : o+20]); return p }},
		{"lut-tag", func(p []byte) []byte { e, _, _ := locate(p, "rTRC"); copy(p[e:e+4], "A2B0"); return p }},
		{"reverse-lut-tag", func(p []byte) []byte { e, _, _ := locate(p, "rTRC"); copy(p[e:e+4], "B2A0"); return p }},
		{"mab-type", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); copy(p[o:o+4], "mAB "); return p }},
		{"mba-type", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); copy(p[o:o+4], "mBA "); return p }},
		{"lut8-type", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); copy(p[o:o+4], "mft1"); return p }},
		{"lut16-type", func(p []byte) []byte { _, o, _ := locate(p, "rTRC"); copy(p[o:o+4], "mft2"); return p }},
		{"hdr-cicp", func(p []byte) []byte { e, _, _ := locate(p, "rTRC"); copy(p[e:e+4], "cicp"); return p }},
		{"invalid-chad", func(p []byte) []byte { _, o, _ := locate(p, "chad"); copy(p[o:o+4], "XYZ "); return p }},
		{"singular-chad", func(p []byte) []byte { _, o, n := locate(p, "chad"); clear(p[o+8 : o+n]); return p }},
		{"trailing-data", func(p []byte) []byte {
			p = append(p, 0, 0, 0, 0)
			binary.BigEndian.PutUint32(p, uint32(len(p)))
			return p
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.modify(fixture(t, "display-p3"))
			if _, err := Parse(data); err == nil || !strings.HasPrefix(err.Error(), "colorprofile:") {
				t.Fatalf("malformed profile accepted: %v", err)
			}
		})
	}
}

func TestSharedCurvesAndNegativeXYZ(t *testing.T) {
	data := fixture(t, "display-p3")
	_, offset, size := locate(data, "rTRC")
	for _, name := range []string{"gTRC", "bTRC"} {
		_, actualOffset, actualSize := locate(data, name)
		if actualOffset != offset || actualSize != size {
			t.Fatal("fixture must exercise shared curve data")
		}
	}
	_, redOffset, _ := locate(data, "rXYZ")
	if fixed16(data[redOffset+16:]) >= 0 {
		t.Fatal("fixture must exercise a negative XYZ colorant")
	}
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

type genericImage struct{ source image.Image }

func (i genericImage) Bounds() image.Rectangle { return i.source.Bounds() }
func (i genericImage) ColorModel() color.Model { return i.source.ColorModel() }
func (i genericImage) At(x, y int) color.Color { return i.source.At(x, y) }

func TestPixelAccessPreservesStandardSemantics(t *testing.T) {
	transform, err := FromPNG(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := image.Rect(0, 0, 4, 1)
	for _, img := range []image.Image{
		image.NewNRGBA64(r), image.NewNRGBA(r), image.NewRGBA64(r), image.NewRGBA(r),
		image.NewGray16(r), image.NewGray(r), image.NewYCbCr(r, image.YCbCrSubsampleRatio444),
	} {
		if editable, ok := img.(interface{ Set(int, int, color.Color) }); ok {
			for x, a := range []uint16{0, 1, 43210, 65535} {
				editable.Set(x, 0, color.NRGBA64{R: 12345, G: 23456, B: 34567, A: a})
			}
		}
		actual, err := transform.Apply(context.Background(), img)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := transform.Apply(context.Background(), genericImage{img})
		if err != nil {
			t.Fatal(err)
		}
		for x := range 4 {
			if actual.At(x, 0) != expected.At(x, 0) {
				t.Fatalf("%T pixel %d differs: %v / %v", img, x, actual.At(x, 0), expected.At(x, 0))
			}
		}
	}
}

func TestProfileDataOwnershipAndDescription(t *testing.T) {
	data := fixture(t, "display-p3")
	transform, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	before := applyPixel(t, transform, [3]uint16{12000, 23000, 34000})
	clear(data)
	if after := applyPixel(t, transform, [3]uint16{12000, 23000, 34000}); after != before {
		t.Fatal("transform retains mutable profile bytes")
	}
	data = fixture(t, "display-p3")
	_, o, n := locate(data, "desc")
	// Keep the tag's reserved bytes. Change only its untrusted description data.
	for i := o + 8; i < o+n; i++ {
		data[i] = byte(i)
	}
	modified, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if after := applyPixel(t, modified, [3]uint16{12000, 23000, 34000}); after != before {
		t.Fatal("description changed the numerical transform")
	}
}

func TestAlphaPrecisionBoundsAndConcurrency(t *testing.T) {
	transform, err := FromPNG(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	bounds := image.Rect(-4, 7, 2, 8)
	img := image.NewNRGBA64(bounds)
	alpha := []uint16{0, 1, 257, 32768, 43210, 65535}
	for i, a := range alpha {
		img.SetNRGBA64(bounds.Min.X+i, 7, color.NRGBA64{R: 12345, G: 23456, B: 34567, A: a})
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			out, err := transform.Apply(context.Background(), img)
			if err != nil {
				t.Error(err)
				return
			}
			if out.Bounds() != bounds {
				t.Error("bounds changed")
			}
			for i := range alpha {
				if out.At(bounds.Min.X+i, 7) != img.At(bounds.Min.X+i, 7) {
					t.Errorf("16-bit channels or alpha changed at %d: %v", i, out.At(bounds.Min.X+i, 7))
				}
			}
		})
	}
	wg.Wait()
	img8 := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for i, a := range []uint8{0, 1, 255} {
		img8.SetNRGBA(i, 0, color.NRGBA{R: 73, G: 121, B: 209, A: a})
	}
	out, err := transform.Apply(context.Background(), img8)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range []uint16{0, 257, 65535} {
		if c := out.(*image.NRGBA64).NRGBA64At(i, 0); c.R != 73*257 || c.G != 121*257 || c.B != 209*257 || c.A != a {
			t.Fatalf("8-bit unassociated channels changed: %v", c)
		}
	}
}

type boundedImage struct {
	bounds image.Rectangle
	cancel context.CancelFunc
	reads  int
}

func (i *boundedImage) Bounds() image.Rectangle { return i.bounds }
func (*boundedImage) ColorModel() color.Model   { return color.NRGBA64Model }
func (i *boundedImage) At(x, y int) color.Color {
	i.reads++
	if i.cancel != nil && y == i.bounds.Min.Y {
		i.cancel()
	}
	return color.NRGBA64{R: 12345, G: 23456, B: 34567, A: 43210}
}

func TestApplyAdmissionAndCancellation(t *testing.T) {
	transform, err := FromPNG(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := &boundedImage{bounds: image.Rect(0, 0, 2, 2)}
	if _, err := transform.Apply(ctx, src); !errors.Is(err, context.Canceled) || src.reads != 0 {
		t.Fatal("canceled input allocated or read pixels")
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	src.cancel = cancel
	if _, err := transform.Apply(ctx, src); !errors.Is(err, context.Canceled) || src.reads != 2 {
		t.Fatalf("row cancellation: %v, reads=%d", err, src.reads)
	}
	for _, bounds := range []image.Rectangle{
		{}, image.Rect(0, 0, MaxImagePixels, 2),
		{Min: image.Pt(math.MinInt, 0), Max: image.Pt(math.MaxInt, 1)},
	} {
		src := &boundedImage{bounds: bounds}
		if _, err := transform.Apply(context.Background(), src); err == nil || src.reads != 0 {
			t.Fatal("invalid dimensions admitted")
		}
	}
	var typedNil *image.NRGBA
	for _, src := range []image.Image{nil, typedNil} {
		if _, err := transform.Apply(context.Background(), src); err == nil {
			t.Fatal("nil image admitted")
		}
	}
	if _, err := transform.Apply(nil, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("nil context admitted")
	}
	for _, transform := range []*Transform{nil, {}} {
		if _, err := transform.Apply(context.Background(), image.NewNRGBA(image.Rect(0, 0, 1, 1))); err == nil {
			t.Fatal("unconstructed transform admitted")
		}
	}
}

func BenchmarkApplyProfile(b *testing.B) {
	transform, err := Parse(fixture(b, "display-p3"))
	if err != nil {
		b.Fatal(err)
	}
	img := image.NewNRGBA64(image.Rect(0, 0, 960, 640))
	for y := range 640 {
		for x := range 960 {
			img.SetNRGBA64(x, y, color.NRGBA64{R: uint16(x * 68), G: uint16(y * 102), B: 30000, A: 65535})
		}
	}
	b.SetBytes(960 * 640 * 8)
	b.ResetTimer()
	for b.Loop() {
		if _, err := transform.Apply(context.Background(), img); err != nil {
			b.Fatal(err)
		}
	}
}
