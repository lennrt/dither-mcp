package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func randomStyleService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, root
}

func styleCall(t *testing.T, s *Service, raw string) RandomStyleResult {
	t.Helper()
	result, err := s.Do(context.Background(), "dither_random_style", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return result.(RandomStyleResult)
}

func TestRandomStyleSeedReplayAndBoundaries(t *testing.T) {
	s, _ := randomStyleService(t)
	for _, seed := range []int64{-MaxStyleSeed, -1, 0, 1, MaxStyleSeed} {
		raw := fmt.Sprintf(`{"seed":%d}`, seed)
		first, again := styleCall(t, s, raw), styleCall(t, s, raw)
		if first.Seed != seed || first.Recipe.Options.Seed != seed || !reflect.DeepEqual(first, again) {
			t.Fatalf("seed %d: unstable or mismatched result: %+v / %+v", seed, first, again)
		}
		q := RandomStyleRequest{Seed: &seed}
		if got, err := randomStyle(context.Background(), q); err != nil || !reflect.DeepEqual(first, got) || *q.Seed != seed {
			t.Fatalf("seed %d: direct request changed or disagreed with service: %v", seed, err)
		}
		wire, err := json.Marshal(first)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip RandomStyleResult
		if err := json.Unmarshal(wire, &roundTrip); err != nil || !reflect.DeepEqual(first, roundTrip) {
			t.Fatalf("seed %d: JSON lost recipe or integer precision: %v", seed, err)
		}
	}
}

func TestRandomStyleOmittedSeedIsReplayable(t *testing.T) {
	s, _ := randomStyleService(t)
	seeds := map[int64]bool{}
	for range 16 {
		result := styleCall(t, s, `{}`)
		if result.Seed < -MaxStyleSeed || result.Seed > MaxStyleSeed || result.Recipe.Options.Seed != result.Seed {
			t.Fatalf("unsafe random seed: %+v", result)
		}
		again := styleCall(t, s, fmt.Sprintf(`{"seed":%d}`, result.Seed))
		if !reflect.DeepEqual(result, again) {
			t.Fatal("omitted seed cannot replay returned recipe")
		}
		seeds[result.Seed] = true
	}
	if len(seeds) < 2 {
		t.Fatal("omitted seeds never produced a fresh selection")
	}
}

func TestRandomStyleRejectsInvalidArguments(t *testing.T) {
	s, root := randomStyleService(t)
	for _, raw := range []string{
		`null`, `[]`, `{"seed":null}`, `{"seed":true}`, `{"seed":"1"}`, `{"seed":1.5}`,
		`{"seed":9007199254740992}`, `{"seed":-9007199254740992}`,
		`{"seed":9223372036854775808}`, `{"input":"missing.png"}`, `{"output":"unexpected.json"}`,
		`{"seed":0,"typo":1}`, `{"seed":0} {"seed":1}`,
	} {
		if result, err := s.Do(context.Background(), "dither_random_style", []byte(raw)); err == nil || result != nil {
			t.Errorf("accepted %s: result=%+v error=%v", raw, result, err)
		}
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatalf("invalid arguments touched empty workspace: %v", err)
	}
}

func TestRandomStyleDiversityBoundsAndRendering(t *testing.T) {
	algorithms, palettes := engine.Catalog(), engine.Palettes()
	algorithmIDs, paletteIDs := map[string]bool{}, map[string]bool{}
	for _, algorithm := range algorithms {
		algorithmIDs[algorithm.ID] = true
	}
	for _, palette := range palettes {
		paletteIDs[palette.ID] = true
	}
	source := image.NewNRGBA(image.Rect(5, -3, 37, 21))
	for y := source.Rect.Min.Y; y < source.Rect.Max.Y; y++ {
		for x := source.Rect.Min.X; x < source.Rect.Max.X; x++ {
			source.SetNRGBA(x, y, color.NRGBA{uint8(x * 11), uint8(y * 17), uint8(x*7 + y*13), 255})
		}
	}
	before := append([]byte(nil), source.Pix...)
	seenAlgorithms, seenPalettes := map[string]bool{}, map[string]bool{}
	seenEffects := map[string]bool{}
	for seed := int64(0); seed < 512; seed++ {
		result, err := randomStyle(context.Background(), RandomStyleRequest{Seed: &seed})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		recipe, options := result.Recipe, result.Recipe.Options
		if recipe.Version != 1 || len(recipe.Colors) != 0 || len(options.Palette) != 0 || !algorithmIDs[options.Algorithm] || !paletteIDs[recipe.Palette] {
			t.Fatalf("seed %d: invalid catalog selection: %+v", seed, result)
		}
		if options.Width != 0 || options.Height != 0 || options.PixelScale != 0 || options.ResizeFilter != "" || options.Crop != nil || options.Mask != nil {
			t.Fatalf("seed %d: style changed geometry: %+v", seed, options)
		}
		if options.Brightness < -.08 || options.Brightness > .08 || options.Contrast == nil || *options.Contrast < .9 || *options.Contrast > 1.35 || options.Gamma == nil || *options.Gamma < .85 || *options.Gamma > 1.15 || options.Saturation == nil || *options.Saturation < .75 || *options.Saturation > 1.35 || (options.Strength != nil && (*options.Strength < .8 || *options.Strength > 1.1)) {
			t.Fatalf("seed %d: tone settings exceeded the creative range: %+v", seed, options)
		}
		effects := options.Effects
		if effects.Glitch != 0 || effects.PixelSort || effects.Scanlines > .18 || effects.CRT > .18 || effects.Noise > .06 {
			t.Fatalf("seed %d: excessive post-effects: %+v", seed, effects)
		}
		effectCount := 0
		for name, value := range map[string]float64{"scanlines": effects.Scanlines, "crt": effects.CRT, "noise": effects.Noise} {
			if value != 0 {
				seenEffects[name] = true
				effectCount++
			}
		}
		if effectCount > 1 {
			t.Fatalf("seed %d: stacked textures", seed)
		}
		if effectCount == 0 {
			seenEffects["none"] = true
		}
		if err := validateRecipe(context.Background(), recipe); err != nil {
			t.Fatalf("seed %d: invalid portable recipe: %v", seed, err)
		}
		if !seenAlgorithms[options.Algorithm] {
			options.Palette, err = engine.ResolvePalette(recipe.Palette, nil)
			if err != nil {
				t.Fatal(err)
			}
			first, err := engine.Process(context.Background(), source, options)
			if err != nil {
				t.Fatalf("seed %d, %s could not render: %v", seed, options.Algorithm, err)
			}
			again, err := engine.Process(context.Background(), source, options)
			if err != nil || !bytes.Equal(first.Pix, again.Pix) || first.Bounds() != image.Rect(0, 0, 32, 24) {
				t.Fatalf("seed %d, %s rendered inconsistently or changed dimensions: %v", seed, options.Algorithm, err)
			}
		}
		seenAlgorithms[options.Algorithm], seenPalettes[recipe.Palette] = true, true
	}
	if len(seenAlgorithms) != len(algorithms) || len(seenPalettes) < len(palettes)/2 || len(seenEffects) != 4 {
		t.Fatalf("insufficient style variety: %d/%d algorithms, %d/%d palettes, effects %v", len(seenAlgorithms), len(algorithms), len(seenPalettes), len(palettes), seenEffects)
	}
	if !bytes.Equal(before, source.Pix) || !reflect.DeepEqual(algorithms, engine.Catalog()) || !reflect.DeepEqual(palettes, engine.Palettes()) {
		t.Fatal("generation or rendering changed source or catalogs")
	}
}

func TestRandomStyleDoesNotReadOrWriteWorkspace(t *testing.T) {
	s, root := randomStyleService(t)
	marker := []byte("This is not an image and must remain untouched.")
	if err := os.WriteFile(filepath.Join(root, "marker"), marker, 0600); err != nil {
		t.Fatal(err)
	}
	first := styleCall(t, s, `{"seed":42}`)
	// All returned recipe settings are owned by this response.
	*first.Recipe.Options.Contrast = 0
	again := styleCall(t, s, `{"seed":42}`)
	if *again.Recipe.Options.Contrast == 0 {
		t.Fatal("caller mutation changed future styles")
	}
	after, err := os.ReadFile(filepath.Join(root, "marker"))
	files, listErr := os.ReadDir(root)
	if err != nil || listErr != nil || !bytes.Equal(marker, after) || len(files) != 1 {
		t.Fatalf("style generation mutated workspace: %v / %v", err, listErr)
	}
	// A closed workspace root also cannot affect this image-independent tool.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if closedRoot := styleCall(t, s, `{"seed":42}`); !reflect.DeepEqual(again, closedRoot) {
		t.Fatal("generation depends on workspace access")
	}
}

func TestRandomStyleCancellationReleasesAdmission(t *testing.T) {
	s, _ := randomStyleService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := s.Do(ctx, "dither_random_style", []byte(`{"seed":42}`)); !errors.Is(err, context.Canceled) || result != nil {
		t.Fatalf("cancelled generation returned result=%v error=%v", result, err)
	}
	styleCall(t, s, `{"seed":42}`)
}
