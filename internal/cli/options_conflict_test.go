package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/lennrt/dither-mcp/internal/app"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionsRejectExplicitFlagOverlap(t *testing.T) {
	for _, tc := range []struct{ flag, value, key, option string }{
		{"width", "0", "width", "12"},
		{"algorithm", "atkinson", "Algorithm", `"bayer-4"`},
		{"pixel-scale", "2", "pixel_scale", "3"},
		{"invert", "false", "invert", "true"},
		{"contrast", "0", "contrast", "null"},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			err := Run(context.Background(), []string{"render", "--root", "/nonexistent/options-conflict", "--" + tc.flag + "=" + tc.value, "--options", `{"` + tc.key + `":` + tc.option + `}`}, nil, &out, &diagnostics)
			if err == nil || !strings.Contains(err.Error(), "--"+tc.flag) || !strings.Contains(err.Error(), "--options") {
				t.Fatalf("want early options conflict, got %v", err)
			}
			if out.Len() != 0 {
				t.Fatal("conflict wrote stdout")
			}
		})
	}
}

func TestOptionsComposeWithDistinctExplicitFlags(t *testing.T) {
	root := t.TempDir()
	f, err := os.Create(filepath.Join(root, "source.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 8, 6))); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	var out, diagnostics bytes.Buffer
	err = Run(context.Background(), []string{"render", "--root", root, "--input", "source.png", "--output", "result.png", "--width", "13", "--contrast", "0", "--options", `{"algorithm":"atkinson","invert":true,"effects":{"scanlines":0.1}}`}, nil, &out, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	var artifact app.Artifact
	if err = json.Unmarshal(out.Bytes(), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Width != 13 || artifact.Recipe == nil || artifact.Recipe.Options.Algorithm != "atkinson" || !artifact.Recipe.Options.Invert || artifact.Recipe.Options.Contrast == nil || *artifact.Recipe.Options.Contrast != 0 || artifact.Recipe.Options.Effects.Scanlines != .1 {
		t.Fatalf("distinct options lost: %+v", artifact)
	}
}
