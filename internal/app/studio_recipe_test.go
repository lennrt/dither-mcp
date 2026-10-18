package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func TestStudioSavedRecipeRetainsExternalMask(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "implicit"
		if explicit {
			name = "explicit-inverted"
		}
		t.Run(name, func(t *testing.T) {
			s, root := studioFixture(t, 32, 24)
			mask := image.NewGray(image.Rect(0, 0, 2, 1))
			mask.SetGray(1, 0, color.Gray{Y: 255})
			normWrite(t, root, "mask.png", normPNG(t, mask, 0, nil))
			q := StudioRequest{Input: "source.png", MaskInput: "mask.png", Colors: []string{"#172a35", "#fae8d2"}, Options: engine.Config{Algorithm: "blue-noise", Width: 16, Seed: 17, Saturation: engine.Float(0), Crop: &engine.Rect{X: 2, Y: 2, Width: 24, Height: 18}}}
			if explicit {
				q.Options.Mask = &engine.Mask{Shape: "image", Invert: true}
			}
			preview := studioCall(t, s, q)
			wantMask := &engine.Mask{Shape: "image", Invert: explicit}
			if !reflect.DeepEqual(preview.Recipe.Options.Mask, wantMask) {
				t.Errorf("preview recipe lost external mask intent: got %+v, want %+v", preview.Recipe.Options.Mask, wantMask)
			}
			saved := normCall(t, s, "dither_recipe_save", RecipeSaveRequest{Output: "recipes/accepted.json", Recipe: preview.Recipe}).(Artifact)
			loaded := normCall(t, s, "dither_recipe_load", InputRequest{Input: saved.Path}).(Recipe)
			if !reflect.DeepEqual(loaded, preview.Recipe) {
				t.Fatal("saving changed accepted preview settings")
			}
			data, err := os.ReadFile(filepath.Join(root, saved.Path))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "source.png") || strings.Contains(string(data), "mask.png") || strings.Contains(string(data), "Image") {
				t.Fatal("saved recipe contains a source path or in-memory image")
			}
			request := RenderRequest{Input: "source.png", Output: "missing-mask/result.png", Recipe: saved.Path}
			raw, _ := json.Marshal(request)
			if _, err := s.Do(context.Background(), "dither_render", raw); err == nil {
				t.Error("saved image-mask recipe silently rendered without its external mask")
			}
			if _, err := os.Stat(filepath.Join(root, "missing-mask")); !os.IsNotExist(err) {
				t.Error("missing-mask rejection published an artifact or created its directory")
			}
			request.Output, request.MaskInput = "replayed.png", "mask.png"
			replayed := normCall(t, s, "dither_render", request).(Artifact)
			wantPNG, err := base64.StdEncoding.DecodeString(preview.Data)
			if err != nil {
				t.Fatal(err)
			}
			gotPNG, err := os.ReadFile(filepath.Join(root, replayed.Path))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotPNG, wantPNG) {
				t.Fatal("saved recipe with its external mask changed preview PNG bytes")
			}
			// Ensure this fixture would detect silently omitting the mask rather
			// than merely comparing two coincidentally identical previews.
			unmasked := preview.Request()
			unmasked.MaskInput, unmasked.Options.Mask = "", nil
			if studioCall(t, s, unmasked).Data == preview.Data {
				t.Fatal("mask fixture does not change preview pixels")
			}
		})
	}
}

func TestConfigExternalMaskRecipeDoesNotMutateCaller(t *testing.T) {
	s, _ := studioFixture(t, 8, 8)
	callerImage := image.NewGray(image.Rect(0, 0, 1, 1))
	callerMask := &engine.Mask{Shape: "image", Invert: true, X: 3, Image: callerImage}
	q := RenderRequest{MaskInput: "source.png", Options: engine.Config{Mask: callerMask}}
	cfg, recipe, _, err := s.configWithMaskDigest(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if recipe.Options.Mask == callerMask || recipe.Options.Mask == cfg.Mask || recipe.Options.Mask.Image != nil {
		t.Fatal("recipe retains mutable caller/runtime mask data")
	}
	if !recipe.Options.Mask.Invert || recipe.Options.Mask.X != 3 || cfg.Mask.Image == nil || !cfg.Mask.Invert {
		t.Fatal("mask options or runtime mask were lost")
	}
	recipe.Options.Mask.Invert = false
	if callerMask.Image != callerImage || !callerMask.Invert || callerMask.X != 3 {
		t.Fatal("normalizing recipe changed caller-owned mask options")
	}
}
