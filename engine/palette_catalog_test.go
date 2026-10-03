package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"image/color"
	stdpalette "image/color/palette"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestPaletteLibraryQuality(t *testing.T) {
	catalog := Palettes()
	if len(catalog) != 256 {
		t.Fatalf("palette library has %d entries, want 256", len(catalog))
	}
	identifier := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hexColor := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	ids, names, descriptions, signatures := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]string{}
	categories, sizes, origins := map[string]int{}, map[int]int{}, map[string]int{}
	for _, p := range catalog {
		if !identifier.MatchString(p.ID) || ids[p.ID] {
			t.Errorf("invalid or duplicate ID %q", p.ID)
		}
		ids[p.ID] = true
		if strings.TrimSpace(p.Name) == "" || names[p.Name] {
			t.Errorf("empty or duplicate display name for %s", p.ID)
		}
		names[p.Name] = true
		if len(p.Description) < 45 || descriptions[p.Description] {
			t.Errorf("insufficient or duplicated description for %s", p.ID)
		}
		descriptions[p.Description] = true
		if !identifier.MatchString(p.Category) {
			t.Errorf("invalid category for %s: %q", p.ID, p.Category)
		}
		if len(p.Tags) < 2 {
			t.Errorf("%s needs descriptive tags", p.ID)
		}
		tags := map[string]bool{}
		for _, tag := range p.Tags {
			if !identifier.MatchString(tag) || tags[tag] {
				t.Errorf("%s has invalid or repeated tag %q", p.ID, tag)
			}
			tags[tag] = true
		}
		if p.Origin != "original" && p.Origin != "reference" && p.Origin != "approximation" {
			t.Errorf("%s has unknown origin %q", p.ID, p.Origin)
		}
		if p.Origin != "original" && p.Source == "" {
			t.Errorf("%s must identify its reference or historical context", p.ID)
		}
		if p.Source != "" {
			u, err := url.Parse(p.Source)
			if err != nil || u.Scheme != "https" || u.Host == "" {
				t.Errorf("%s has invalid source URL %q", p.ID, p.Source)
			}
		}
		if len(p.Colors) < 2 || len(p.Colors) > 256 {
			t.Errorf("%s has %d colors", p.ID, len(p.Colors))
		}
		colors := map[string]bool{}
		for _, c := range p.Colors {
			if !hexColor.MatchString(c) || colors[c] {
				t.Errorf("%s has invalid or repeated color %q", p.ID, c)
			}
			colors[c] = true
		}
		ordered := append([]string(nil), p.Colors...)
		slices.Sort(ordered)
		signature := strings.Join(ordered, ",")
		if previous := signatures[signature]; previous != "" {
			t.Errorf("%s repeats %s's unordered color set", p.ID, previous)
		}
		signatures[signature] = p.ID
		categories[p.Category]++
		sizes[len(p.Colors)]++
		origins[p.Origin]++
	}
	wantCategories := []string{"architecture", "botanical", "cosmic", "duotone", "food", "interface", "landscape", "mineral", "neon", "neutral", "ocean", "pastel", "print", "retro", "seasonal", "terminal"}
	if len(categories) != len(wantCategories) {
		t.Errorf("category breadth: got %d, want %d", len(categories), len(wantCategories))
	}
	for _, category := range wantCategories {
		if categories[category] < 12 {
			t.Errorf("category %s has only %d palettes", category, categories[category])
		}
	}
	compact, medium, broad := 0, 0, 0
	for n, count := range sizes {
		switch {
		case n <= 4:
			compact += count
		case n <= 12:
			medium += count
		default:
			broad += count
		}
	}
	if compact < 30 || medium < 150 || broad < 10 || sizes[256] == 0 {
		t.Errorf("insufficient palette-size variety: compact=%d, medium=%d, broad=%d, sizes=%v", compact, medium, broad, sizes)
	}
	if origins["original"] < 200 || origins["reference"] < 5 || origins["approximation"] < 5 {
		t.Errorf("catalog should combine substantial original design with identified historical families: %v", origins)
	}
}

func TestLegacyPaletteCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/legacy-palettes.json")
	if err != nil {
		t.Fatal(err)
	}
	var legacy []struct {
		ID     string   `json:"id"`
		Name   string   `json:"name"`
		Colors []string `json:"colors"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	if len(legacy) != 22 {
		t.Fatalf("migration snapshot must contain all 22 original palettes")
	}
	catalog := Palettes()
	for i, old := range legacy {
		current := catalog[i]
		if current.ID != old.ID || current.Name != old.Name || !reflect.DeepEqual(current.Colors, old.Colors) {
			t.Errorf("legacy palette %s changed. Exact color order is part of recipe compatibility", old.ID)
		}
		resolved, err := ResolvePalette(old.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		for j, c := range resolved {
			if Hex(c) != old.Colors[j] {
				t.Errorf("%s index %d resolves differently", old.ID, j)
			}
		}
	}
}

func TestPaletteLibraryReferences(t *testing.T) {
	for _, tc := range []struct {
		id     string
		colors color.Palette
	}{
		{"plan9-256", stdpalette.Plan9},
		{"web-safe-216", stdpalette.WebSafe},
	} {
		t.Run(tc.id, func(t *testing.T) {
			got, err := ResolvePalette(tc.id, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.colors) {
				t.Fatalf("reference palette has %d colors, want %d", len(got), len(tc.colors))
			}
			for i, c := range tc.colors {
				if got[i] != color.NRGBAModel.Convert(c).(color.NRGBA) {
					t.Errorf("reference palette color %d differs from Go's documented table", i)
				}
			}
		})
	}
	for id, count := range map[string]int{"apple-ii": 15, "zx-spectrum": 15, "msx": 15, "nes": 55, "teletext": 8, "grayscale-16": 16, "rgb-workbench-64": 64} {
		p, err := ResolvePalette(id, nil)
		if err != nil || len(p) != count {
			t.Errorf("historical or broad-color family %s: count=%d, err=%v", id, len(p), err)
		}
	}
	gray, err := ResolvePalette("grayscale-16", nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range gray {
		v := uint8(i * 17)
		if c != (color.NRGBA{v, v, v, 255}) {
			t.Errorf("gray index %d: got %v, want %d", i, c, v)
		}
	}
}

func TestEveryPaletteRenders(t *testing.T) {
	img := fixture(23, 17)
	original := append([]byte(nil), img.Pix...)
	for _, p := range Palettes() {
		t.Run(p.ID, func(t *testing.T) {
			palette, err := ResolvePalette(p.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			membership := map[color.NRGBA]bool{}
			for _, c := range palette {
				membership[c] = true
			}
			for _, algorithm := range []string{"atkinson", "bayer-8"} {
				cfg := Config{Algorithm: algorithm, Palette: palette, Seed: 91, Serpentine: true}
				for _, space := range []string{"srgb", "linear-rgb"} {
					cfg.ColorSpace = space
					a, err := Process(context.Background(), img, cfg)
					if err != nil {
						t.Fatal(err)
					}
					b, err := Process(context.Background(), img, cfg)
					if err != nil || !bytes.Equal(a.Pix, b.Pix) {
						t.Errorf("%s/%s is not deterministic: %v", algorithm, space, err)
					}
					for y := 0; y < img.Bounds().Dy(); y++ {
						for x := 0; x < img.Bounds().Dx(); x++ {
							c := a.NRGBAAt(x, y)
							if c.A != img.NRGBAAt(x, y).A {
								t.Errorf("%s/%s changed alpha at %d,%d", algorithm, space, x, y)
							}
							if c.A != 0 {
								c.A = 255
								if !membership[c] {
									t.Fatalf("%s/%s emitted %s outside palette", algorithm, space, Hex(c))
								}
							}
						}
					}
				}
			}
		})
	}
	if !bytes.Equal(original, img.Pix) {
		t.Fatal("rendering mutated the input fixture")
	}
}

func TestPaletteLibraryDefensiveCopies(t *testing.T) {
	original := Palettes()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			copy := Palettes()
			for i := range copy {
				copy[i].Colors[0] = "#123456"
				copy[i].Tags[0] = "changed"
				copy[i].Description = "changed"
			}
		})
	}
	wg.Wait()
	if !reflect.DeepEqual(original, Palettes()) {
		t.Fatal("catalog caller mutated shared preset data")
	}
	p, err := ResolvePalette("midnight-orchid", nil)
	if err != nil {
		t.Fatal(err)
	}
	p[0] = color.NRGBA{255, 0, 0, 255}
	fresh, _ := ResolvePalette("midnight-orchid", nil)
	if fresh[0] == p[0] {
		t.Fatal("resolved colors share mutable backing storage")
	}
}
