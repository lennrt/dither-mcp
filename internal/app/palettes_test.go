package app

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func paletteCall(t *testing.T, svc *Service, q PaletteQuery) PaletteList {
	t.Helper()
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	v, err := svc.Do(context.Background(), "dither_palettes", b)
	if err != nil {
		t.Fatal(err)
	}
	return v.(PaletteList)
}
func TestPaletteDiscoveryPagination(t *testing.T) {
	svc, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	first := paletteCall(t, svc, PaletteQuery{})
	if first.Total != 256 || first.Matched != 256 || first.Count != 32 || first.Limit != 32 || first.Offset != 0 || first.NextOffset == nil || *first.NextOffset != 32 {
		t.Fatalf("default page: %+v", first)
	}
	sum := 0
	previous := ""
	for _, c := range first.Categories {
		if c.ID <= previous || c.Count < 1 {
			t.Fatal("category ordering/count")
		}
		sum += c.Count
		previous = c.ID
	}
	if sum != first.Total || len(first.Categories) != 16 {
		t.Fatal("category distribution")
	}
	expected := make([]string, 0, first.Total)
	for _, p := range engine.Palettes() {
		expected = append(expected, p.ID)
	}
	sort.Strings(expected)
	for _, limit := range []int{1, 7, 32, 100, 255, 256} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			var got []string
			offset := 0
			for pages := 0; pages <= first.Total; pages++ {
				result := paletteCall(t, svc, PaletteQuery{Limit: limit, Offset: offset})
				if result.Offset != offset || result.Count != len(result.Palettes) || result.Count > limit {
					t.Fatal("page metadata")
				}
				for _, p := range result.Palettes {
					got = append(got, p.ID)
				}
				if result.NextOffset == nil {
					break
				}
				if *result.NextOffset != offset+result.Count {
					t.Fatal("pagination must advance by returned count")
				}
				offset = *result.NextOffset
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("pagination gap/duplicate for limit%d", limit)
			}
		})
	}
	empty := paletteCall(t, svc, PaletteQuery{Offset: math.MaxInt, Limit: 256})
	if empty.Count != 0 || empty.NextOffset != nil || empty.Palettes == nil || empty.Offset != math.MaxInt {
		t.Fatal("out-of-range offset must be empty JSON array")
	}
}
func TestPaletteDiscoveryFilterComposition(t *testing.T) {
	svc, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	q := PaletteQuery{Query: "  GaMe BoY ", Category: " RETRO ", MinColors: 4, MaxColors: 4, Limit: 256}
	result := paletteCall(t, svc, q)
	found := false
	for _, p := range result.Palettes {
		if p.Category != "retro" || len(p.Colors) != 4 {
			t.Fatalf("filter leak: %+v", p)
		}
		if p.ID == "gameboy" {
			found = true
		}
		if _, err := engine.ResolvePalette(p.ID, nil); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Fatal("search lost compatible Game Boy preset")
	}
	exact := paletteCall(t, svc, PaletteQuery{Query: "gameboy-pocket"})
	if exact.Matched != 1 || exact.Palettes[0].ID != "gameboy-pocket" {
		t.Fatal("exact ID discovery")
	}
	none := paletteCall(t, svc, PaletteQuery{Query: "gameboy definitely-no-such-token"})
	if none.Matched != 0 || none.Count != 0 || none.NextOffset != nil {
		t.Fatal("AND terms must all match")
	}
	hex := paletteCall(t, svc, PaletteQuery{Query: "#0F380F"})
	found = false
	for _, p := range hex.Palettes {
		if p.ID == "gameboy" {
			found = true
		}
	}
	if !found {
		t.Fatal("hex search is case insensitive")
	}
	all := paletteCall(t, svc, PaletteQuery{Category: "ocean", MaxColors: 8, Limit: 256})
	if all.Count < 2 {
		t.Fatal("ocean variety missing")
	}
	var pages []engine.Palette
	offset := 0
	for {
		part := paletteCall(t, svc, PaletteQuery{Category: "ocean", MaxColors: 8, Limit: 3, Offset: offset})
		pages = append(pages, part.Palettes...)
		if part.NextOffset == nil {
			break
		}
		offset = *part.NextOffset
	}
	if !reflect.DeepEqual(pages, all.Palettes) {
		t.Fatal("filter+pagination result differs from full filtered catalog")
	}
	// Mutating a caller-owned result must not alter later discovery or resolution.
	result.Palettes[0].Colors[0] = "#123456"
	result.Palettes[0].Tags[0] = "changed"
	again := paletteCall(t, svc, q)
	if reflect.DeepEqual(result.Palettes, again.Palettes) {
		t.Fatal("catalog mutation escaped result")
	}
}
func TestPaletteDiscoveryInvalidInputs(t *testing.T) {
	svc, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	for _, q := range []PaletteQuery{{Query: strings.Repeat("a", 257)}, {Query: strings.Repeat("é", 129)}, {Category: "missing"}, {MinColors: 1}, {MaxColors: 257}, {MinColors: 8, MaxColors: 4}, {Limit: -1}, {Limit: 257}, {Offset: -1}} {
		b, _ := json.Marshal(q)
		if _, err := svc.Do(context.Background(), "dither_palettes", b); err == nil {
			t.Fatalf("accepted invalid discovery: %s", b)
		}
	}
	if _, err := svc.Do(context.Background(), "dither_palettes", []byte(`{"query":"ocean","surprise":true}`)); err == nil {
		t.Fatal("unknown discovery argument accepted")
	}
}
