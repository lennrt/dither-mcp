package engine

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
)

// Palettes returns independent copies of the built-in palette catalog.
func Palettes() []Palette {
	out := make([]Palette, len(presets))
	for i, p := range presets {
		out[i] = p
		out[i].Colors = append([]string(nil), p.Colors...)
		out[i].Tags = append([]string(nil), p.Tags...)
	}
	return out
}

// ParseHex accepts #RGB, RGB, #RRGGBB, or RRGGBB. Palette colors are opaque.
func ParseHex(s string) (color.NRGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.NRGBA{}, fmt.Errorf("invalid palette color %q: expected #RGB or #RRGGBB", s)
	}
	v, err := strconv.ParseUint(s, 16, 24)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid palette color %q", s)
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

// Hex returns an opaque color as a lowercase six-digit RGB hexadecimal string.
func Hex(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// ResolvePalette resolves a preset or custom colors. A request must specify
// either a name or custom colors. It rejects duplicate custom colors to keep
// recipe intent and effective palette size explicit. Empty arguments select mono.
func ResolvePalette(name string, hex []string) ([]color.NRGBA, error) {
	if len(hex) > 0 && name != "" {
		return nil, fmt.Errorf("choose a palette name or custom colors, not both")
	}
	if len(hex) == 0 {
		if name == "" {
			name = "mono"
		}
		found := false
		for _, p := range presets {
			if p.ID == name {
				hex = p.Colors
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown palette %q", name)
		}
	}
	if len(hex) < 2 || len(hex) > 256 {
		return nil, fmt.Errorf("palette must have 2–256 colors")
	}
	out := make([]color.NRGBA, len(hex))
	seen := map[color.NRGBA]bool{}
	for i, s := range hex {
		c, err := ParseHex(s)
		if err != nil {
			return nil, err
		}
		if seen[c] {
			return nil, fmt.Errorf("duplicate palette color %s", Hex(c))
		}
		seen[c] = true
		out[i] = c
	}
	return out, nil
}

type histogramColor struct {
	c color.NRGBA
	colorTotals
}
type colorTotals struct {
	weight int64
	sums   [3]int64
}

func (total *colorTotals) add(c color.NRGBA, weight int64) {
	total.weight += weight
	total.sums[0] += int64(c.R) * weight
	total.sums[1] += int64(c.G) * weight
	total.sums[2] += int64(c.B) * weight
}

type colorBox struct {
	colors []histogramColor
	weight int64
	ranges [3]int
}

func newBox(cs []histogramColor) colorBox {
	b := colorBox{colors: cs}
	lo := [3]int{255, 255, 255}
	hi := [3]int{}
	for _, p := range cs {
		b.weight += p.weight
		a := [3]int{int(p.c.R), int(p.c.G), int(p.c.B)}
		for k := 0; k < 3; k++ {
			lo[k] = min(lo[k], a[k])
			hi[k] = max(hi[k], a[k])
		}
	}
	for k := 0; k < 3; k++ {
		b.ranges[k] = hi[k] - lo[k]
	}
	return b
}

const maxPaletteHistogram = 262144

func paletteBin(c color.NRGBA) int {
	return int(c.R>>2)<<12 | int(c.G>>2)<<6 | int(c.B>>2)
}

// paletteHistogram visits every pixel, retaining exact RGB counts until the
// histogram limit. On overflow it folds once into 64^3 bins, carrying original
// weighted RGB sums through both the fold and the final median-cut centroids.
func paletteHistogram(ctx context.Context, img image.Image) ([]histogramColor, error) {
	bounds := img.Bounds()
	read := pixelReader(img)
	exact := make(map[uint32]int64)
	var bins []colorTotals
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := read(x, y)
			if c.A == 0 {
				continue
			}
			if bins == nil {
				key := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
				if weight, exists := exact[key]; exists || len(exact) < maxPaletteHistogram {
					exact[key] = weight + int64(c.A)
					continue
				}
				// The dense bins need 8 MiB only after exact colors overflow;
				// ordinary low-color images keep the small exact map.
				bins = make([]colorTotals, maxPaletteHistogram)
				i := 0
				for rgb, weight := range exact {
					if i%1024 == 0 {
						if err := ctx.Err(); err != nil {
							return nil, err
						}
					}
					i++
					c := color.NRGBA{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb), 255}
					bins[paletteBin(c)].add(c, weight)
				}
				exact = nil
			}
			bins[paletteBin(c)].add(c, int64(c.A))
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if bins == nil {
		keys := make([]uint32, 0, len(exact))
		for k := range exact {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		colors := make([]histogramColor, 0, len(keys))
		for i, k := range keys {
			if i%1024 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			c := color.NRGBA{uint8(k >> 16), uint8(k >> 8), uint8(k), 255}
			var total colorTotals
			total.add(c, exact[k])
			colors = append(colors, histogramColor{c, total})
		}
		return colors, nil
	}
	count := 0
	for _, total := range bins {
		if total.weight != 0 {
			count++
		}
	}
	colors := make([]histogramColor, 0, count)
	for i, total := range bins {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if total.weight == 0 {
			continue
		}
		c := color.NRGBA{
			uint8((total.sums[0] + total.weight/2) / total.weight),
			uint8((total.sums[1] + total.weight/2) / total.weight),
			uint8((total.sums[2] + total.weight/2) / total.weight),
			255,
		}
		colors = append(colors, histogramColor{c, total})
	}
	return colors, nil
}

// ExtractPalette uses deterministic, alpha-weighted median cut. Every visible
// pixel contributes. It retains at most 262,144 histogram entries, merging into
// six-bit RGB bins only if the image exceeds that many distinct visible colors.
// Fully transparent pixels never contribute to the palette. Extraction checks
// cancellation per row and while folding a full histogram.
// It returns up to count distinct opaque colors, sorted by luminance and then RGB.
func ExtractPalette(ctx context.Context, img image.Image, count int) ([]color.NRGBA, error) {
	if count < 2 || count > 256 {
		return nil, fmt.Errorf("palette count must be 2–256")
	}
	if err := checkImage(img); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	colors, err := paletteHistogram(ctx, img)
	if err != nil {
		return nil, err
	}
	if len(colors) == 0 {
		return nil, fmt.Errorf("cannot extract a palette from a fully transparent image")
	}
	boxes := []colorBox{newBox(colors)}
	for len(boxes) < count {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		best := -1
		score := int64(-1)
		for i, b := range boxes {
			if len(b.colors) < 2 {
				continue
			}
			r := max(b.ranges[0], b.ranges[1], b.ranges[2])
			s := int64(r) * b.weight
			if s > score {
				best = i
				score = s
			}
		}
		if best < 0 {
			break
		}
		b := boxes[best]
		axis := 0
		for k := 1; k < 3; k++ {
			if b.ranges[k] > b.ranges[axis] {
				axis = k
			}
		}
		component := func(c color.NRGBA) uint8 {
			if axis == 0 {
				return c.R
			}
			if axis == 1 {
				return c.G
			}
			return c.B
		}
		sort.SliceStable(b.colors, func(i, j int) bool { return component(b.colors[i].c) < component(b.colors[j].c) })
		var total int64
		split := 1
		for i, p := range b.colors[:len(b.colors)-1] {
			total += p.weight
			split = i + 1
			if total*2 >= b.weight {
				break
			}
		}
		boxes[best] = newBox(b.colors[:split])
		boxes = append(boxes, newBox(b.colors[split:]))
	}
	out := make([]color.NRGBA, 0, len(boxes))
	seen := map[color.NRGBA]bool{}
	for _, b := range boxes {
		var sums [3]int64
		for _, p := range b.colors {
			for channel := range sums {
				sums[channel] += p.sums[channel]
			}
		}
		weight := b.weight
		c := color.NRGBA{uint8((sums[0] + weight/2) / weight), uint8((sums[1] + weight/2) / weight), uint8((sums[2] + weight/2) / weight), 255}
		if !seen[c] {
			out = append(out, c)
			seen[c] = true
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		la, lb := 299*int(a.R)+587*int(a.G)+114*int(a.B), 299*int(b.R)+587*int(b.G)+114*int(b.B)
		if la != lb {
			return la < lb
		}
		return uint32(a.R)<<16|uint32(a.G)<<8|uint32(a.B) < uint32(b.R)<<16|uint32(b.G)<<8|uint32(b.B)
	})
	return out, nil
}
