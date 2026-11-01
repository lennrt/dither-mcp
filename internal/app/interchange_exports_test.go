package app

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func TestInterchangeRenderFormats(t *testing.T) {
	s, root := securityFixture(t)
	catalog := s.Catalog()
	if !slices.Equal(catalog.OutputFormats, engine.Formats()) || !slices.Equal(catalog.Media.Still.OutputFormats, engine.Formats()) {
		t.Fatal("catalog encoder discovery differs from engine formats")
	}
	for _, tt := range []struct{ path, explicit, want string }{
		{"out.bmp", "", "bmp"}, {"out.tiff", "", "tiff"}, {"out.tif", "", "tiff"}, {"override.dat", "TIF", "tiff"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			artifact, err := s.render(context.Background(), RenderRequest{Input: "input.png", Output: tt.path, Format: tt.explicit, Options: engine.Config{ResizeFilter: "area", Width: 3, Height: 2}})
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Format != tt.want || artifact.Width != 3 || artifact.Height != 2 || artifact.Recipe.Options.ResizeFilter != "area" {
				t.Fatalf("artifact=%+v", artifact)
			}
			f, err := os.Open(filepath.Join(root, tt.path))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			decoded, format, err := image.Decode(f)
			if err != nil || format != tt.want || decoded.Bounds() != image.Rect(0, 0, 3, 2) {
				t.Fatalf("decoded=%v format=%q err=%v", decoded, format, err)
			}
			if _, err := s.render(context.Background(), RenderRequest{Input: "input.png", Output: tt.path}); err == nil {
				t.Fatal("existing output overwritten")
			}
		})
	}
}

func TestInterchangeEncodingBoundsAndDPI(t *testing.T) {
	for _, format := range []string{"bmp", "tiff"} {
		t.Run(format, func(t *testing.T) {
			img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
			b := &boundedBuffer{max: 32, ctx: context.Background()}
			if err := engine.EncodeContext(context.Background(), format, b, img); err == nil || b.Len() > 32 {
				t.Fatalf("output budget not enforced: bytes=%d err=%v", b.Len(), err)
			}
			if data, err := encodeBounded(context.Background(), format, img, 300); err == nil || data != nil {
				t.Fatal("non-PNG DPI admitted")
			}
		})
	}
}
