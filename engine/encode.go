package engine

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sort"
	"strings"
)

// Formats lists supported export formats.
func Formats() []string { return []string{"png", "jpeg", "gif", "svg", "pbm", "ascii"} }

func encode(ctx context.Context, format string, w io.Writer, img image.Image) (err error) {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = ctx.Err()
		}
	}()
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	if err := checkImage(img); err != nil {
		return err
	}
	w = contextWriter{ctx: ctx, writer: w}
	switch format {
	case "png":
		return png.Encode(w, img)
	case "jpeg", "jpg":
		opaque := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
		white := image.NewUniform(color.White)
		for y := 0; y < opaque.Bounds().Dy(); y++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			row := image.Rect(0, y, opaque.Bounds().Dx(), y+1)
			draw.Draw(opaque, row, white, image.Point{}, draw.Src)
			draw.Draw(opaque, row, img, img.Bounds().Min.Add(image.Pt(0, y)), draw.Over)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return jpeg.Encode(w, opaque, &jpeg.Options{Quality: 95})
	case "gif":
		p, err := PalettizeContext(ctx, img)
		if err != nil {
			return err
		}
		return gif.Encode(w, p, nil)
	case "svg":
		return encodeSVG(ctx, w, img)
	case "pbm":
		return encodePBM(ctx, w, img)
	case "ascii":
		return encodeASCII(ctx, w, img)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.writer.Write(p)
	if err != nil {
		return n, err
	}
	return n, w.ctx.Err()
}

// Palettize preserves all visible RGB colors when they fit into GIF's 256 slots.
// It reserves one transparent slot where needed. Larger sets use deterministic
// median cut and nearest-color mapping. GIF uses binary alpha. Pixels below 128
// become transparent. Pixels at or above 128 become opaque.
//
// Palettize maps colors without adding a second dither pattern.
func Palettize(img image.Image) (*image.Paletted, error) {
	return PalettizeContext(context.Background(), img)
}

// PalettizeContext is Palettize with cancellation checks between image rows.
// Palette extraction uses the same context. Cancellation returns no partial image.
func PalettizeContext(ctx context.Context, img image.Image) (*image.Paletted, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := checkImage(img); err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	read := pixelReader(img)
	seen := map[uint32]bool{}
	transparent := false
	overflow := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := read(x, y)
			if c.A < 128 {
				transparent = true
				continue
			}
			if !overflow {
				k := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
				seen[k] = true
				if len(seen) > 256 {
					overflow = true
				}
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := 256
	if transparent {
		limit = 255
	}
	var colors []color.NRGBA
	if len(seen) <= limit && !overflow {
		keys := make([]uint32, 0, len(seen))
		for k := range seen {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, k := range keys {
			colors = append(colors, color.NRGBA{uint8(k >> 16), uint8(k >> 8), uint8(k), 255})
		}
	} else {
		var err error
		colors, err = ExtractPalette(ctx, &gifPaletteImage{Image: img, read: pixelReader(img)}, limit)
		if err != nil {
			return nil, err
		}
	}
	palette := make(color.Palette, 0, len(colors)+1)
	for _, c := range colors {
		palette = append(palette, c)
	}
	transparentIndex := -1
	if transparent {
		transparentIndex = len(palette)
		palette = append(palette, color.NRGBA{})
	}
	if len(palette) == 0 {
		palette = append(palette, color.Black)
	}
	out := image.NewPaletted(image.Rect(0, 0, bounds.Dx(), bounds.Dy()), palette)
	exact := make(map[uint32]uint8, len(colors))
	for i, c := range colors {
		exact[uint32(c.R)<<16|uint32(c.G)<<8|uint32(c.B)] = uint8(i)
	}
	q := newQuantizer(colors, false)
	for y := 0; y < bounds.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := 0; x < bounds.Dx(); x++ {
			c := read(bounds.Min.X+x, bounds.Min.Y+y)
			index := uint8(0)
			if c.A < 128 && transparentIndex >= 0 {
				index = uint8(transparentIndex)
			} else {
				key := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
				if i, ok := exact[key]; ok {
					index = i
				} else {
					index = uint8(q.nearest(q.vector(c)))
				}
			}
			out.SetColorIndex(x, y, index)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func encodeSVG(ctx context.Context, w io.Writer, img image.Image) error {
	b := bufio.NewWriter(w)
	r := img.Bounds()
	read := pixelReader(img)
	if _, err := fmt.Fprintf(b, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" shape-rendering=\"crispEdges\">\n", r.Dx(), r.Dy(), r.Dx(), r.Dy()); err != nil {
		return err
	}
	for y := 0; y < r.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := 0; x < r.Dx(); {
			c := read(r.Min.X+x, r.Min.Y+y)
			start := x
			x++
			for x < r.Dx() {
				next := read(r.Min.X+x, r.Min.Y+y)
				if next != c {
					break
				}
				x++
			}
			if c.A == 0 {
				continue
			}
			var err error
			if c.A == 255 {
				_, err = fmt.Fprintf(b, "<path fill=\"%s\" d=\"M%d %dh%dv1H%dz\"/>\n", Hex(c), start, y, x-start, start)
			} else {
				_, err = fmt.Fprintf(b, "<path fill=\"%s\" fill-opacity=\"%.8f\" d=\"M%d %dh%dv1H%dz\"/>\n", Hex(c), float64(c.A)/255, start, y, x-start, start)
			}
			if err != nil {
				return err
			}
		}
	}
	if _, err := io.WriteString(b, "</svg>\n"); err != nil {
		return err
	}
	return b.Flush()
}

func matteLuminance(c color.NRGBA) float64 { a := float64(c.A) / 255; return luminance(c)*a + (1 - a) }
func encodePBM(ctx context.Context, w io.Writer, img image.Image) error {
	b := bufio.NewWriter(w)
	r := img.Bounds()
	read := pixelReader(img)
	if _, err := fmt.Fprintf(b, "P4\n%d %d\n", r.Dx(), r.Dy()); err != nil {
		return err
	}
	row := make([]byte, (r.Dx()+7)/8)
	for y := 0; y < r.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		clear(row)
		for x := 0; x < r.Dx(); x++ {
			c := read(r.Min.X+x, r.Min.Y+y)
			if matteLuminance(c) < .5 {
				row[x/8] |= 1 << uint(7-x%8)
			}
		}
		if _, err := b.Write(row); err != nil {
			return err
		}
	}
	return b.Flush()
}
func encodeASCII(ctx context.Context, w io.Writer, img image.Image) error {
	const ramp = "@%#*+=-:. "
	b := bufio.NewWriter(w)
	r := img.Bounds()
	read := pixelReader(img)
	row := make([]byte, r.Dx()+1)
	row[len(row)-1] = '\n'
	for y := 0; y < r.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := 0; x < r.Dx(); x++ {
			c := read(r.Min.X+x, r.Min.Y+y)
			index := int(matteLuminance(c)*float64(len(ramp)-1) + .5)
			row[x] = ramp[index]
		}
		if _, err := b.Write(row); err != nil {
			return err
		}
	}
	return b.Flush()
}

// FormatFromExtension maps a filename extension to an encoder name. It does not
// read the filename or infer formats from content.
func FormatFromExtension(extension string) string {
	e := strings.ToLower(strings.TrimPrefix(extension, "."))
	switch e {
	case "jpg":
		return "jpeg"
	case "txt":
		return "ascii"
	default:
		return e
	}
}

// GIF palette training uses the same binary alpha rule as its emitted pixels.
// Invisible RGB must not bias the palette chosen for the visible photograph.
type gifPaletteImage struct {
	image.Image
	read func(int, int) color.NRGBA
}

func (im *gifPaletteImage) ColorModel() color.Model { return color.NRGBAModel }
func (im *gifPaletteImage) NRGBAAt(x, y int) color.NRGBA {
	c := im.read(x, y)
	if c.A < 128 {
		return color.NRGBA{}
	}
	c.A = 255
	return c
}
func (im *gifPaletteImage) At(x, y int) color.Color { return im.NRGBAAt(x, y) }
