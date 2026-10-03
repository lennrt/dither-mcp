// Command palette-atlas exports the engine’s palette catalog and a complete swatch atlas.
// It also creates twelve image treatments with portable recipes.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var paper = color.NRGBA{240, 238, 230, 255}
var ink = color.NRGBA{20, 20, 19, 255}
var muted = color.NRGBA{61, 61, 58, 255}
var line = color.NRGBA{204, 203, 200, 255}

var treatmentIDs = []string{"copperplate", "midnight-orchid", "alpine-morning", "tidal-glass", "malachite", "autumn-orchard", "ultraviolet-city", "pistachio-rose", "black-sesame", "nebula-rose", "brutalist-sun", "oat-and-ink"}

type treatment struct {
	ID        string `json:"id"`
	Image     string `json:"image"`
	Algorithm string `json:"algorithm"`
	Source    string `json:"source"`
}
type category struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}
type catalogue struct {
	Count      int              `json:"count"`
	Categories []category       `json:"categories"`
	Palettes   []engine.Palette `json:"palettes"`
	Treatments []treatment      `json:"treatments"`
}
type recipe struct {
	Version int           `json:"version"`
	Palette string        `json:"palette"`
	Options engine.Config `json:"options"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "palette-atlas:", err)
		os.Exit(1)
	}
}
func face(data []byte, size float64) font.Face {
	f, err := opentype.Parse(data)
	must(err)
	out, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	must(err)
	return out
}
func text(img draw.Image, f font.Face, x, y int, c color.Color, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: f, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
func fill(img draw.Image, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}
func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	must(err)
	must(png.Encode(f, img))
	must(f.Close())
	fmt.Println(path)
}
func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0644))
	fmt.Println(path)
}
func swatch(img draw.Image, r image.Rectangle, colors []string) {
	for i, h := range colors {
		c, err := engine.ParseHex(h)
		must(err)
		cell := image.Rect(r.Min.X+i*r.Dx()/len(colors), r.Min.Y, r.Min.X+(i+1)*r.Dx()/len(colors), r.Max.Y)
		fill(img, cell, c)
	}
}

func main() {
	out := flag.String("out", "docs/assets", "Output directory")
	flag.Parse()
	must(os.MkdirAll(*out, 0755))
	must(os.MkdirAll(filepath.Join(*out, "recipes"), 0755))
	palettes := engine.Palettes()
	sort.Slice(palettes, func(i, j int) bool { return palettes[i].ID < palettes[j].ID })
	byID := map[string]engine.Palette{}
	counts := map[string]int{}
	for _, p := range palettes {
		byID[p.ID] = p
		counts[p.Category]++
	}
	cat := catalogue{Count: len(palettes), Palettes: palettes}
	for id, n := range counts {
		cat.Categories = append(cat.Categories, category{id, n})
	}
	sort.Slice(cat.Categories, func(i, j int) bool { return cat.Categories[i].ID < cat.Categories[j].ID })
	sourceFile, err := os.Open("docs/assets/source/moon-garden.png")
	must(err)
	source, err := png.Decode(sourceFile)
	must(err)
	must(sourceFile.Close())
	labels := face(gomono.TTF, 16)
	small := face(gomono.TTF, 12)
	serif, err := os.ReadFile("docs/assets/fonts/SourceSerif4-Regular.ttf")
	must(err)
	titles := face(serif, 58)
	tileTitle := face(serif, 27)
	defer labels.Close()
	defer small.Close()
	defer titles.Close()
	defer tileTitle.Close()
	// The atlas groups the complete catalog by category at full resolution.
	ordered := append([]engine.Palette(nil), palettes...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Category != ordered[j].Category {
			return ordered[i].Category < ordered[j].Category
		}
		return ordered[i].ID < ordered[j].ID
	})
	const columns = 8
	const cellWidth = 290
	const cellHeight = 100
	rows := (len(ordered) + columns - 1) / columns
	atlas := image.NewNRGBA(image.Rect(0, 0, 2400, 200+rows*cellHeight))
	fill(atlas, atlas.Bounds(), paper)
	text(atlas, titles, 40, 77, ink, "Color with character.")
	text(atlas, labels, 40, 120, muted, fmt.Sprintf("%d CURATED PALETTES  /  %d CATEGORIES  /  GO ENGINE PALETTE CATALOG", len(palettes), len(cat.Categories)))
	fill(atlas, image.Rect(40, 148, 2360, 150), ink)
	for i, p := range ordered {
		x := 40 + (i%columns)*cellWidth
		y := 180 + (i/columns)*cellHeight
		text(atlas, labels, x, y+18, ink, p.ID)
		text(atlas, small, x, y+37, muted, fmt.Sprintf("%s / %d colors", strings.ToUpper(p.Category), len(p.Colors)))
		swatch(atlas, image.Rect(x, y+48, x+268, y+74), p.Colors)
		fill(atlas, image.Rect(x, y+84, x+268, y+85), line)
	}
	writePNG(filepath.Join(*out, "palette-atlas.png"), atlas)
	// Every treatment uses the same original source and the same image settings.
	const tileWidth = 480
	const tileHeight = 410
	gallery := image.NewNRGBA(image.Rect(0, 0, 1440, 4*tileHeight))
	fill(gallery, gallery.Bounds(), paper)
	for i, id := range treatmentIDs {
		p, ok := byID[id]
		if !ok {
			must(fmt.Errorf("required treatment palette %q is absent", id))
		}
		resolved, err := engine.ResolvePalette(id, nil)
		must(err)
		cfg := engine.Config{Algorithm: "floyd-steinberg", Width: 960, Height: 640, PixelScale: 2, Brightness: .06, Contrast: engine.Float(1.1), Seed: 42, Palette: resolved}
		img, err := engine.Process(context.Background(), source, cfg)
		must(err)
		name := "palette-" + id + ".png"
		writePNG(filepath.Join(*out, name), img)
		cfg.Palette = nil
		writeJSON(filepath.Join(*out, "recipes", "palette-"+id+".json"), recipe{1, id, cfg})
		cat.Treatments = append(cat.Treatments, treatment{id, name, "floyd-steinberg", "moon-garden"})
		x := (i % 3) * tileWidth
		y := (i / 3) * tileHeight
		for dy := 0; dy < 320; dy++ {
			for dx := 0; dx < tileWidth; dx++ {
				gallery.Set(x+dx, y+dy, img.At(dx*2, dy*2))
			}
		}
		text(gallery, tileTitle, x+18, y+350, ink, p.Name)
		text(gallery, small, x+18, y+373, muted, id+" / "+p.Category)
		swatch(gallery, image.Rect(x+18, y+384, x+tileWidth-18, y+402), p.Colors)
	}
	writePNG(filepath.Join(*out, "palette-treatments.png"), gallery)
	writeJSON(filepath.Join(*out, "palettes.json"), cat)
}
