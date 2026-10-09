package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func studioFixture(t *testing.T, w, h int) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			im.SetNRGBA(x, y, color.NRGBA{uint8(x * 7), uint8(y * 11), 137, uint8(128 + (x+y)%128)})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.png"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, root
}

func studioCall(t *testing.T, s *Service, q StudioRequest) StudioResult {
	t.Helper()
	b, _ := json.Marshal(q)
	v, err := s.Do(context.Background(), "dither_studio", b)
	if err != nil {
		t.Fatal(err)
	}
	return v.(StudioResult)
}

func TestStudioPreviewReplaysWithoutWriting(t *testing.T) {
	s, root := studioFixture(t, 64, 48)
	before, _ := os.ReadFile(filepath.Join(root, "source.png"))
	zero := 0.0
	q := StudioRequest{Input: "source.png", Colors: []string{"#152a39", "#efddc2"}, Options: engine.Config{Algorithm: "atkinson", Width: 36, PixelScale: 2, Contrast: &zero, Seed: 23, Crop: &engine.Rect{X: 4, Y: 2, Width: 48, Height: 32}, Effects: engine.Effects{Scanlines: .1}}}
	r := studioCall(t, s, q)
	if r.Width != 36 || r.Height != 24 || r.Recipe.Options.Contrast == nil || *r.Recipe.Options.Contrast != 0 || r.MIMEType != "image/png" {
		t.Fatalf("unexpected preview: %+v", r.StudioMetadata)
	}
	files, _ := os.ReadDir(root)
	after, _ := os.ReadFile(filepath.Join(root, "source.png"))
	if len(files) != 1 || !bytes.Equal(before, after) {
		t.Fatal("preview changed workspace files")
	}
	canonical := r.Request()
	if canonical.Options.Width != r.Width || canonical.Options.Height != r.Height || !reflect.DeepEqual(canonical.Colors, q.Colors) {
		t.Fatal("canonical request does not preserve the preview recipe")
	}
	if again := studioCall(t, s, canonical); again.Data != r.Data {
		t.Fatal("canonical preview is not deterministic")
	}
	render := RenderRequest{Input: canonical.Input, Output: "saved.png", Colors: canonical.Colors, Palette: canonical.Palette, Options: canonical.Options, MaskInput: canonical.MaskInput}
	b, _ := json.Marshal(render)
	if _, err := s.Do(context.Background(), "dither_render", b); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(filepath.Join(root, "saved.png"))
	preview, _ := base64.StdEncoding.DecodeString(r.Data)
	if !bytes.Equal(saved, preview) {
		t.Fatal("saving the same recipe changed preview PNG bytes")
	}
	if _, err := s.Do(context.Background(), "dither_render", b); err == nil {
		t.Fatal("save overwrote an existing output")
	}
}

func TestStudioGeometryAndAdmission(t *testing.T) {
	s, root := studioFixture(t, 640, 960)
	for _, tt := range []struct {
		name          string
		options       engine.Config
		width, height int
	}{
		{"default", engine.Config{}, 341, 512},
		{"width", engine.Config{Width: 100}, 100, 150},
		{"height", engine.Config{Height: 99}, 66, 99},
		{"both", engine.Config{Width: 123, Height: 87}, 123, 87},
		{"crop", engine.Config{Crop: &engine.Rect{X: 1, Y: 1, Width: 20, Height: 10}}, 20, 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := studioCall(t, s, StudioRequest{Input: "source.png", Options: tt.options})
			if r.Width != tt.width || r.Height != tt.height || r.Recipe.Palette != "mono" || r.Recipe.Options.Algorithm != "floyd-steinberg" {
				t.Fatalf("preview=%+v", r.StudioMetadata)
			}
		})
	}
	for _, raw := range []string{
		`{"input":"source.png","options":{"width":1025}}`,
		`{"input":"source.png","options":{"width":1024}}`,
		`{"input":"source.png","options":{"height":-1}}`,
		`{"input":"source.png","options":{"seed":9007199254740992}}`,
		`{"input":"source.png","options":{"seed":-9007199254740992}}`,
		`{"input":"source.png","options":{"gamma":0}}`,
		`{"input":"source.png","options":{"crop":{"x":600,"y":0,"width":100,"height":20}}}`,
		`{"input":"source.png","output":"should-not-exist.png"}`,
		`{"input":"source.png","recipe":"saved.json"}`,
		`{"input":"../outside.png"}`,
		`{"input":"https://example.com/source.png"}`,
		`{"input":"source.png","palette":"gameboy","colors":["#000","#fff"]}`,
	} {
		if _, err := s.Do(context.Background(), "dither_studio", []byte(raw)); err == nil {
			t.Fatalf("accepted invalid studio request: %s", raw)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Do(ctx, "dither_studio", []byte(`{"input":"source.png"}`)); err == nil {
		t.Fatal("accepted canceled preview")
	}
	files, _ := os.ReadDir(root)
	if len(files) != 1 {
		t.Fatal("studio validation or cancellation wrote files")
	}
}

func TestStudioImageMaskReplay(t *testing.T) {
	s, root := studioFixture(t, 32, 24)
	q := StudioRequest{Input: "source.png", MaskInput: "source.png", Palette: "gameboy", Options: engine.Config{Mask: &engine.Mask{Shape: "image", Invert: true}}}
	r := studioCall(t, s, q)
	if r.MaskInput != q.MaskInput || !r.Recipe.Options.Mask.Invert {
		t.Fatal("preview discarded image mask settings")
	}
	b, _ := json.Marshal(RenderRequest{Input: q.Input, Output: "mask.png", Palette: r.Recipe.Palette, Options: r.Recipe.Options, MaskInput: r.MaskInput})
	if _, err := s.Do(context.Background(), "dither_render", b); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(filepath.Join(root, "mask.png"))
	preview, _ := base64.StdEncoding.DecodeString(r.Data)
	if !bytes.Equal(saved, preview) {
		t.Fatal("mask preview did not replay")
	}
}

func TestStudioNormalization(t *testing.T) {
	s, root := studioFixture(t, 3, 2)
	im := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y := range 2 {
		for x := range 3 {
			im.SetNRGBA(x, y, color.NRGBA{128, 128, 128, 170})
		}
	}
	normWrite(t, root, "normalized.png", normPNG(t, im, 6, normProfile(t)))
	r := studioCall(t, s, StudioRequest{Input: "normalized.png", Colors: []string{"#808080", "#bcbcbc"}, Options: engine.Config{Algorithm: "threshold"}})
	if r.Width != 2 || r.Height != 3 {
		t.Fatal("studio did not apply EXIF orientation before deriving dimensions")
	}
	b, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	c := color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA)
	if c.R != 188 || c.A != 170 {
		t.Fatalf("studio did not normalize input color and preserve alpha: %+v", c)
	}
}

func TestStudioEncodedBudgetAndSafeSeeds(t *testing.T) {
	s, root := studioFixture(t, 2, 2)
	for _, seed := range []int64{-(1<<53 - 1), 1<<53 - 1} {
		r := studioCall(t, s, StudioRequest{Input: "source.png", Options: engine.Config{Seed: seed}})
		if r.Recipe.Options.Seed != seed {
			t.Fatal("studio changed a boundary seed")
		}
	}
	im := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	state := uint32(0x12345678)
	next := func() uint8 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return uint8(state)
	}
	for i := 0; i < len(im.Pix); i += 4 {
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = next(), next(), next(), 255
	}
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, im); err != nil {
		t.Fatal(err)
	}
	if pngBytes.Len() <= StudioMaxPNGBytes {
		t.Fatal("budget fixture must exceed the preview limit")
	}
	normWrite(t, root, "source.png", pngBytes.Bytes())
	q := StudioRequest{Input: "source.png", Options: engine.Config{Width: 1024, Height: 1024, Mask: &engine.Mask{Shape: "rectangle", X: -1, Y: -1, Width: 1, Height: 1}}}
	b, _ := json.Marshal(q)
	if _, err := s.Do(context.Background(), "dither_studio", b); err == nil || !strings.Contains(err.Error(), "studio PNG preview: encoded output exceeds byte limit") {
		t.Fatalf("expected PNG budget error, got %v", err)
	}
	files, _ := os.ReadDir(root)
	if len(files) != 1 {
		t.Fatal("oversized preview wrote files")
	}
}
