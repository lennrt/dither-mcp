package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func previewRender(r StudioResult, output string) RenderRequest {
	return RenderRequest{Input: r.Path, Output: output, Palette: r.Recipe.Palette, Colors: r.Recipe.Colors, Options: r.Recipe.Options, MaskInput: r.MaskInput, ExpectedSourceSHA256: r.SourceSHA256, ExpectedMaskSHA256: r.MaskSHA256}
}

func assertChanged(t *testing.T, err error, code, path, expected, actual string) {
	t.Helper()
	var changed *InputChangedError
	if !errors.As(err, &changed) || changed.Code != code || changed.Path != path || changed.ExpectedSHA256 != expected || changed.ActualSHA256 != actual {
		t.Fatalf("changed input error: %v", err)
	}
}

func TestGuardedRenderDetectsSourceAndMaskReplacement(t *testing.T) {
	for _, kind := range []string{"source", "mask"} {
		t.Run(kind, func(t *testing.T) {
			s, root := studioFixture(t, 32, 24)
			original, err := os.ReadFile(filepath.Join(root, "source.png"))
			if err != nil {
				t.Fatal(err)
			}
			// Equal-length ancillary metadata changes preserve every pixel, the
			// dimensions, and the file size. Only the byte fingerprint changes.
			before := append(append([]byte(nil), original[:33]...), normPNGChunk("tEXt", []byte("note\x00a"))...)
			before = append(before, original[33:]...)
			after := append(append([]byte(nil), original[:33]...), normPNGChunk("tEXt", []byte("note\x00b"))...)
			after = append(after, original[33:]...)
			path := kind + ".png"
			normWrite(t, root, path, before)
			q := StudioRequest{Input: "source.png", Palette: "gameboy", Options: engine.Config{Width: 16}}
			if kind == "mask" {
				q.MaskInput = path
			}
			r := studioCall(t, s, q)
			stamp, _ := os.Stat(filepath.Join(root, path))
			normWrite(t, root, path, after)
			if err := os.Chtimes(filepath.Join(root, path), stamp.ModTime(), stamp.ModTime()); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(previewRender(r, "exports/blocked.png"))
			_, err = s.Do(context.Background(), "dither_render", b)
			assertChanged(t, err, kind+"_changed", path, digest(before), digest(after))
			batch := normCall(t, s, "dither_batch", BatchRequest{Items: []RenderRequest{previewRender(r, "exports/blocked.png")}}).(BatchResult)
			if len(batch.Items) != 1 || batch.Items[0].Artifact != nil || !strings.HasPrefix(batch.Items[0].Error, kind+"_changed:") {
				t.Fatal("batch discarded the stable changed-input code")
			}
			if _, err := os.Stat(filepath.Join(root, "exports")); !os.IsNotExist(err) {
				t.Fatal("changed input created an output directory or artifact")
			}
			// A refreshed preview accepts the replacement, including its new hash.
			fresh := studioCall(t, s, q)
			saved := normCall(t, s, "dither_render", previewRender(fresh, "saved.png")).(Artifact)
			preview, _ := base64.StdEncoding.DecodeString(fresh.Data)
			if saved.SHA256 != digest(preview) {
				t.Fatal("refreshed guarded export changed preview pixels")
			}
		})
	}
}

func TestGuardChecksBytesBeforeDecodingAndPreservesNormalRenders(t *testing.T) {
	s, root := studioFixture(t, 24, 16)
	r := studioCall(t, s, StudioRequest{Input: "source.png"})
	normWrite(t, root, "source.png", []byte("replaced with an invalid image"))
	b, _ := json.Marshal(previewRender(r, "blocked.png"))
	_, err := s.Do(context.Background(), "dither_render", b)
	assertChanged(t, err, "source_changed", "source.png", r.SourceSHA256, digest([]byte("replaced with an invalid image")))
	if _, err := os.Stat(filepath.Join(root, "blocked.png")); !os.IsNotExist(err) {
		t.Fatal("fingerprint mismatch published output")
	}
	// Unguarded requests continue processing the current source.
	im := image.NewNRGBA(image.Rect(0, 0, 24, 16))
	im.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	normWrite(t, root, "source.png", normPNG(t, im, 0, nil))
	normCall(t, s, "dither_render", RenderRequest{Input: "source.png", Output: "current.png"})
	q := previewRender(studioCall(t, s, StudioRequest{Input: "source.png"}), "uppercase.png")
	q.ExpectedSourceSHA256 = strings.ToUpper(q.ExpectedSourceSHA256)
	normCall(t, s, "dither_render", q)
	for _, q := range []RenderRequest{
		{Input: "source.png", Output: "bad.png", ExpectedSourceSHA256: "bad"},
		{Input: "source.png", Output: "bad.png", ExpectedSourceSHA256: strings.Repeat("z", 64)},
		{Input: "source.png", Output: "bad.png", ExpectedMaskSHA256: strings.Repeat("0", 64)},
		{Input: "source.png", Output: "bad.png", MaskInput: "source.png", ExpectedMaskSHA256: "bad"},
	} {
		b, _ := json.Marshal(q)
		if _, err := s.Do(context.Background(), "dither_render", b); err == nil {
			t.Fatal("accepted invalid guard arguments")
		}
	}
}

func TestGuardedRenderInvalidatesUnavailableSourceAndMask(t *testing.T) {
	for _, kind := range []string{"source", "mask"} {
		t.Run(kind, func(t *testing.T) {
			s, root := studioFixture(t, 24, 16)
			path := kind + ".png"
			q := StudioRequest{Input: "source.png"}
			if kind == "mask" {
				source, _ := os.ReadFile(filepath.Join(root, "source.png"))
				normWrite(t, root, path, source)
				q.MaskInput = path
			}
			r := studioCall(t, s, q)
			expected := r.SourceSHA256
			if kind == "mask" {
				expected = r.MaskSHA256
			}
			if err := os.Remove(filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(previewRender(r, "exports/blocked.png"))
			_, err := s.Do(context.Background(), "dither_render", b)
			assertChanged(t, err, kind+"_changed", path, expected, "")
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatal("changed-input error lost its underlying read failure")
			}
			if _, err := os.Stat(filepath.Join(root, "exports")); !os.IsNotExist(err) {
				t.Fatal("unavailable input created an output directory or artifact")
			}
		})
	}
}

func TestStudioSourceDimensionsFollowOrientationAndCrop(t *testing.T) {
	s, root := studioFixture(t, 8, 12)
	im := image.NewNRGBA(image.Rect(0, 0, 8, 12))
	source := normPNG(t, im, 6, nil)
	normWrite(t, root, "upright.png", source)
	for _, tt := range []struct {
		name          string
		crop          *engine.Rect
		width, height int
	}{
		{"upright", nil, 12, 8},
		{"crop", &engine.Rect{X: 1, Y: 1, Width: 9, Height: 5}, 9, 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := studioCall(t, s, StudioRequest{Input: "upright.png", Options: engine.Config{Width: 6, Crop: tt.crop, Algorithm: "atkinson", Seed: 7}})
			if r.SourceWidth != tt.width || r.SourceHeight != tt.height || r.SourceSHA256 != digest(source) {
				t.Fatalf("source geometry: %+v", r.StudioMetadata)
			}
			q := previewRender(r, "export-"+tt.name+".png")
			q.Options.Width, q.Options.Height = r.SourceWidth, r.SourceHeight
			a := normCall(t, s, "dither_render", q).(Artifact)
			if a.Width != tt.width || a.Height != tt.height {
				t.Fatalf("source-size export: %+v", a)
			}
			cfg := a.Recipe.Options
			cfg.Width, cfg.Height = r.Recipe.Options.Width, r.Recipe.Options.Height
			if !reflect.DeepEqual(cfg, r.Recipe.Options) {
				t.Fatal("source-size export changed accepted settings beyond dimensions")
			}
		})
	}
}

// replaceOnDecodeContext replaces an input when metadata parsing begins, after
// its bytes have been read and checked. It deterministically exercises the gap
// where a separate fingerprint check and second path read would be unsafe.
type replaceOnDecodeContext struct {
	context.Context
	once    sync.Once
	replace func()
}

func (c *replaceOnDecodeContext) Err() error {
	c.once.Do(c.replace)
	return c.Context.Err()
}

func TestStudioAndGuardedRenderUseTheDecodedSnapshot(t *testing.T) {
	for _, kind := range []string{"source", "mask"} {
		t.Run(kind, func(t *testing.T) {
			s, root := studioFixture(t, 24, 16)
			original, _ := os.ReadFile(filepath.Join(root, "source.png"))
			im := image.NewNRGBA(image.Rect(0, 0, 24, 16))
			replacement := normPNG(t, im, 0, nil)
			path := kind + ".png"
			normWrite(t, root, path, original)
			ctx := func() context.Context {
				return &replaceOnDecodeContext{Context: context.Background(), replace: func() { normWrite(t, root, path, replacement) }}
			}
			q := StudioRequest{Input: "source.png", Palette: "cga", Options: engine.Config{Width: 12, Seed: 11}}
			if kind == "mask" {
				q.MaskInput = path
			}
			r, err := s.studio(ctx(), q)
			if err != nil {
				t.Fatal(err)
			}
			hash := r.SourceSHA256
			if kind == "mask" {
				hash = r.MaskSHA256
			}
			if hash != digest(original) {
				t.Fatal("preview hashed bytes other than the decoded snapshot")
			}
			normWrite(t, root, path, original)
			a, err := s.render(ctx(), previewRender(r, "snapshot.png"))
			if err != nil {
				t.Fatal(err)
			}
			preview, _ := base64.StdEncoding.DecodeString(r.Data)
			if a.SHA256 != digest(preview) {
				t.Fatal("guarded export reread input after validating its fingerprint")
			}
			after, _ := os.ReadFile(filepath.Join(root, path))
			if !bytes.Equal(after, replacement) {
				t.Fatal("test did not replace the input during decoding")
			}
		})
	}
}

func TestEmbeddedRenderRequestsApplySourceGuard(t *testing.T) {
	s, root := studioFixture(t, 24, 16)
	r := studioCall(t, s, StudioRequest{Input: "source.png"})
	normWrite(t, root, "source.png", []byte("changed"))
	q := previewRender(r, "blocked.png")
	separations := q
	separations.Output = "blocked.zip"
	for _, tt := range []struct {
		name string
		q    any
	}{
		{"dither_compare", CompareRequest{RenderRequest: q, Algorithms: []string{"atkinson"}}},
		{"dither_animate", AnimateRequest{RenderRequest: q, Frames: 1}},
		{"dither_separate", separations},
	} {
		b, _ := json.Marshal(tt.q)
		_, err := s.Do(context.Background(), tt.name, b)
		assertChanged(t, err, "source_changed", "source.png", r.SourceSHA256, digest([]byte("changed")))
	}
	batch := normCall(t, s, "dither_batch", BatchRequest{Items: []RenderRequest{q}}).(BatchResult)
	if len(batch.Items) != 1 || batch.Items[0].Artifact != nil || !strings.HasPrefix(batch.Items[0].Error, "source_changed:") {
		t.Fatalf("batch ignored fingerprint guard: %+v", batch)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatal("mismatched inherited guards wrote files")
	}
}
