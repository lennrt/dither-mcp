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
	c      color.NRGBA
	weight int
}
type colorBox struct {
	colors []histogramColor
	weight int
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

// ExtractPalette uses deterministic, alpha-weighted median cut. It samples at
// most 262,144 pixels on a fixed stride. It ignores fully transparent pixels.
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
	bounds := img.Bounds()
	read := pixelReader(img)
	width, height := bounds.Dx(), bounds.Dy()
	step := max(1, (width*height+262143)/262144)
	histogram := make(map[uint32]int)
	for y := 0; y < height; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		first := (step - (y*width)%step) % step
		for x := first; x < width; x += step {
			c := read(bounds.Min.X+x, bounds.Min.Y+y)
			if c.A == 0 {
				continue
			}
			key := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
			histogram[key] += int(c.A)
		}
	}
	if len(histogram) == 0 {
		return nil, fmt.Errorf("cannot extract a palette from a fully transparent image")
	}
	keys := make([]uint32, 0, len(histogram))
	for k := range histogram {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	colors := make([]histogramColor, 0, len(keys))
	for _, k := range keys {
		colors = append(colors, histogramColor{color.NRGBA{uint8(k >> 16), uint8(k >> 8), uint8(k), 255}, histogram[k]})
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
			s := int64(r) * int64(b.weight)
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
		total := 0
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
			w := int64(p.weight)
			sums[0] += int64(p.c.R) * w
			sums[1] += int64(p.c.G) * w
			sums[2] += int64(p.c.B) * w
		}
		weight := int64(b.weight)
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
