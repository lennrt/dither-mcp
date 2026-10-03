package app_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/app"
)

func workflowFixture(t *testing.T) (*app.Service, string, *image.NRGBA) {
	t.Helper()
	root := t.TempDir()
	source := image.NewNRGBA(image.Rect(0, 0, 24, 18))
	for y := 0; y < 18; y++ {
		for x := 0; x < 24; x++ {
			a := uint8(255)
			if x == 0 {
				a = 0
			}
			source.SetNRGBA(x, y, color.NRGBA{uint8(x * 11), uint8(y * 14), uint8((x*9 + y*17) % 256), a})
		}
	}
	workflowWritePNG(t, root, "input.png", source)
	s, err := app.New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, root, source
}
func workflowWritePNG(t *testing.T, root, path string, im image.Image) {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, path), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}
func workflowCall[T any](t *testing.T, s *app.Service, name string, q any) T {
	t.Helper()
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Do(context.Background(), name, b)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	wire, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded T
	if err = json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
func workflowRead(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func workflowDecode(t *testing.T, b []byte) image.Image {
	t.Helper()
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return im
}
func workflowArtifact(t *testing.T, root string, a app.Artifact) {
	t.Helper()
	b := workflowRead(t, root, a.Path)
	sum := sha256.Sum256(b)
	if a.Bytes != len(b) || a.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("artifact integrity mismatch %#v", a)
	}
	if a.Recipe == nil {
		t.Fatal("artifact omitted normalized recipe")
	}
}
func workflowEqualPixels(t *testing.T, a, b image.Image) {
	t.Helper()
	if a.Bounds().Dx() != b.Bounds().Dx() || a.Bounds().Dy() != b.Bounds().Dy() {
		t.Fatalf("bounds differ %v %v", a.Bounds(), b.Bounds())
	}
	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			aa := color.NRGBAModel.Convert(a.At(a.Bounds().Min.X+x, a.Bounds().Min.Y+y)).(color.NRGBA)
			bb := color.NRGBAModel.Convert(b.At(b.Bounds().Min.X+x, b.Bounds().Min.Y+y)).(color.NRGBA)
			if aa != bb {
				t.Fatalf("pixel %d,%d differs %#v %#v", x, y, aa, bb)
			}
		}
	}
}

func TestWorkflowRenderFormatsInspectAndPreview(t *testing.T) {
	s, root, source := workflowFixture(t)
	cfg := engine.Config{Algorithm: "atkinson", Width: 16, PixelScale: 2, Seed: 7, Contrast: engine.Float(1.2)}
	expectedCfg := cfg
	expectedCfg.Palette, _ = engine.ResolvePalette("gameboy", nil)
	expected, err := engine.Process(context.Background(), source, expectedCfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range engine.Formats() {
		t.Run(format, func(t *testing.T) {
			path := "render." + format
			a := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: path, Palette: "gameboy", Options: cfg, Format: format})
			workflowArtifact(t, root, a)
			if a.Width != 16 || a.Height != 12 || a.Frames != 1 || a.Format != format {
				t.Fatalf("metadata %#v", a)
			}
			b := workflowRead(t, root, path)
			switch format {
			case "png":
				workflowEqualPixels(t, workflowDecode(t, b), expected)
			case "gif":
				out := workflowDecode(t, b)
				for y := 0; y < 12; y++ {
					for x := 0; x < 16; x++ {
						want := expected.NRGBAAt(x, y)
						got := color.NRGBAModel.Convert(out.At(x, y)).(color.NRGBA)
						if want.A < 128 {
							if got.A != 0 {
								t.Fatal("GIF transparency changed")
							}
						} else if got.R != want.R || got.G != want.G || got.B != want.B || got.A != 255 {
							t.Fatalf("GIF visible color changed at %d,%d", x, y)
						}
					}
				}
			case "jpeg":
				out := workflowDecode(t, b)
				if out.Bounds().Dx() != 16 || out.Bounds().Dy() != 12 {
					t.Fatal("JPEG dimensions changed")
				}
			case "svg":
				decoder := xml.NewDecoder(bytes.NewReader(b))
				token, err := decoder.Token()
				if err != nil {
					t.Fatal(err)
				}
				element, ok := token.(xml.StartElement)
				if !ok || element.Name.Local != "svg" {
					t.Fatal("invalid SVG root")
				}
				attrs := map[string]string{}
				for _, a := range element.Attr {
					attrs[a.Name.Local] = a.Value
				}
				if attrs["width"] != "16" || attrs["height"] != "12" || attrs["shape-rendering"] != "crispEdges" {
					t.Fatal(attrs)
				}
			case "pbm":
				if !bytes.HasPrefix(b, []byte("P4\n16 12\n")) {
					t.Fatal("invalid PBM header")
				}
			case "ascii":
				lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
				if len(lines) != 12 {
					t.Fatal("ASCII row count")
				}
				for _, line := range lines {
					if len(line) != 16 {
						t.Fatal("ASCII row width")
					}
				}
			}
		})
	}
	inspection := workflowCall[app.Inspection](t, s, "dither_inspect", app.InputRequest{Input: "render.png"})
	if inspection.Width != 16 || inspection.Height != 12 || inspection.Frames != 1 || inspection.Format != "png" {
		t.Fatalf("inspection %#v", inspection)
	}
	preview := workflowCall[app.PreviewResult](t, s, "dither_preview", app.PreviewRequest{Input: "render.png", Width: 7})
	data, err := base64.StdEncoding.DecodeString(preview.Data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Width != 7 || preview.Height != 5 || preview.MIMEType != "image/png" {
		t.Fatal(preview)
	}
	out := workflowDecode(t, data)
	if out.Bounds().Dx() != preview.Width || out.Bounds().Dy() != preview.Height {
		t.Fatal("preview metadata does not match image")
	}
}

func TestWorkflowRecipeRoundTripAndExternalMask(t *testing.T) {
	s, root, _ := workflowFixture(t)
	recipe := app.Recipe{Version: 1, Colors: []string{"#242a25", "#eee9de"}, Options: engine.Config{Algorithm: "blue-noise", Width: 18, Seed: 123, Strength: engine.Float(.8), Saturation: engine.Float(0)}}
	saved := workflowCall[app.Artifact](t, s, "dither_recipe_save", app.RecipeSaveRequest{Output: "recipes/print.json", Recipe: recipe})
	workflowArtifact(t, root, saved)
	loaded := workflowCall[app.Recipe](t, s, "dither_recipe_load", app.InputRequest{Input: saved.Path})
	if !reflect.DeepEqual(loaded, recipe) {
		t.Fatalf("recipe changed %#v %#v", loaded, recipe)
	}
	a := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: "inline.png", Colors: recipe.Colors, Options: recipe.Options})
	b := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: "saved.png", Recipe: saved.Path})
	if a.SHA256 != b.SHA256 {
		t.Fatal("saved recipe changed output")
	}
	mask := image.NewGray(image.Rect(0, 0, 2, 1))
	mask.SetGray(1, 0, color.Gray{Y: 255})
	workflowWritePNG(t, root, "mask.png", mask)
	maskedRecipe := app.Recipe{Version: 1, Palette: "mono", Options: engine.Config{Algorithm: "threshold", Mask: &engine.Mask{Shape: "image"}}}
	workflowCall[app.Artifact](t, s, "dither_recipe_save", app.RecipeSaveRequest{Output: "recipes/mask.json", Recipe: maskedRecipe})
	workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: "masked.png", Recipe: "recipes/mask.json", MaskInput: "mask.png"})
	out := workflowDecode(t, workflowRead(t, root, "masked.png"))
	original := workflowDecode(t, workflowRead(t, root, "input.png"))
	for y := 0; y < 18; y++ {
		for x := 1; x < 24; x++ {
			c := color.NRGBAModel.Convert(out.At(x, y)).(color.NRGBA)
			if x < 12 {
				if c != color.NRGBAModel.Convert(original.At(x, y)).(color.NRGBA) {
					t.Fatal("external mask changed unselected pixels")
				}
			} else if c.R != 0 && c.R != 255 {
				t.Fatal("external mask did not quantize selected pixels")
			}
		}
	}
}

func TestWorkflowCompareCellsAndBatchPartialFailure(t *testing.T) {
	s, root, source := workflowFixture(t)
	cfg := engine.Config{Width: 12, Seed: 42}
	ids := []string{"atkinson", "bayer-8", "blue-noise"}
	result := workflowCall[app.CompareResult](t, s, "dither_compare", app.CompareRequest{RenderRequest: app.RenderRequest{Input: "input.png", Output: "compare.png", Colors: []string{"#000000", "#ffffff"}, Options: cfg}, Algorithms: ids, Columns: 2})
	workflowArtifact(t, root, result.Artifact)
	if result.Artifact.Width != 24 || result.Artifact.Height != 18 || len(result.Cells) != 3 {
		t.Fatalf("comparison metadata %#v", result)
	}
	sheet := workflowDecode(t, workflowRead(t, root, "compare.png"))
	cfg.Palette, _ = engine.ResolvePalette("mono", nil)
	for i, cell := range result.Cells {
		if cell.Algorithm != ids[i] || cell.X != i%2*12 || cell.Y != i/2*9 || cell.Width != 12 || cell.Height != 9 {
			t.Fatalf("cell metadata %#v", cell)
		}
		cfg.Algorithm = ids[i]
		expected, err := engine.Process(context.Background(), source, cfg)
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 9; y++ {
			for x := 0; x < 12; x++ {
				c := color.NRGBAModel.Convert(sheet.At(cell.X+x, cell.Y+y)).(color.NRGBA)
				if c != expected.NRGBAAt(x, y) {
					t.Fatalf("comparison cell %s differs at %d,%d", cell.Algorithm, x, y)
				}
			}
		}
	}
	// Fail an interior item and check both successful neighbors are published.
	batch := workflowCall[app.BatchResult](t, s, "dither_batch", app.BatchRequest{Items: []app.RenderRequest{{Input: "input.png", Output: "batch-first.png"}, {Input: "missing.png", Output: "batch-missing.png"}, {Input: "input.png", Output: "batch-last.png", Palette: "cga"}}})
	if len(batch.Items) != 3 || batch.Items[0].Artifact == nil || batch.Items[1].Error == "" || batch.Items[1].Artifact != nil || batch.Items[2].Artifact == nil {
		t.Fatalf("batch results %#v", batch)
	}
	for _, i := range []int{0, 2} {
		if batch.Items[i].Index != i {
			t.Fatal("batch order changed")
		}
		workflowArtifact(t, root, *batch.Items[i].Artifact)
	}
	if _, err := os.Stat(filepath.Join(root, "batch-missing.png")); !os.IsNotExist(err) {
		t.Fatal("failed batch item published output")
	}
}

func workflowPNGPhysicalResolution(t *testing.T, b []byte, dpi int) {
	t.Helper()
	found := false
	for offset := 8; offset+12 <= len(b); {
		length := int(binary.BigEndian.Uint32(b[offset : offset+4]))
		if length > len(b)-offset-12 {
			t.Fatal("truncated PNG chunk")
		}
		kind := string(b[offset+4 : offset+8])
		data := b[offset+8 : offset+8+length]
		crc := binary.BigEndian.Uint32(b[offset+8+length : offset+12+length])
		if crc32.ChecksumIEEE(b[offset+4:offset+8+length]) != crc {
			t.Fatal("PNG chunk CRC mismatch")
		}
		if kind == "pHYs" {
			if found {
				t.Fatal("duplicate density chunk")
			}
			found = true
			expected := uint32(math.Round(float64(dpi) / .0254))
			if len(data) != 9 || binary.BigEndian.Uint32(data[:4]) != expected || binary.BigEndian.Uint32(data[4:8]) != expected || data[8] != 1 {
				t.Fatal("incorrect physical density")
			}
		}
		offset += length + 12
	}
	if !found {
		t.Fatal("PNG omitted physical density")
	}
}
func TestWorkflowPrintDensityAndSeparations(t *testing.T) {
	s, root, source := workflowFixture(t)
	cfg := engine.Config{Algorithm: "floyd-steinberg", Width: 20}
	palette, _ := engine.ResolvePalette("gameboy", nil)
	expectedCfg := cfg
	expectedCfg.Palette = palette
	expected, err := engine.Process(context.Background(), source, expectedCfg)
	if err != nil {
		t.Fatal(err)
	}
	rendered := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: "print.png", Palette: "gameboy", Options: cfg, DPI: 300})
	workflowPNGPhysicalResolution(t, workflowRead(t, root, rendered.Path), 300)
	archive := workflowCall[app.Artifact](t, s, "dither_separate", app.RenderRequest{Input: "input.png", Output: "inks.zip", Palette: "gameboy", Options: cfg, DPI: 300})
	workflowArtifact(t, root, archive)
	data := workflowRead(t, root, archive.Path)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string][]byte{}
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents[file.Name] = b
	}
	var manifest struct {
		Version int    `json:"version"`
		Kind    string `json:"kind"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
		DPI     int    `json:"dpi"`
		Inks    []struct {
			File   string `json:"file"`
			Color  string `json:"color"`
			Pixels int    `json:"pixels"`
		} `json:"inks"`
	}
	if err := json.Unmarshal(contents["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != "spot-color-separations" || manifest.Width != 20 || manifest.Height != 15 || manifest.DPI != 300 || len(manifest.Inks) != 4 || len(contents) != 5 {
		t.Fatalf("manifest %#v", manifest)
	}
	blackPerPixel := make([]int, 20*15)
	for i, ink := range manifest.Inks {
		if ink.Color != engine.Hex(palette[i]) {
			t.Fatal("ink order changed")
		}
		pngBytes, ok := contents[ink.File]
		if !ok {
			t.Fatal("manifest references missing ink")
		}
		workflowPNGPhysicalResolution(t, pngBytes, 300)
		mask := workflowDecode(t, pngBytes)
		black := 0
		for y := 0; y < 15; y++ {
			for x := 0; x < 20; x++ {
				c := color.NRGBAModel.Convert(mask.At(x, y)).(color.NRGBA)
				if c.A != 255 || c.R != c.G || c.G != c.B || (c.R != 0 && c.R != 255) {
					t.Fatal("ink mask is not opaque binary")
				}
				p := expected.NRGBAAt(x, y)
				matches := p.A > 0 && p.R == palette[i].R && p.G == palette[i].G && p.B == palette[i].B
				if (c.R == 0) != matches {
					t.Fatal("ink coverage does not match palette membership")
				}
				if c.R == 0 {
					black++
					blackPerPixel[y*20+x]++
				}
			}
		}
		if black != ink.Pixels {
			t.Fatal("manifest pixel count differs from mask")
		}
	}
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			want := 1
			if expected.NRGBAAt(x, y).A == 0 {
				want = 0
			}
			if blackPerPixel[y*20+x] != want {
				t.Fatal("ink masks overlap or omit a visible pixel")
			}
		}
	}
}

func TestWorkflowAnimationDeterminismAndSpriteFrames(t *testing.T) {
	s, root, _ := workflowFixture(t)
	q := app.AnimateRequest{RenderRequest: app.RenderRequest{Input: "input.png", Output: "motion-a.gif", Palette: "cga", Options: engine.Config{Algorithm: "bayer-4", Width: 16, Seed: 7}}, Effect: "noise", Frames: 6, FPS: 20}
	a := workflowCall[app.Artifact](t, s, "dither_animate", q)
	q.Output = "motion-b.gif"
	b := workflowCall[app.Artifact](t, s, "dither_animate", q)
	if a.SHA256 != b.SHA256 {
		t.Fatal("same animation recipe produced different bytes")
	}
	workflowArtifact(t, root, a)
	decoded, err := gif.DecodeAll(bytes.NewReader(workflowRead(t, root, a.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Image) != 6 || a.Frames != 6 || a.Width != 16 || a.Height != 12 {
		t.Fatal("animation metadata differs")
	}
	distinct := false
	for i, frame := range decoded.Image {
		if frame.Bounds() != image.Rect(0, 0, 16, 12) || decoded.Delay[i] != 5 {
			t.Fatal("frame bounds/timing changed")
		}
		if i > 0 && !bytes.Equal(frame.Pix, decoded.Image[0].Pix) {
			distinct = true
		}
	}
	if !distinct {
		t.Fatal("noise animation produced identical frames")
	}
	q.Output = "motion-sheet.png"
	q.Format = "png"
	q.Columns = 3
	sprite := workflowCall[app.Artifact](t, s, "dither_animate", q)
	if sprite.Width != 48 || sprite.Height != 24 || sprite.Frames != 6 {
		t.Fatalf("sprite metadata %#v", sprite)
	}
	sheet := workflowDecode(t, workflowRead(t, root, sprite.Path))
	for i, frame := range decoded.Image {
		for y := 0; y < 12; y++ {
			for x := 0; x < 16; x++ {
				got := color.NRGBAModel.Convert(sheet.At(i%3*16+x, i/3*12+y)).(color.NRGBA)
				want := color.NRGBAModel.Convert(frame.At(x, y)).(color.NRGBA)
				if got.A < 128 {
					got = color.NRGBA{}
				} else {
					got.A = 255
				}
				if got != want {
					t.Fatalf("sprite and GIF frame %d differ at %d,%d", i, x, y)
				}
			}
		}
	}
}

func TestWorkflowSourceGIFFramesTimingAndDisposal(t *testing.T) {
	s, root, _ := workflowFixture(t)
	palette := color.Palette{color.NRGBA{}, color.NRGBA{255, 0, 0, 255}, color.NRGBA{0, 0, 255, 255}, color.White}
	first := image.NewPaletted(image.Rect(0, 0, 8, 6), palette)
	for i := range first.Pix {
		first.Pix[i] = 1
	}
	second := image.NewPaletted(image.Rect(2, 1, 5, 4), palette)
	for i := range second.Pix {
		second.Pix[i] = 2
	}
	third := image.NewPaletted(image.Rect(6, 4, 8, 6), palette)
	for i := range third.Pix {
		third.Pix[i] = 3
	}
	g := &gif.GIF{Image: []*image.Paletted{first, second, third}, Delay: []int{3, 7, 11}, Disposal: []byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalNone}, LoopCount: 2, Config: image.Config{ColorModel: palette, Width: 8, Height: 6}}
	var encoded bytes.Buffer
	if err := gif.EncodeAll(&encoded, g); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.gif"), encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	a := workflowCall[app.Artifact](t, s, "dither_animate", app.AnimateRequest{RenderRequest: app.RenderRequest{Input: "source.gif", Output: "source-dithered.gif", Colors: []string{"#ff0000", "#0000ff", "#ffffff"}, Options: engine.Config{Algorithm: "threshold", Width: 8}}, Effect: "source"})
	result, err := gif.DecodeAll(bytes.NewReader(workflowRead(t, root, a.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Image) != 3 || result.LoopCount != 2 || !reflect.DeepEqual(result.Delay, []int{3, 7, 11}) {
		t.Fatal("source GIF timing/loop changed")
	}
	for i, frame := range result.Image {
		for y := 0; y < 6; y++ {
			for x := 0; x < 8; x++ {
				want := color.NRGBA{255, 0, 0, 255}
				if i == 1 && x >= 2 && x < 5 && y >= 1 && y < 4 {
					want = color.NRGBA{0, 0, 255, 255}
				}
				if i == 2 && x >= 6 && y >= 4 {
					want = color.NRGBA{255, 255, 255, 255}
				}
				got := color.NRGBAModel.Convert(frame.At(x, y)).(color.NRGBA)
				if got != want {
					t.Fatalf("composited frame %d at %d,%d %#v want %#v", i, x, y, got, want)
				}
			}
		}
	}
	inspection := workflowCall[app.Inspection](t, s, "dither_inspect", app.InputRequest{Input: a.Path})
	if inspection.Frames != 3 || inspection.Width != 8 || inspection.Height != 6 {
		t.Fatalf("source GIF inspection %#v", inspection)
	}
}

func TestWorkflowPaletteExtractionConsumableColors(t *testing.T) {
	s, root, _ := workflowFixture(t)
	extracted := workflowCall[struct {
		Colors []string `json:"colors"`
	}](t, s, "dither_palette_extract", app.ExtractRequest{Input: "input.png", Count: 6})
	if len(extracted.Colors) != 6 {
		t.Fatal("unexpected extracted palette size")
	}
	for _, s := range extracted.Colors {
		if _, err := engine.ParseHex(s); err != nil {
			t.Fatal(err)
		}
	}
	a := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: "input.png", Output: "extracted.png", Colors: extracted.Colors, Options: engine.Config{Algorithm: "sierra-lite"}})
	workflowArtifact(t, root, a)
	im := workflowDecode(t, workflowRead(t, root, a.Path))
	allowed := map[string]bool{}
	for _, s := range extracted.Colors {
		allowed[s] = true
	}
	for y := 0; y < 18; y++ {
		for x := 0; x < 24; x++ {
			c := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
			if c.A > 0 && !allowed[fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)] {
				t.Fatal("render did not use extracted colors")
			}
		}
	}
}
