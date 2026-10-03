// Command palette-docs generates an offline Markdown reference from the palette catalog.
// Use -check in CI to detect differences between the catalog and documentation.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
)

func main() {
	output := flag.String("output", "docs/palettes.md", "Path to the Markdown file")
	check := flag.Bool("check", false, "Check the existing output")
	flag.Parse()
	all := engine.Palettes()
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	byCategory := map[string][]engine.Palette{}
	for _, p := range all {
		byCategory[p.Category] = append(byCategory[p.Category], p)
	}
	categories := make([]string, 0, len(byCategory))
	for c := range byCategory {
		categories = append(categories, c)
	}
	sort.Strings(categories)
	var b bytes.Buffer
	fmt.Fprintf(&b, "# The palette library\n\nThe library contains **%d palettes across %d categories**. Use them with the same image pipeline, recipes, comparisons, animations, and print workflows. Each palette includes exact sRGB hex colors, a description, searchable tags, and provenance. The original 22 palette IDs and colors remain stable.\n\n", len(all), len(categories))
	b.WriteString(`## Find a palette

The CLI and MCP use the same search. The default page contains 32 palettes
sorted by ID. Request up to 256 palettes per page. For more results, use ` + "`next_offset`" + `.

` + "```sh" + `
./bin/dither-mcp palettes --query 'game boy' --max-colors 4
./bin/dither-mcp palettes --category ocean --min-colors 4 --max-colors 8 --limit 8
./bin/dither-mcp palettes --category print --limit 8
./bin/dither-mcp palettes --limit 256
` + "```" + `

` + "```json" + `
{
  "name": "dither_palettes",
  "arguments": {
    "query": "ocean",
    "min_colors": 4,
    "max_colors": 8,
    "limit": 8,
    "offset": 0
  }
}
` + "```" + `

Separate query terms with whitespace. Search ignores case and matches each term
as a substring. Every term must match at least one of these fields:

- ID or name.
- Description or category.
- Tags or origin.
- Hex colors.

Category and color-count filters further restrict the results. The response
contains these fields:

| Field | Meaning |
|---|---|
| ` + "`total`" + ` | Number of palettes in the complete library. |
| ` + "`matched`" + ` | Number of palettes that match all filters. |
| ` + "`count`" + ` | Number of palettes on this page. |
| ` + "`offset`" + ` | Requested start index in the filtered results. |
| ` + "`limit`" + ` | Maximum number of palettes on this page. |
| ` + "`next_offset`" + ` | Start index for the next page. The response omits this field at the end. |
| ` + "`categories`" + ` | Categories and their counts for the complete library. |

An offset beyond the results returns an empty array. Query text accepts up to
256 UTF-8 bytes. Color-count bounds range from 2 to 256. If the category is
unknown, the tool returns an error.

Use a returned ` + "`id`" + ` as the ` + "`palette`" + ` value in a render request.
You can also supply its ` + "`colors`" + ` array or store the palette in a version-1 recipe.
For print separation, choose a palette with up to 16 colors. Larger palettes work
with still images, comparisons, and animation.

## Browse categories

| Category | Palettes | Reference |
|---|---:|---|
`)
	for _, category := range categories {
		fmt.Fprintf(&b, "| %s | %d | [Browse %s](#%s) |\n", category, len(byCategory[category]), category, category)
	}
	b.WriteString(`
## Provenance and color interpretation

` + "`original`" + ` identifies palettes composed for this project. ` + "`reference`" + ` identifies
published palette values with a source link. ` + "`approximation`" + ` identifies an sRGB
interpretation of historical hardware, pigments, or displays. Descriptions explain the intended
appearance. Display calibration and print profiles are separate.

Color counts describe unique colors in each palette. Every preset has a distinct
unordered color set. Reordering the same colors does not create another preset.

[The visual atlas](assets/palette-atlas.png) presents the complete collection.
The standalone showcase includes interactive swatches, filters, and copy controls.
The Go binary embeds the catalog. Palette discovery uses only local data.
`)
	for _, category := range categories {
		fmt.Fprintf(&b, "\n## %s\n", strings.ToUpper(category[:1])+category[1:])
		for _, p := range byCategory[category] {
			fmt.Fprintf(&b, "\n### %s\n\n`%s` · %d colors · %s\n\n%s\n\n", p.Name, p.ID, len(p.Colors), p.Origin, p.Description)
			if len(p.Colors) <= 16 {
				fmt.Fprintf(&b, "Colors: `%s`\n\n", strings.Join(p.Colors, "`, `"))
			} else {
				b.WriteString("Colors, in palette order:\n\n```text\n")
				for start := 0; start < len(p.Colors); start += 8 {
					fmt.Fprintln(&b, strings.Join(p.Colors[start:min(start+8, len(p.Colors))], " "))
				}
				b.WriteString("```\n\n")
			}
			fmt.Fprintf(&b, "Tags: %s.\n", strings.Join(p.Tags, ", "))
			if p.Source != "" {
				fmt.Fprintf(&b, "\n[%s source](%s).\n", p.Name, p.Source)
			}
		}
	}
	if *check {
		existing, err := os.ReadFile(*output)
		if err != nil || !bytes.Equal(existing, b.Bytes()) {
			fmt.Fprintln(os.Stderr, "Palette reference differs from the engine catalog. Run make palette-docs.")
			os.Exit(1)
		}
		fmt.Println("Palette reference matches engine catalog")
		return
	}
	if err := os.WriteFile(*output, b.Bytes(), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
