// Command showcase-render creates the showcase with the Go engine.
// Each exported image has a versioned recipe beside it, with a fixed random seed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"

	"github.com/lennrt/dither-mcp/engine"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type recipe struct {
	Version int           `json:"version"`
	Palette string        `json:"palette,omitempty"`
	Colors  []string      `json:"colors,omitempty"`
	Options engine.Config `json:"options"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func read(path string) image.Image {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	im, err := png.Decode(f)
	must(err)
	return im
}
func write(path string, im image.Image) {
	f, err := os.Create(path)
	must(err)
	must(png.Encode(f, im))
	must(f.Close())
	fmt.Println(path)
}
func render(name string, source image.Image, preset string, colors []string, cfg engine.Config) *image.NRGBA {
	p, err := engine.ResolvePalette(preset, colors)
	must(err)
	cfg.Palette = p
	out, err := engine.Process(context.Background(), source, cfg)
	must(err)
	write("docs/assets/"+name+".png", out)
	cfg.Palette = nil
	b, err := json.MarshalIndent(recipe{1, preset, colors, cfg}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join("docs/assets/recipes", name+".json"), append(b, '\n'), 0644))
	return out
}
func composite(images []image.Image, columns int, w, h int) *image.NRGBA {
	rows := (len(images) + columns - 1) / columns
	canvas := image.NewNRGBA(image.Rect(0, 0, columns*w, rows*h))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.NRGBA{238, 233, 222, 255}}, image.Point{}, draw.Src)
	for i, im := range images {
		cell := image.Rect((i%columns)*w, (i/columns)*h, (i%columns+1)*w, (i/columns+1)*h)
		for y := cell.Min.Y; y < cell.Max.Y; y++ {
			for x := cell.Min.X; x < cell.Max.X; x++ {
				sx := (x-cell.Min.X)*im.Bounds().Dx()/w + im.Bounds().Min.X
				sy := (y-cell.Min.Y)*im.Bounds().Dy()/h + im.Bounds().Min.Y
				canvas.Set(x, y, im.At(sx, sy))
			}
		}
	}
	return canvas
}
func main() {
	first := flag.String("first-frame", "", "Extract the first GIF frame instead of rendering the gallery")
	output := flag.String("out", "docs/assets/garden-wave-still.png", "PNG output path for the first GIF frame")
	flag.Parse()
	if *first != "" {
		f, err := os.Open(*first)
		must(err)
		frame, err := gif.Decode(f)
		must(err)
		must(f.Close())
		write(*output, frame)
		return
	}
	must(os.MkdirAll("docs/assets/recipes", 0755))
	garden := read("docs/assets/source/moon-garden.png")
	mineral := read("docs/assets/source/mineral-nocturne.png")
	base := engine.Config{Algorithm: "atkinson", Width: 960, PixelScale: 2, Brightness: .06, Contrast: engine.Float(1.1), Seed: 42}
	green := render("garden-gameboy", garden, "gameboy", nil, base)
	clay := render("garden-clay", garden, "", []string{"#141413", "#3d3d3a", "#d97757", "#e3dacc", "#f0eee6", "#faf9f5"}, base)
	base.Algorithm = "bayer-4"
	cga := render("garden-cga", garden, "cga", nil, base)
	base.Algorithm = "floyd-steinberg"
	ember := render("mineral-ember", mineral, "", []string{"#251d32", "#ab4e3b", "#e6ad62", "#f0e7d3"}, base)
	base.Algorithm = "atkinson"
	paper := render("garden-paper", garden, "", []string{"#242a25", "#eee9de"}, base)
	// A detail crop keeps each screen's texture visible at ordinary browser sizes.
	study := image.NewNRGBA(image.Rect(0, 0, 600, 450))
	draw.Draw(study, study.Bounds(), garden, image.Pt(200, 100), draw.Src)
	write("docs/assets/source/moon-garden-detail.png", study)
	var studies []image.Image
	for _, a := range []struct{ id, name string }{{"atkinson", "atkinson"}, {"floyd-steinberg", "floyd-steinberg"}, {"bayer-8", "bayer-8"}, {"halftone-8", "halftone"}, {"blue-noise", "blue-noise"}, {"riemersma", "riemersma"}} {
		cfg := engine.Config{Algorithm: a.id, Width: 720, Height: 540, PixelScale: 2, Brightness: .04, Contrast: engine.Float(1.1), Seed: 42}
		studies = append(studies, render("study-"+a.name, study, "", []string{"#242a25", "#eee9de"}, cfg))
	}
	write("docs/assets/algorithm-study.png", composite(studies, 3, 480, 360))
	write("docs/assets/palette-study.png", composite([]image.Image{green, ember, cga}, 3, 480, 320))
	hero := image.NewNRGBA(image.Rect(0, 0, 960, 684))
	draw.Draw(hero, hero.Bounds(), &image.Uniform{color.NRGBA{240, 238, 230, 255}}, image.Point{}, draw.Src)
	draw.Draw(hero, hero.Bounds(), garden, image.Point{}, draw.Src)
	draw.Draw(hero, image.Rect(480, 0, 960, 640), clay, image.Pt(480, 0), draw.Src)
	draw.Draw(hero, image.Rect(479, 0, 481, 640), &image.Uniform{color.NRGBA{240, 238, 230, 255}}, image.Point{}, draw.Src)
	f, err := opentype.Parse(goregular.TTF)
	must(err)
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingFull})
	must(err)
	defer face.Close()
	for _, label := range []struct {
		x    int
		text string
	}{{18, "MOON GARDEN / ORIGINAL"}, {498, "ATKINSON / SIX-COLOR CLAY"}} {
		d := font.Drawer{Dst: hero, Src: image.NewUniform(color.NRGBA{61, 61, 58, 255}), Face: face, Dot: fixed.P(label.x, 667)}
		d.DrawString(label.text)
	}
	write("docs/assets/hero.png", hero)
	_ = paper
}
