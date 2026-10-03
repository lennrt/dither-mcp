// Command brand-banner creates the original clay banner as SVG and PNG.
// Both formats use the same vector paths, including outlined font glyphs.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const width, height = 1200, 370

type point struct{ x, y float32 }
type segment struct {
	op byte
	p  []point
}
type path []segment
type canvas struct {
	png *image.NRGBA
	svg strings.Builder
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "brand-banner:", err)
		os.Exit(1)
	}
}

func hex(h string) color.NRGBA {
	var r, g, b uint8
	_, err := fmt.Sscanf(h, "#%02x%02x%02x", &r, &g, &b)
	must(err)
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func (c *canvas) fill(p path, fill string) {
	minX, minY, maxX, maxY := float32(width), float32(height), float32(0), float32(0)
	for _, s := range p {
		for _, q := range s.p {
			minX, minY = min(minX, q.x), min(minY, q.y)
			maxX, maxY = max(maxX, q.x), max(maxY, q.y)
		}
	}
	bounds := image.Rect(int(math.Floor(float64(minX))), int(math.Floor(float64(minY))), int(math.Ceil(float64(maxX))), int(math.Ceil(float64(maxY))))
	z := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	var d strings.Builder
	for _, s := range p {
		d.WriteByte(s.op)
		for _, q := range s.p {
			fmt.Fprintf(&d, "%.3f %.3f ", q.x, q.y)
		}
		points := append([]point(nil), s.p...)
		for i := range points {
			points[i].x -= float32(bounds.Min.X)
			points[i].y -= float32(bounds.Min.Y)
		}
		switch s.op {
		case 'M':
			z.MoveTo(points[0].x, points[0].y)
		case 'L':
			z.LineTo(points[0].x, points[0].y)
		case 'Q':
			z.QuadTo(points[0].x, points[0].y, points[1].x, points[1].y)
		case 'C':
			z.CubeTo(points[0].x, points[0].y, points[1].x, points[1].y, points[2].x, points[2].y)
		case 'Z':
			z.ClosePath()
		}
	}
	z.Draw(c.png, bounds, image.NewUniform(hex(fill)), image.Point{})
	fmt.Fprintf(&c.svg, "<path fill=\"%s\" d=\"%s\"/>\n", fill, d.String())
}

func rect(x, y, w, h float32) path {
	return path{{'M', []point{{x, y}}}, {'L', []point{{x + w, y}}}, {'L', []point{{x + w, y + h}}}, {'L', []point{{x, y + h}}}, {'Z', nil}}
}

func circle(x, y, r float32) path {
	k := r * .55228475
	return path{
		{'M', []point{{x + r, y}}},
		{'C', []point{{x + r, y + k}, {x + k, y + r}, {x, y + r}}},
		{'C', []point{{x - k, y + r}, {x - r, y + k}, {x - r, y}}},
		{'C', []point{{x - r, y - k}, {x - k, y - r}, {x, y - r}}},
		{'C', []point{{x + k, y - r}, {x + r, y - k}, {x + r, y}}},
		{'Z', nil},
	}
}

func (c *canvas) text(data []byte, size, x, baseline float32, fill, s string, tracking float32) {
	f, err := sfnt.Parse(data)
	must(err)
	var b sfnt.Buffer
	var previous sfnt.GlyphIndex
	ppem := fixed.Int26_6(size * 64)
	for i, r := range s {
		index, err := f.GlyphIndex(&b, r)
		must(err)
		if i > 0 {
			kern, err := f.Kern(&b, previous, index, ppem, font.HintingNone)
			must(err)
			x += float32(kern)/64 + tracking
		}
		segments, err := f.LoadGlyph(&b, index, ppem, nil)
		must(err)
		var outline path
		for _, glyphSegment := range segments {
			var op byte
			var n int
			switch glyphSegment.Op {
			case sfnt.SegmentOpMoveTo:
				if len(outline) > 0 {
					outline = append(outline, segment{'Z', nil})
				}
				op, n = 'M', 1
			case sfnt.SegmentOpLineTo:
				op, n = 'L', 1
			case sfnt.SegmentOpQuadTo:
				op, n = 'Q', 2
			case sfnt.SegmentOpCubeTo:
				op, n = 'C', 3
			}
			points := make([]point, n)
			for j := range n {
				points[j] = point{x + float32(glyphSegment.Args[j].X)/64, baseline + float32(glyphSegment.Args[j].Y)/64}
			}
			outline = append(outline, segment{op, points})
		}
		if len(outline) > 0 {
			outline = append(outline, segment{'Z', nil})
			c.fill(outline, fill)
		}
		advance, err := f.GlyphAdvance(&b, index, ppem, font.HintingNone)
		must(err)
		x += float32(advance) / 64
		previous = index
	}
}

func main() {
	serif, err := os.ReadFile("docs/assets/fonts/SourceSerif4-Regular.ttf")
	must(err)
	c := canvas{png: image.NewNRGBA(image.Rect(0, 0, width, height))}
	draw.Draw(c.png, c.png.Bounds(), image.NewUniform(hex("#f0eee6")), image.Point{}, draw.Src)
	c.svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="370" viewBox="0 0 1200 370" role="img" aria-labelledby="title description">
<title id="title">dither-mcp — local image dithering for agents</title>
<desc id="description">A warm paper banner with a serif wordmark and an original crescent made from clay and ordered ink pixels. Forty-one algorithms. Two hundred fifty-six palettes. MCP and CLI.</desc>
<rect width="1200" height="370" fill="#f0eee6"/>
`)
	c.fill(rect(64, 62, 36, 3), "#d97757")
	c.text(gomono.TTF, 12, 114, 68, "#3d3d3a", "LOCAL IMAGE DITHERING FOR AGENTS", 1.0)
	c.text(serif, 104, 61, 186, "#141413", "dither-mcp", -2.2)
	c.text(goregular.TTF, 22, 66, 235, "#3d3d3a", "Color with character. Texture by design.", 0)
	c.text(gomono.TTF, 13, 66, 285, "#3d3d3a", "41 ALGORITHMS   /   256 PALETTES   /   MCP + CLI", .2)
	c.fill(rect(64, 324, 1072, 1), "#e3dacc")
	c.fill(circle(966, 169, 122), "#d97757")
	// Ordered ink coverage moves diagonally across the original crescent.
	// The 4×4 Bayer ranks set each cell's threshold without random sampling.
	bayer := [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}
	for row := 0; row < 49; row++ {
		for col := 0; col < 49; col++ {
			x, y := float64(846+col*5), float64(49+row*5)
			dx, dy := x+1.5-966, y+1.5-169
			if math.Hypot(dx, dy) > 117 {
				continue
			}
			coverage := .58 - .0025*dx - .0028*dy
			if float64(bayer[row%4][col%4])+.5 < coverage*16 {
				c.fill(rect(float32(x), float32(y), 3, 3), "#141413")
			}
		}
	}
	c.fill(circle(1007, 130, 88), "#f0eee6")
	// Sparse detached pixels give the crescent a quiet print texture.
	for _, p := range []point{{837, 155}, {831, 176}, {839, 197}, {847, 226}, {862, 245}, {882, 266}, {910, 288}} {
		c.fill(rect(p.x, p.y, 3, 3), "#d97757")
	}
	c.fill(circle(1110, 269, 4), "#d97757")
	c.svg.WriteString("</svg>\n")
	must(os.WriteFile("docs/assets/banner.svg", []byte(c.svg.String()), 0644))
	var b bytes.Buffer
	must(png.Encode(&b, c.png))
	must(os.WriteFile("docs/assets/banner.png", b.Bytes(), 0644))
	fmt.Println("docs/assets/banner.svg")
	fmt.Println("docs/assets/banner.png")
}
