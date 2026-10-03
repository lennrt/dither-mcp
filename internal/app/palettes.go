package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
)

const DefaultPaletteLimit = 32
const MaxPaletteLimit = 256

// PaletteQuery uses case-insensitive AND terms and exact category slugs. Empty
// values select the complete catalog. Results use stable lexical ID order with pagination.
type PaletteQuery struct {
	Query     string `json:"query,omitempty" jsonschema:"Separate search terms with whitespace. Every term must match a substring in the palette metadata or hex colors. Matching ignores case. Searchable metadata includes ID, name, description, category, tags, and origin. The maximum length is 256 UTF-8 bytes."`
	Category  string `json:"category,omitempty" jsonschema:"The category slug must match exactly, ignoring case. Omit it to search every category. The response categories array lists available slugs."`
	MinColors int    `json:"min_colors,omitempty" jsonschema:"The inclusive minimum palette size is 2 through 256. Omission or zero selects 2."`
	MaxColors int    `json:"max_colors,omitempty" jsonschema:"The inclusive maximum palette size is 2 through 256. It must be at least min_colors. Omission or zero selects 256."`
	Limit     int    `json:"limit,omitempty" jsonschema:"The page can contain at most this many palettes, from 1 through 256. Omission or zero selects 32."`
	Offset    int    `json:"offset,omitempty" jsonschema:"This nonnegative index selects a page from filtered results sorted by palette ID. Use next_offset from the previous response with the same filters. Offsets beyond the last result return an empty page."`
}
type PaletteCategory struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}
type PaletteList struct {
	Palettes   []engine.Palette  `json:"palettes"`
	Total      int               `json:"total" jsonschema:"This is the number of palettes in the complete library."`
	Matched    int               `json:"matched" jsonschema:"This is the number of palettes that match all filters before pagination."`
	Count      int               `json:"count" jsonschema:"This is the number of palettes in this page."`
	Offset     int               `json:"offset"`
	Limit      int               `json:"limit"`
	NextOffset *int              `json:"next_offset,omitempty" jsonschema:"This index selects the next page under the same filters. The response omits it after the last page."`
	Categories []PaletteCategory `json:"categories" jsonschema:"This array lists all categories and their unfiltered library counts, sorted by category ID."`
}

func paletteCategories(palettes []engine.Palette) []PaletteCategory {
	counts := map[string]int{}
	for _, p := range palettes {
		counts[p.Category]++
	}
	out := make([]PaletteCategory, 0, len(counts))
	for id, count := range counts {
		out = append(out, PaletteCategory{ID: id, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func listPalettes(q PaletteQuery) (PaletteList, error) {
	if len(q.Query) > 256 {
		return PaletteList{}, fmt.Errorf("query must be at most 256 UTF-8 bytes")
	}
	if q.MinColors == 0 {
		q.MinColors = 2
	}
	if q.MaxColors == 0 {
		q.MaxColors = 256
	}
	if q.MinColors < 2 || q.MinColors > 256 || q.MaxColors < 2 || q.MaxColors > 256 || q.MinColors > q.MaxColors {
		return PaletteList{}, fmt.Errorf("color bounds must be 2..256 with min_colors <= max_colors")
	}
	if q.Limit == 0 {
		q.Limit = DefaultPaletteLimit
	}
	if q.Limit < 1 || q.Limit > MaxPaletteLimit {
		return PaletteList{}, fmt.Errorf("limit must be 1..256 (0 selects the default 32)")
	}
	if q.Offset < 0 {
		return PaletteList{}, fmt.Errorf("offset must be nonnegative")
	}
	catalog := engine.Palettes()
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].ID < catalog[j].ID })
	result := PaletteList{Palettes: []engine.Palette{}, Total: len(catalog), Offset: q.Offset, Limit: q.Limit, Categories: paletteCategories(catalog)}
	q.Category = strings.ToLower(strings.TrimSpace(q.Category))
	if q.Category != "" {
		found := false
		for _, c := range result.Categories {
			if c.ID == q.Category {
				found = true
				break
			}
		}
		if !found {
			return PaletteList{}, fmt.Errorf("unknown category %q. Call dither_palettes with {} to discover categories", q.Category)
		}
	}
	terms := strings.Fields(strings.ToLower(q.Query))
	matches := make([]engine.Palette, 0, len(catalog))
	for _, p := range catalog {
		if q.Category != "" && p.Category != q.Category {
			continue
		}
		if len(p.Colors) < q.MinColors || len(p.Colors) > q.MaxColors {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{p.ID, p.Name, p.Category, p.Description, p.Origin, strings.Join(p.Tags, " "), strings.Join(p.Colors, " ")}, " "))
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
		}
		if matched {
			matches = append(matches, p)
		}
	}
	result.Matched = len(matches)
	start := min(q.Offset, len(matches))
	end := start + min(q.Limit, len(matches)-start)
	result.Palettes = append(result.Palettes, matches[start:end]...)
	result.Count = len(result.Palettes)
	if end < len(matches) {
		next := end
		result.NextOffset = &next
	}
	return result, nil
}
