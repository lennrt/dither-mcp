package imagemeta

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"os"
	"testing"
)

func exifBytes(o uint16, order binary.ByteOrder) []byte {
	b := make([]byte, 26)
	copy(b, "II")
	if order == binary.BigEndian {
		copy(b, "MM")
	}
	order.PutUint16(b[2:], 42)
	order.PutUint32(b[4:], 8)
	order.PutUint16(b[8:], 1)
	order.PutUint16(b[10:], 0x112)
	order.PutUint16(b[12:], 3)
	order.PutUint32(b[14:], 1)
	order.PutUint16(b[18:], o)
	return b
}

func pngChunk(kind string, payload []byte) []byte {
	b := make([]byte, len(payload)+12)
	binary.BigEndian.PutUint32(b, uint32(len(payload)))
	copy(b[4:], kind)
	copy(b[8:], payload)
	binary.BigEndian.PutUint32(b[len(b)-4:], crc32.ChecksumIEEE(b[4:len(b)-4]))
	return b
}
func pngContainer(chunks ...[]byte) []byte {
	b := []byte("\x89PNG\r\n\x1a\n")
	for _, c := range chunks {
		b = append(b, c...)
	}
	return append(b, pngChunk("IEND", nil)...)
}
func iccp(profile []byte) []byte {
	var b bytes.Buffer
	b.WriteString("Original\x00\x00")
	w := zlib.NewWriter(&b)
	_, _ = w.Write(profile)
	_ = w.Close()
	return pngChunk("iCCP", b.Bytes())
}
func jpegSegment(marker byte, payload []byte) []byte {
	b := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(b[2:], uint16(len(payload)+2))
	return append(b, payload...)
}
func jpegContainer(parts ...[]byte) []byte {
	b := []byte{0xff, 0xd8}
	for _, p := range parts {
		b = append(b, p...)
	}
	return append(b, 0xff, 0xd9)
}
func riffChunk(kind string, payload []byte) []byte {
	b := make([]byte, 8)
	copy(b, kind)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(payload)))
	b = append(b, payload...)
	if len(payload)%2 != 0 {
		b = append(b, 0)
	}
	return b
}
func webpContainer(chunks ...[]byte) []byte {
	b := []byte("RIFF\x00\x00\x00\x00WEBP")
	for _, c := range chunks {
		b = append(b, c...)
	}
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	return b
}
func profileFixture(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/linear-srgb.icc")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAllOrientationsPreserveSamples(t *testing.T) {
	// The hand-written grids distinguish every mirror and rotation on a
	// non-square source. Nonzero bounds also exercise coordinate translation.
	want := [][]uint16{{1, 2, 3, 4, 5, 6}, {2, 1, 4, 3, 6, 5}, {6, 5, 4, 3, 2, 1}, {5, 6, 3, 4, 1, 2}, {1, 3, 5, 2, 4, 6}, {5, 3, 1, 6, 4, 2}, {6, 4, 2, 5, 3, 1}, {2, 4, 6, 1, 3, 5}}
	src := image.NewNRGBA64(image.Rect(4, 9, 6, 12))
	for y := 0; y < 3; y++ {
		for x := 0; x < 2; x++ {
			v := uint16(y*2 + x + 1)
			src.SetNRGBA64(x+4, y+9, color.NRGBA64{R: v, G: v + 17, B: v + 123, A: v * 10001})
		}
	}
	for o := 1; o <= 8; o++ {
		for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			m, err := Parse(context.Background(), exifBytes(uint16(o), order), "tiff")
			if err != nil {
				t.Fatal(err)
			}
			im, info, err := m.Apply(context.Background(), src)
			if err != nil {
				t.Fatal(err)
			}
			w, h := 2, 3
			if o >= 5 {
				w, h = h, w
			}
			if im.Bounds() != image.Rect(0, 0, w, h) || info.OrientationApplied != (o != 1) {
				t.Fatalf("orientation %d: bounds/info %+v %+v", o, im.Bounds(), info)
			}
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					v := want[o-1][y*w+x]
					got := color.NRGBA64Model.Convert(im.At(x, y)).(color.NRGBA64)
					if got != (color.NRGBA64{R: v, G: v + 17, B: v + 123, A: v * 10001}) {
						t.Fatalf("orientation %d pixel %d,%d: %+v", o, x, y, got)
					}
				}
			}
		}
	}
}

func TestEXIFContainersAndBounds(t *testing.T) {
	exif := exifBytes(6, binary.LittleEndian)
	for format, b := range map[string][]byte{
		"jpeg": jpegContainer(jpegSegment(0xe1, append([]byte("Exif\x00\x00"), exif...))),
		"png":  pngContainer(pngChunk("eXIf", exif)),
		"webp": webpContainer(riffChunk("EXIF", exif)), "tiff": exif,
	} {
		m, err := Parse(context.Background(), b, format)
		if err != nil || m.orientation != 6 {
			t.Fatalf("%s: %v %+v", format, err, m)
		}
	}
	for _, o := range []uint16{0, 9, 65535} {
		if _, err := Parse(context.Background(), exifBytes(o, binary.LittleEndian), "tiff"); err == nil {
			t.Fatalf("accepted orientation %d", o)
		}
	}
	bad := exifBytes(1, binary.LittleEndian)
	binary.LittleEndian.PutUint32(bad[4:], 0xfffffff0)
	if _, err := Parse(context.Background(), bad, "tiff"); err == nil {
		t.Fatal("accepted out-of-bounds IFD")
	}
	if _, err := Parse(context.Background(), pngContainer(pngChunk("eXIf", exif), pngChunk("eXIf", exif)), "png"); err == nil {
		t.Fatal("accepted duplicate EXIF")
	}
}

func TestICCContainersAndSequence(t *testing.T) {
	p := profileFixture(t)
	part := func(seq, total byte, raw []byte) []byte {
		return jpegSegment(0xe2, append(append([]byte("ICC_PROFILE\x00"), seq, total), raw...))
	}
	// A TIFF with one ICC entry points beyond its directory, not into it.
	tif := exifBytes(1, binary.LittleEndian)
	binary.LittleEndian.PutUint16(tif[10:], 34675)
	binary.LittleEndian.PutUint16(tif[12:], 7)
	binary.LittleEndian.PutUint32(tif[14:], uint32(len(p)))
	binary.LittleEndian.PutUint32(tif[18:], 26)
	tif = append(tif, p...)
	bmp := make([]byte, 138)
	copy(bmp, "BM")
	binary.LittleEndian.PutUint32(bmp[14:], 124)
	binary.LittleEndian.PutUint32(bmp[70:], 0x4d424544)
	binary.LittleEndian.PutUint32(bmp[126:], 124)
	binary.LittleEndian.PutUint32(bmp[130:], uint32(len(p)))
	bmp = append(bmp, p...)
	for format, b := range map[string][]byte{
		"jpeg": jpegContainer(part(2, 2, p[len(p)/2:]), part(1, 2, p[:len(p)/2])),
		"png":  pngContainer(iccp(p)), "webp": webpContainer(riffChunk("ICCP", p)), "tiff": tif, "bmp": bmp,
	} {
		m, err := Parse(context.Background(), b, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		src.SetNRGBA(0, 0, color.NRGBA{128, 128, 128, 73})
		im, info, err := m.Apply(context.Background(), src)
		if err != nil {
			t.Fatal(err)
		}
		got := color.NRGBAModel.Convert(im.At(0, 0)).(color.NRGBA)
		if got.R < 187 || got.R > 189 || got.G != got.R || got.B != got.R || got.A != 73 || info.ColorSource != "embedded-icc" || len(info.ICCSHA256) != 64 {
			t.Fatalf("%s: %+v %+v", format, got, info)
		}
	}
	for _, b := range [][]byte{jpegContainer(part(1, 2, p)), jpegContainer(part(1, 1, p), part(1, 1, p)), jpegContainer(part(0, 1, p)), jpegContainer(part(1, 2, p), part(2, 3, p))} {
		if _, err := Parse(context.Background(), b, "jpeg"); err == nil {
			t.Fatal("accepted invalid JPEG ICC sequence")
		}
	}
}

func TestPNGPrecedenceAndBoundedMetadata(t *testing.T) {
	p := profileFixture(t)
	g := []byte{0, 1, 0x86, 0xa0} // Gamma 1.0: linear input.
	for _, tc := range []struct {
		chunks    [][]byte
		source    string
		converted bool
	}{
		{nil, "assumed-srgb", false},
		{[][]byte{pngChunk("gAMA", g)}, "png-gamma-chromaticities", true},
		{[][]byte{pngChunk("gAMA", g), pngChunk("sRGB", []byte{0})}, "png-srgb", false},
		{[][]byte{pngChunk("gAMA", g), iccp(p)}, "embedded-icc", true},
		{[][]byte{iccp(p), pngChunk("cICP", []byte{1, 13, 0, 1})}, "png-cicp-srgb", false},
	} {
		m, err := Parse(context.Background(), pngContainer(tc.chunks...), "png")
		if err != nil {
			t.Fatal(err)
		}
		_, info, err := m.Apply(context.Background(), image.NewNRGBA(image.Rect(0, 0, 1, 1)))
		if err != nil || info.ColorSource != tc.source || info.ColorConverted != tc.converted {
			t.Fatalf("%s: %v %+v", tc.source, err, info)
		}
		if !tc.converted && info.ICCSHA256 != "" {
			t.Fatal("reported an ignored profile as selected")
		}
	}
	badCRC := pngChunk("eXIf", exifBytes(1, binary.LittleEndian))
	badCRC[len(badCRC)-1] ^= 1
	for _, b := range [][]byte{
		pngContainer(iccp(p), pngChunk("sRGB", []byte{0})),
		pngContainer(pngChunk("cICP", []byte{9, 16, 0, 1})),
		pngContainer(badCRC), pngContainer(iccp(make([]byte, (4<<20)+1))),
		pngContainer(pngChunk("eXIf", make([]byte, MaxEXIFBytes+1))),
		pngContainer(pngChunk("gAMA", []byte{0, 0, 0, 0})),
	} {
		if _, err := Parse(context.Background(), b, "png"); err == nil {
			t.Fatal("accepted invalid PNG metadata")
		}
	}
}

func TestCancellationAndUnsupportedSourceModels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Parse(ctx, nil, "png"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	m := &Metadata{orientation: 1, source: "assumed-srgb"}
	if _, _, err := m.Apply(ctx, image.NewNRGBA(image.Rect(0, 0, 1, 1))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, _, err := m.Apply(context.Background(), image.NewCMYK(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("accepted untagged CMYK")
	}
}

func FuzzMetadata(f *testing.F) {
	f.Add(byte(0), jpegContainer(jpegSegment(0xe1, append([]byte("Exif\x00\x00"), exifBytes(6, binary.LittleEndian)...))))
	f.Add(byte(1), pngContainer(iccp(profileFixture(f)), pngChunk("eXIf", exifBytes(5, binary.BigEndian))))
	f.Add(byte(2), webpContainer(riffChunk("EXIF", exifBytes(8, binary.LittleEndian))))
	f.Add(byte(3), exifBytes(2, binary.BigEndian))
	f.Add(byte(4), []byte("BM"))
	f.Add(byte(5), []byte("GIF89a"))
	f.Fuzz(func(t *testing.T, format byte, b []byte) {
		if len(b) > 1<<20 {
			t.Skip()
		}
		_, _ = Parse(context.Background(), b, []string{"jpeg", "png", "webp", "tiff", "bmp", "gif"}[format%6])
	})
}
