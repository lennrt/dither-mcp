package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/app"
)

func TestPaletteComparisonMatchesStandaloneRenders(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		t.Run(fmt.Sprintf("advanced=%v", advanced), func(t *testing.T) {
			s, root, _ := workflowFixture(t)
			q := app.CompareRequest{
				RenderRequest: app.RenderRequest{Input: "input.png", Output: "comparison.png"},
				Palettes:      []string{"cga", "oat-and-ink", "gameboy"},
				Columns:       2,
			}
			if advanced {
				zero := 0.0
				q.Options = engine.Config{Algorithm: "random", Width: 13, Height: 9, PixelScale: 2, Seed: 27, ColorSpace: "linear-rgb", Saturation: &zero, Crop: &engine.Rect{X: 1, Y: 2, Width: 20, Height: 14}, Effects: engine.Effects{Scanlines: .2}}
				q.MaskInput = "mask.png"
				mask := image.NewGray(image.Rect(0, 0, 2, 1))
				mask.SetGray(1, 0, color.Gray{Y: 255})
				workflowWritePNG(t, root, q.MaskInput, mask)
			}
			result := workflowCall[app.CompareResult](t, s, "dither_compare", q)
			encoded := workflowRead(t, root, q.Output)
			if result.Artifact.SHA256 != fmt.Sprintf("%x", sha256.Sum256(encoded)) || result.Artifact.Bytes != len(encoded) {
				t.Fatal("sheet digest or byte count does not match published bytes")
			}
			if result.Artifact.Recipe != nil || result.Artifact.Operation != "dither_compare" {
				t.Fatalf("palette sheet must use operation provenance, not a single recipe: %+v", result.Artifact)
			}
			wire, _ := json.Marshal(q)
			var parameters map[string]any
			if err := json.Unmarshal(wire, &parameters); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Artifact.Parameters, parameters) {
				t.Fatalf("operation parameters changed: got %#v, want %#v", result.Artifact.Parameters, parameters)
			}
			if len(result.Cells) != len(q.Palettes) {
				t.Fatalf("wrong cell count: %+v", result.Cells)
			}
			sheet := workflowDecode(t, workflowRead(t, root, q.Output))
			for i, cell := range result.Cells {
				standalone := q.RenderRequest
				standalone.Output = fmt.Sprintf("cell-%d.png", i)
				standalone.Palette = q.Palettes[i]
				if standalone.Options.Width == 0 && standalone.Options.Height == 0 {
					standalone.Options.Width = 320
				}
				artifact := workflowCall[app.Artifact](t, s, "dither_render", standalone)
				expected := workflowDecode(t, workflowRead(t, root, artifact.Path))
				if cell.Palette != q.Palettes[i] || cell.Algorithm != artifact.Recipe.Options.Algorithm || cell.X != i%2*artifact.Width || cell.Y != i/2*artifact.Height || cell.Width != artifact.Width || cell.Height != artifact.Height {
					t.Fatalf("incorrect cell metadata: %+v", cell)
				}
				for y := 0; y < cell.Height; y++ {
					for x := 0; x < cell.Width; x++ {
						got := color.NRGBAModel.Convert(sheet.At(cell.X+x, cell.Y+y)).(color.NRGBA)
						want := color.NRGBAModel.Convert(expected.At(x, y)).(color.NRGBA)
						if got != want {
							t.Fatalf("palette %s differs from standalone render at %d,%d: got %v, want %v", cell.Palette, x, y, got, want)
						}
					}
				}
			}
			cell := result.Cells[0]
			if result.Artifact.Width != 2*cell.Width || result.Artifact.Height != 2*cell.Height {
				t.Fatalf("incorrect sheet dimensions: %+v", result.Artifact)
			}
			for y := cell.Height; y < 2*cell.Height; y++ {
				for x := cell.Width; x < 2*cell.Width; x++ {
					if _, _, _, alpha := sheet.At(x, y).RGBA(); alpha != 0 {
						t.Fatal("unused trailing cell is not transparent")
					}
				}
			}
			q.Output = "comparison-again.png"
			again := workflowCall[app.CompareResult](t, s, "dither_compare", q)
			if again.Artifact.SHA256 != result.Artifact.SHA256 || !reflect.DeepEqual(again.Cells, result.Cells) {
				t.Fatal("same palette comparison did not produce identical pixels and cell metadata")
			}
		})
	}
}

func TestPaletteComparisonRejectsAmbiguityAndLimitsWithoutPublishing(t *testing.T) {
	tests := []struct {
		name   string
		change func(*app.CompareRequest)
		want   string
	}{
		{"both modes", func(q *app.CompareRequest) { q.Algorithms = []string{"atkinson"} }, "not both"},
		{"no candidates", func(q *app.CompareRequest) { q.Palettes = nil }, "1..12"},
		{"too many", func(q *app.CompareRequest) { q.Palettes = make([]string, 13) }, "1..12"},
		{"empty ID", func(q *app.CompareRequest) { q.Palettes = []string{""} }, "must not be empty"},
		{"unknown ID", func(q *app.CompareRequest) { q.Palettes = []string{"gameboy", "not-a-palette"} }, "unknown palette"},
		{"common palette", func(q *app.CompareRequest) { q.Palette = "gameboy" }, "cannot be combined"},
		{"common colors", func(q *app.CompareRequest) { q.Colors = []string{"#000", "#fff"} }, "cannot be combined"},
		{"saved recipe", func(q *app.CompareRequest) { q.Recipe = "look.json" }, "cannot be combined"},
		{"columns", func(q *app.CompareRequest) { q.Columns = 13 }, "columns"},
		{"algorithm", func(q *app.CompareRequest) { q.Options.Algorithm = "not-a-method" }, "unknown algorithm"},
		{"sheet width", func(q *app.CompareRequest) { q.Options.Width = 8193; q.Options.Height = 1; q.Columns = 2 }, "sheet exceeds"},
		{"sheet pixels", func(q *app.CompareRequest) {
			q.Palettes = []string{"mono", "mono", "mono", "mono", "mono", "mono", "mono", "mono", "mono", "mono", "mono", "mono"}
			q.Options = engine.Config{Width: 1400, Height: 1000, PixelScale: 256}
			q.Columns = 4
		}, "sheet exceeds"},
		{"invalid path", func(q *app.CompareRequest) { q.Output = "../escaped.png" }, "traversal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, root, _ := workflowFixture(t)
			q := app.CompareRequest{RenderRequest: app.RenderRequest{Input: "input.png", Output: "exports/blocked.png", Options: engine.Config{Width: 4}}, Palettes: []string{"gameboy"}}
			tt.change(&q)
			raw, _ := json.Marshal(q)
			if _, err := s.Do(context.Background(), "dither_compare", raw); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != "input.png" {
				t.Fatalf("invalid comparison created files: %v, %v", entries, err)
			}
		})
	}
}

func TestPaletteComparisonPreservesPublicationAndInputGuards(t *testing.T) {
	s, root, _ := workflowFixture(t)
	q := app.CompareRequest{RenderRequest: app.RenderRequest{Input: "input.png", Output: "comparison.png", Options: engine.Config{Width: 4}}, Palettes: []string{"gameboy"}}
	for i := 1; i < 12; i++ {
		q.Palettes = append(q.Palettes, "gameboy")
	}
	result := workflowCall[app.CompareResult](t, s, "dither_compare", q)
	if len(result.Cells) != 12 || result.Artifact.Width != 12 || result.Artifact.Height != 12 {
		t.Fatalf("12 candidates or default three columns failed: %+v", result)
	}
	original := workflowRead(t, root, q.Output)
	raw, _ := json.Marshal(q)
	if _, err := s.Do(context.Background(), "dither_compare", raw); err == nil {
		t.Fatal("comparison replaced existing output")
	}
	if !bytes.Equal(original, workflowRead(t, root, q.Output)) {
		t.Fatal("collision changed existing bytes")
	}
	q.Output = "canceled.png"
	raw, _ = json.Marshal(q)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Do(ctx, "dither_compare", raw); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, q.Output)); !os.IsNotExist(err) {
		t.Fatal("canceled comparison created output")
	}
	for _, kind := range []string{"source", "mask"} {
		t.Run(kind, func(t *testing.T) {
			s, root, source := workflowFixture(t)
			workflowWritePNG(t, root, "mask.png", source)
			preview := workflowCall[app.StudioResult](t, s, "dither_studio", app.StudioRequest{Input: "input.png", MaskInput: "mask.png", Options: engine.Config{Width: 4}})
			q := app.CompareRequest{RenderRequest: app.RenderRequest{Input: "input.png", MaskInput: "mask.png", Output: "exports/guarded.png", Options: engine.Config{Width: 4}, ExpectedSourceSHA256: preview.SourceSHA256, ExpectedMaskSHA256: preview.MaskSHA256}, Palettes: []string{"mono", "gameboy"}}
			path := "input.png"
			if kind == "mask" {
				path = "mask.png"
			}
			if err := os.WriteFile(filepath.Join(root, path), []byte("changed bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(q)
			_, err := s.Do(context.Background(), "dither_compare", raw)
			var changed *app.InputChangedError
			if !errors.As(err, &changed) || changed.Code != kind+"_changed" {
				t.Fatalf("input guard lost: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "exports")); !os.IsNotExist(err) {
				t.Fatal("changed input created output directory")
			}
		})
	}
}
