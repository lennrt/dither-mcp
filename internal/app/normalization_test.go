package app

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func normProfile(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../imagemeta/testdata/linear-srgb.icc")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func normEXIF(o uint16) []byte {
	b := make([]byte, 26)
	copy(b, "II")
	binary.LittleEndian.PutUint16(b[2:], 42)
	binary.LittleEndian.PutUint32(b[4:], 8)
	binary.LittleEndian.PutUint16(b[8:], 1)
	binary.LittleEndian.PutUint16(b[10:], 274)
	binary.LittleEndian.PutUint16(b[12:], 3)
	binary.LittleEndian.PutUint32(b[14:], 1)
	binary.LittleEndian.PutUint16(b[18:], o)
	return b
}
func normPNGChunk(kind string, p []byte) []byte {
	b := make([]byte, len(p)+12)
	binary.BigEndian.PutUint32(b, uint32(len(p)))
	copy(b[4:], kind)
	copy(b[8:], p)
	binary.BigEndian.PutUint32(b[len(b)-4:], crc32.ChecksumIEEE(b[4:len(b)-4]))
	return b
}
func normPNG(t *testing.T, im image.Image, o uint16, profile []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	r := append([]byte(nil), b.Bytes()[:33]...)
	if o != 0 {
		r = append(r, normPNGChunk("eXIf", normEXIF(o))...)
	}
	if profile != nil {
		var compressed bytes.Buffer
		compressed.WriteString("Original\x00\x00")
		z := zlib.NewWriter(&compressed)
		_, _ = z.Write(profile)
		_ = z.Close()
		r = append(r, normPNGChunk("iCCP", compressed.Bytes())...)
	}
	return append(r, b.Bytes()[33:]...)
}
func normSegment(marker byte, p []byte) []byte {
	b := []byte{255, marker, 0, 0}
	binary.BigEndian.PutUint16(b[2:], uint16(len(p)+2))
	return append(b, p...)
}
func normCall(t *testing.T, s *Service, name string, q any) any {
	t.Helper()
	b, e := json.Marshal(q)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.Do(context.Background(), name, b)
	if e != nil {
		t.Fatalf("%s: %v", name, e)
	}
	return v
}
func normWrite(t *testing.T, root, name string, b []byte) {
	t.Helper()
	if e := os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
		t.Fatal(e)
	}
}

func TestNormalizationAcrossDecodedContainers(t *testing.T) {
	ctx := context.Background()
	p := normProfile(t)
	src := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	for y := range 3 {
		for x := range 2 {
			src.SetNRGBA(x, y, color.NRGBA{128, 128, 128, 255})
		}
	}
	var jpg, tif, bitmap bytes.Buffer
	if err := jpeg.Encode(&jpg, src, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	jb := append([]byte(nil), jpg.Bytes()[:2]...)
	jb = append(jb, normSegment(0xe1, append([]byte("Exif\x00\x00"), normEXIF(6)...))...)
	jb = append(jb, normSegment(0xe2, append([]byte("ICC_PROFILE\x00\x01\x01"), p...))...)
	jb = append(jb, jpg.Bytes()[2:]...)
	if err := tiff.Encode(&tif, src, nil); err != nil {
		t.Fatal(err)
	}
	// Append a replacement IFD0. Pixel offsets in the original directory remain
	// valid, while the two new metadata entries refer to this same file.
	tb := append([]byte(nil), tif.Bytes()...)
	old := int(binary.LittleEndian.Uint32(tb[4:]))
	n := int(binary.LittleEndian.Uint16(tb[old:]))
	dir := make([]byte, 2+12*(n+2)+4)
	binary.LittleEndian.PutUint16(dir, uint16(n+2))
	copy(dir[2:], tb[old+2:old+2+12*n])
	copy(dir[2+12*n:], normEXIF(6)[10:22])
	entry := dir[2+12*(n+1):]
	binary.LittleEndian.PutUint16(entry, 34675)
	binary.LittleEndian.PutUint16(entry[2:], 7)
	binary.LittleEndian.PutUint32(entry[4:], uint32(len(p)))
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(tb)+len(dir)))
	entries := make([][]byte, n+2)
	for i := range entries {
		entries[i] = append([]byte(nil), dir[2+12*i:2+12*(i+1)]...)
	}
	sort.Slice(entries, func(i, j int) bool {
		return binary.LittleEndian.Uint16(entries[i]) < binary.LittleEndian.Uint16(entries[j])
	})
	for i, e := range entries {
		copy(dir[2+12*i:], e)
	}
	binary.LittleEndian.PutUint32(tb[4:], uint32(len(tb)))
	tb = append(tb, dir...)
	tb = append(tb, p...)
	if err := bmp.Encode(&bitmap, src); err != nil {
		t.Fatal(err)
	}
	bb := make([]byte, 138)
	copy(bb, bitmap.Bytes()[:54])
	binary.LittleEndian.PutUint32(bb[14:], 124)
	binary.LittleEndian.PutUint32(bb[10:], 138)
	binary.LittleEndian.PutUint32(bb[70:], 0x4d424544)
	pixels := bitmap.Bytes()[54:]
	binary.LittleEndian.PutUint32(bb[126:], uint32(124+len(pixels)))
	binary.LittleEndian.PutUint32(bb[130:], uint32(len(p)))
	bb = append(bb, pixels...)
	bb = append(bb, p...)
	binary.LittleEndian.PutUint32(bb[2:], uint32(len(bb)))
	for name, b := range map[string][]byte{"jpeg": jb, "png": normPNG(t, src, 6, p), "tiff": tb, "bmp": bb} {
		im, f, info, err := decodeNormalized(ctx, b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		w, h := 3, 2
		if name == "bmp" {
			w, h = 2, 3
		}
		if f != name || im.Bounds() != image.Rect(0, 0, w, h) || !info.ColorConverted {
			t.Fatalf("%s: %s %+v %+v", name, f, im.Bounds(), info)
		}
		c := color.NRGBAModel.Convert(im.At(0, 0)).(color.NRGBA)
		if c.R < 187 || c.R > 189 || c.A != 255 {
			t.Fatalf("%s: %+v", name, c)
		}
	}
	wb, err := os.ReadFile("../imagemeta/testdata/oriented.webp")
	if err != nil {
		t.Fatal(err)
	}
	im, f, info, err := decodeNormalized(ctx, wb)
	if err != nil {
		t.Fatal(err)
	}
	if f != "webp" || im.Bounds() != image.Rect(0, 0, 3, 2) || info.EXIFOrientation != 6 || !info.ColorConverted {
		t.Fatalf("webp: %+v %+v", im.Bounds(), info)
	}
	// Rotated top left is source (0,2): encoded linear (180,200,220).
	c := color.NRGBAModel.Convert(im.At(0, 0)).(color.NRGBA)
	if c.R < 218 || c.R > 220 || c.G < 228 || c.G > 230 || c.B < 238 || c.B > 240 {
		t.Fatalf("webp samples: %+v", c)
	}
}

func TestNormalizationSharedWorkflows(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	src := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	values := []uint8{0, 32, 64, 96, 128, 255}
	for y := range 3 {
		for x := range 2 {
			v := values[y*2+x]
			src.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	b := normPNG(t, src, 6, normProfile(t))
	normWrite(t, root, "source.png", b)
	inspect := normCall(t, s, "dither_inspect", InputRequest{"source.png"}).(Inspection)
	if inspect.Width != 3 || inspect.Height != 2 || inspect.Normalization.StoredWidth != 2 || inspect.Normalization.StoredHeight != 3 || inspect.Normalization.EXIFOrientation != 6 || inspect.SHA256 != digest(b) {
		t.Fatalf("inspect: %+v", inspect)
	}
	preview := normCall(t, s, "dither_preview", PreviewRequest{Input: "source.png", Width: 3}).(PreviewResult)
	pb, err := base64.StdEncoding.DecodeString(preview.Data)
	if err != nil {
		t.Fatal(err)
	}
	pi, err := png.Decode(bytes.NewReader(pb))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Width != 3 || preview.Height != 2 || color.NRGBAModel.Convert(pi.At(0, 0)).(color.NRGBA).R != 188 {
		t.Fatal("preview did not normalize source")
	}
	extracted := normCall(t, s, "dither_palette_extract", ExtractRequest{Input: "source.png", Count: 6}).(map[string]any)
	colors := extracted["colors"].([]string)
	found := false
	for _, c := range colors {
		found = found || c == "#bcbcbc"
	}
	if !found {
		t.Fatalf("extraction did not use sRGB: %v", colors)
	}
	request := RenderRequest{Input: "source.png", Output: "crop.png", Colors: []string{"808080", "bcbcbc"}, Options: engine.Config{Algorithm: "threshold", Crop: &engine.Rect{X: 0, Y: 0, Width: 1, Height: 1}}}
	normCall(t, s, "dither_render", request)
	out, _, _, err := s.decode(context.Background(), "crop.png")
	if err != nil {
		t.Fatal(err)
	}
	if color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA).R != 188 {
		t.Fatal("crop coordinates or profile conversion were incorrect")
	}
	reinspect := normCall(t, s, "dither_inspect", InputRequest{"crop.png"}).(Inspection)
	if reinspect.Normalization.EXIFOrientation != 1 || reinspect.Normalization.ColorSource != "assumed-srgb" || reinspect.Normalization.ICCSHA256 != "" {
		t.Fatal("source metadata propagated to output")
	}
	normCall(t, s, "dither_compare", CompareRequest{RenderRequest: RenderRequest{Input: "source.png", Output: "compare.png", Options: engine.Config{Width: 3}}, Algorithms: []string{"threshold"}, Columns: 1})
	normCall(t, s, "dither_animate", AnimateRequest{RenderRequest: RenderRequest{Input: "source.png", Output: "sheet.png", Options: engine.Config{Width: 3}}, Effect: "wave", Frames: 2, Columns: 2})
}

func TestMaskNormalizationAndFailureBeforePublication(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y := range 2 {
		for x := range 3 {
			src.SetNRGBA(x, y, color.NRGBA{90, 90, 90, 255})
		}
	}
	normWrite(t, root, "source.png", normPNG(t, src, 0, nil))
	mask := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	for y := range 3 {
		for x := range 2 {
			mask.SetNRGBA(x, y, color.NRGBA{A: 255})
		}
	}
	mask.SetNRGBA(0, 2, color.NRGBA{100, 100, 100, 255})
	normWrite(t, root, "mask.png", normPNG(t, mask, 6, normProfile(t)))
	normCall(t, s, "dither_render", RenderRequest{Input: "source.png", Output: "masked.png", MaskInput: "mask.png", Options: engine.Config{Algorithm: "threshold"}})
	out, _, _, err := s.decode(context.Background(), "masked.png")
	if err != nil {
		t.Fatal(err)
	}
	if color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA).R != 0 || color.NRGBAModel.Convert(out.At(1, 0)).(color.NRGBA).R != 90 {
		t.Fatal("mask orientation or color conversion was skipped")
	}
	bad := normProfile(t)
	copy(bad[16:20], "CMYK")
	normWrite(t, root, "bad.png", normPNG(t, src, 0, bad))
	_, err = s.Do(context.Background(), "dither_render", []byte(`{"input":"bad.png","output":"must-not-exist.png"}`))
	if err == nil || !strings.Contains(err.Error(), "sRGB") {
		t.Fatalf("unsupported profile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-exist.png")); !os.IsNotExist(err) {
		t.Fatal("failed normalization published an output")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".dither-") {
			t.Fatal("failed normalization left staging files")
		}
	}
}
