package app_test

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/app"
)

func TestComparisonCellRecipeReplaysBeforeOutputEncoding(t *testing.T) {
	s, root, _ := workflowFixture(t)
	mask := image.NewGray(image.Rect(0, 0, 2, 1))
	mask.SetGray(1, 0, color.Gray{Y: 255})
	workflowWritePNG(t, root, "mask.png", mask)
	q := app.CompareRequest{
		RenderRequest: app.RenderRequest{
			Input: "input.png", Output: "reference.png", MaskInput: "mask.png",
			Options: engine.Config{Width: 13, Height: 9, PixelScale: 2, ResizeFilter: "area", Seed: 27, Effects: engine.Effects{Noise: .1}},
		},
		Palettes: []string{"cga", "gameboy"}, Columns: 2,
	}
	reference := workflowCall[app.CompareResult](t, s, "dither_compare", q)
	sheet := workflowDecode(t, workflowRead(t, root, reference.Artifact.Path))
	for _, format := range engine.Formats() {
		t.Run(format, func(t *testing.T) {
			q.Output = "sheet." + format
			result := workflowCall[app.CompareResult](t, s, "dither_compare", q)
			cell := result.Cells[1]
			if !reflect.DeepEqual(cell.Recipe, reference.Cells[1].Recipe) {
				t.Fatalf("output encoding changed the reusable recipe: %+v", cell.Recipe)
			}
			path := format + "-cell.json"
			workflowCall[app.Artifact](t, s, "dither_recipe_save", app.RecipeSaveRequest{Output: path, Recipe: cell.Recipe})
			artifact := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: q.Input, Output: format + "-cell.png", Recipe: path, MaskInput: q.MaskInput})
			replay := workflowDecode(t, workflowRead(t, root, artifact.Path))
			for y := 0; y < cell.Height; y++ {
				for x := 0; x < cell.Width; x++ {
					got := color.NRGBAModel.Convert(replay.At(x, y))
					want := color.NRGBAModel.Convert(sheet.At(cell.X+x, cell.Y+y))
					if got != want {
						t.Fatalf("processing pixel differs at %d,%d: %v != %v", x, y, got, want)
					}
				}
			}
		})
	}
}

func TestComparisonCellRecipesReplayPixels(t *testing.T) {
	for _, mode := range []string{"algorithms-default", "palettes-advanced", "custom-colors", "saved-recipe"} {
		t.Run(mode, func(t *testing.T) {
			s, root, _ := workflowFixture(t)
			q := app.CompareRequest{RenderRequest: app.RenderRequest{Input: "input.png", Output: "sheet.png"}, Algorithms: []string{"atkinson", "bayer-4"}, Columns: 2}
			switch mode {
			case "palettes-advanced":
				q.Algorithms = nil
				q.Palettes = []string{"gameboy", "cga"}
				q.Options = engine.Config{Algorithm: "random", Width: 13, Height: 9, Seed: 27, PixelScale: 2, Crop: &engine.Rect{X: 1, Y: 1, Width: 20, Height: 14}, Effects: engine.Effects{Scanlines: .1}}
				q.MaskInput = "mask.png"
				mask := image.NewGray(image.Rect(0, 0, 2, 1))
				mask.SetGray(1, 0, color.Gray{Y: 255})
				workflowWritePNG(t, root, q.MaskInput, mask)
			case "custom-colors":
				q.Colors = []string{"#112233", "#ffeedd"}
				q.Options.Height = 11
			case "saved-recipe":
				recipe := app.Recipe{Version: 1, Palette: "cga", Options: engine.Config{Width: 15, Seed: 19, Effects: engine.Effects{Noise: .03}}}
				workflowCall[app.Artifact](t, s, "dither_recipe_save", app.RecipeSaveRequest{Output: "base.json", Recipe: recipe})
				q.Recipe = "base.json"
			}
			type cellWithRecipe struct {
				app.Cell
				Recipe *app.Recipe `json:"recipe"`
			}
			type resultWithRecipes struct {
				Cells []cellWithRecipe `json:"cells"`
			}
			result := workflowCall[resultWithRecipes](t, s, "dither_compare", q)
			sheet := workflowDecode(t, workflowRead(t, root, q.Output))
			for i, cell := range result.Cells {
				if cell.Recipe == nil {
					t.Fatal("comparison cell has no reusable recipe")
				}
				if cell.Recipe.Options.Width != cell.Width || cell.Recipe.Options.Height != cell.Height {
					t.Fatalf("recipe geometry mismatch: %+v", cell)
				}
				if cell.Recipe.Options.Algorithm != cell.Algorithm {
					t.Fatal("recipe algorithm differs from candidate")
				}
				if q.Palettes != nil && cell.Recipe.Palette != cell.Palette {
					t.Fatal("recipe palette differs from candidate")
				}
				wire, _ := json.Marshal(cell.Recipe)
				if len(wire) == 0 {
					t.Fatal("empty recipe")
				}
				path := fmt.Sprintf("cell-%d.json", i)
				workflowCall[app.Artifact](t, s, "dither_recipe_save", app.RecipeSaveRequest{Output: path, Recipe: *cell.Recipe})
				artifact := workflowCall[app.Artifact](t, s, "dither_render", app.RenderRequest{Input: q.Input, Output: fmt.Sprintf("cell-%d.png", i), Recipe: path, MaskInput: q.MaskInput})
				replay := workflowDecode(t, workflowRead(t, root, artifact.Path))
				for y := 0; y < cell.Height; y++ {
					for x := 0; x < cell.Width; x++ {
						got := color.NRGBAModel.Convert(replay.At(x, y))
						want := color.NRGBAModel.Convert(sheet.At(cell.X+x, cell.Y+y))
						if got != want {
							t.Fatalf("cell %d replay differs at %d,%d: %v != %v", i, x, y, got, want)
						}
					}
				}
			}
		})
	}
}
