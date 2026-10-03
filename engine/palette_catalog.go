package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// paletteCatalogJSON is a reviewed static color library. Recipes reference
// stable palette IDs. Changes to an existing color list require an explicit
// compatibility decision. This project includes its own original palette designs.
//
//go:embed palettes.json
var paletteCatalogJSON []byte

var presets = loadPaletteCatalog()

func loadPaletteCatalog() []Palette {
	var catalog struct {
		SchemaVersion int       `json:"schema_version"`
		Palettes      []Palette `json:"palettes"`
	}
	if err := json.Unmarshal(paletteCatalogJSON, &catalog); err != nil {
		panic(fmt.Sprintf("engine: invalid embedded palette catalog: %v", err))
	}
	if catalog.SchemaVersion != 1 {
		panic("engine: unsupported embedded palette catalog schema")
	}
	return catalog.Palettes
}
