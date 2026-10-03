package engine

import (
	"context"
	"image"
	"math"
)

type diffusionWeight struct {
	x, y   int
	weight float64
}
type diffusionKernel []diffusionWeight

func kernel(divisor float64, entries ...[3]int) diffusionKernel {
	out := make(diffusionKernel, len(entries))
	for i, e := range entries {
		out[i] = diffusionWeight{e[0], e[1], float64(e[2]) / divisor}
	}
	return out
}

// Kernels use explicit weights. Atkinson propagates 6/8 of the error, so its
// weights retain their original sum. SOURCES.md identifies sources and conventions.
var kernels = map[string]diffusionKernel{
	"floyd-steinberg":       kernel(16, [3]int{1, 0, 7}, [3]int{-1, 1, 3}, [3]int{0, 1, 5}, [3]int{1, 1, 1}),
	"false-floyd-steinberg": kernel(8, [3]int{1, 0, 3}, [3]int{0, 1, 3}, [3]int{1, 1, 2}),
	"jarvis-judice-ninke":   kernel(48, [3]int{1, 0, 7}, [3]int{2, 0, 5}, [3]int{-2, 1, 3}, [3]int{-1, 1, 5}, [3]int{0, 1, 7}, [3]int{1, 1, 5}, [3]int{2, 1, 3}, [3]int{-2, 2, 1}, [3]int{-1, 2, 3}, [3]int{0, 2, 5}, [3]int{1, 2, 3}, [3]int{2, 2, 1}),
	"atkinson":              kernel(8, [3]int{1, 0, 1}, [3]int{2, 0, 1}, [3]int{-1, 1, 1}, [3]int{0, 1, 1}, [3]int{1, 1, 1}, [3]int{0, 2, 1}),
	"stucki":                kernel(42, [3]int{1, 0, 8}, [3]int{2, 0, 4}, [3]int{-2, 1, 2}, [3]int{-1, 1, 4}, [3]int{0, 1, 8}, [3]int{1, 1, 4}, [3]int{2, 1, 2}, [3]int{-2, 2, 1}, [3]int{-1, 2, 2}, [3]int{0, 2, 4}, [3]int{1, 2, 2}, [3]int{2, 2, 1}),
	"burkes":                kernel(32, [3]int{1, 0, 8}, [3]int{2, 0, 4}, [3]int{-2, 1, 2}, [3]int{-1, 1, 4}, [3]int{0, 1, 8}, [3]int{1, 1, 4}, [3]int{2, 1, 2}),
	"sierra":                kernel(32, [3]int{1, 0, 5}, [3]int{2, 0, 3}, [3]int{-2, 1, 2}, [3]int{-1, 1, 4}, [3]int{0, 1, 5}, [3]int{1, 1, 4}, [3]int{2, 1, 2}, [3]int{-1, 2, 2}, [3]int{0, 2, 3}, [3]int{1, 2, 2}),
	"two-row-sierra":        kernel(16, [3]int{1, 0, 4}, [3]int{2, 0, 3}, [3]int{-2, 1, 1}, [3]int{-1, 1, 2}, [3]int{0, 1, 3}, [3]int{1, 1, 2}, [3]int{2, 1, 1}),
	"sierra-lite":           kernel(4, [3]int{1, 0, 2}, [3]int{-1, 1, 1}, [3]int{0, 1, 1}),
	"simple-2d":             kernel(2, [3]int{1, 0, 1}, [3]int{0, 1, 1}),
	"steven-pigeon":         kernel(14, [3]int{1, 0, 2}, [3]int{2, 0, 1}, [3]int{-1, 1, 2}, [3]int{0, 1, 2}, [3]int{1, 1, 2}, [3]int{-2, 2, 1}, [3]int{0, 2, 1}, [3]int{2, 2, 1}),
	"fan":                   kernel(16, [3]int{1, 0, 7}, [3]int{-2, 1, 1}, [3]int{-1, 1, 3}, [3]int{0, 1, 5}),
	"shiau-fan":             kernel(8, [3]int{1, 0, 4}, [3]int{-2, 1, 1}, [3]int{-1, 1, 1}, [3]int{0, 1, 2}),
	"shiau-fan-2":           kernel(16, [3]int{1, 0, 8}, [3]int{-3, 1, 1}, [3]int{-2, 1, 1}, [3]int{-1, 1, 2}, [3]int{0, 1, 4}),
	"stevenson-arce":        kernel(200, [3]int{2, 0, 32}, [3]int{-3, 1, 12}, [3]int{-1, 1, 26}, [3]int{1, 1, 30}, [3]int{3, 1, 16}, [3]int{-2, 2, 12}, [3]int{0, 2, 26}, [3]int{2, 2, 12}, [3]int{-3, 3, 5}, [3]int{-1, 3, 12}, [3]int{1, 3, 12}, [3]int{3, 3, 5}),
}

var catalog = []Algorithm{
	{"floyd-steinberg", "Floyd–Steinberg", "diffusion", "The kernel distributes error to four neighbors with 7:3:5:1 weights."},
	{"false-floyd-steinberg", "False Floyd–Steinberg", "diffusion", "The kernel distributes error to three neighbors with 3:3:2 weights."},
	{"jarvis-judice-ninke", "Jarvis–Judice–Ninke", "diffusion", "The kernel distributes error to twelve neighbors across three rows."},
	{"atkinson", "Atkinson", "diffusion", "The kernel distributes six 1/8 shares. It retains 1/4 of the quantization error."},
	{"stucki", "Stucki", "diffusion", "The kernel distributes error across three rows with a denominator of 42."},
	{"burkes", "Burkes", "diffusion", "The kernel simplifies Stucki to two rows."},
	{"sierra", "Sierra", "diffusion", "The kernel uses the full three-row Sierra distribution."},
	{"two-row-sierra", "Two-row Sierra", "diffusion", "The kernel distributes error to seven neighbors with a denominator of 16."},
	{"sierra-lite", "Sierra Lite", "diffusion", "The kernel uses the three-neighbor Sierra 2-4A distribution."},
	{"simple-2d", "Simple 2D", "diffusion", "The kernel distributes equal error shares to the right and below."},
	{"steven-pigeon", "Steven Pigeon", "diffusion", "The sparse kernel distributes error across three rows to separated distant neighbors."},
	{"fan", "Fan", "diffusion", "The kernel biases a Floyd-style distribution toward lower-left neighbors."},
	{"shiau-fan", "Shiau–Fan", "diffusion", "The kernel distributes error to four neighbors with power-of-two weights."},
	{"shiau-fan-2", "Shiau–Fan 2", "diffusion", "The kernel extends the distribution to five neighbors with power-of-two weights."},
	{"stevenson-arce", "Stevenson–Arce", "diffusion", "The sparse hexagonal kernel spans seven columns and four rows."},
	{"riemersma", "Riemersma", "curve", "Hilbert traversal uses a 16-entry error history whose weights decay exponentially."},
	{"threshold", "Threshold", "quantization", "The algorithm selects the nearest palette color without a dither pattern."},
	{"bayer-2", "Bayer 2×2", "ordered", "The recursive Bayer screen uses a 2×2 dispersed-dot tile."},
	{"bayer-4", "Bayer 4×4", "ordered", "The recursive Bayer screen uses a 4×4 dispersed-dot tile."},
	{"bayer-8", "Bayer 8×8", "ordered", "The recursive Bayer screen uses an 8×8 dispersed-dot tile."},
	{"bayer-16", "Bayer 16×16", "ordered", "The recursive Bayer screen uses a 16×16 dispersed-dot tile."},
	{"bayer-32", "Bayer 32×32", "ordered", "The recursive Bayer screen uses a 32×32 dispersed-dot tile."},
	{"clustered-4", "Clustered 4×4", "ordered", "The screen forms angled dot clusters with a 4×4 tile."},
	{"clustered-8", "Clustered 8×8", "ordered", "The screen forms angled dot clusters with an 8×8 tile."},
	{"random", "Uniform noise", "noise", "Seeded coordinate hashing produces independent uniform thresholds."},
	{"blue-noise", "Blue noise", "noise", "The seeded void-and-cluster construction produces a 16×16 tile with toroidal distance."},
	{"interleaved-gradient", "Interleaved gradient noise", "noise", "Interleaved fractional coordinates produce a deterministic low-discrepancy screen."},
	{"arithmetic-add", "Arithmetic add", "ordered", "Coordinate sums and products determine screen thresholds."},
	{"arithmetic-xor", "Arithmetic XOR", "ordered", "Bitwise coordinate operations determine screen thresholds."},
	{"halftone-4", "Dot screen 4", "screen", "The ranked screen forms circular dots in four-pixel cells."},
	{"halftone-8", "Dot screen 8", "screen", "The ranked screen forms circular dots in eight-pixel cells."},
	{"halftone-16", "Dot screen 16", "screen", "The ranked screen forms circular dots in sixteen-pixel cells."},
	{"horizontal-lines", "Horizontal lines", "screen", "The engraving screen repeats horizontal lines every eight pixels."},
	{"vertical-lines", "Vertical lines", "screen", "The engraving screen repeats vertical lines every eight pixels."},
	{"diagonal-lines", "Diagonal lines", "screen", "The engraving screen repeats 45-degree lines every eight pixels."},
	{"crosshatch", "Crosshatch", "screen", "The ranked screen forms crossed diagonal lines."},
	{"checkerboard", "Checkerboard", "ordered", "The screen alternates two threshold levels in a checkerboard pattern."},
	{"diamond", "Diamond screen", "screen", "The ranked screen forms diamond cells using Manhattan distance."},
	{"spiral", "Spiral screen", "screen", "The original ranked screen forms spiral cells using polar coordinates."},
	{"dots", "Stipple screen", "screen", "The original screen grows sparse ordered points."},
	{"waves", "Wave screen", "screen", "The original ranked screen forms sinusoidal engraving patterns."},
}

// Catalog returns a copy of the 41 implemented algorithm descriptions.
func Catalog() []Algorithm { return append([]Algorithm(nil), catalog...) }
func algorithmByID(id string) (Algorithm, bool) {
	for _, a := range catalog {
		if a.ID == id {
			return a, true
		}
	}
	return Algorithm{}, false
}

func diffuse(ctx context.Context, img *image.NRGBA, kernel diffusionKernel, q quantizer, cfg Config, active func(int, int) bool) error {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	depth := 1
	for _, e := range kernel {
		depth = max(depth, e.y+1)
	}
	rows := make([][]vec3, depth)
	for i := range rows {
		rows[i] = make([]vec3, w)
	}
	strength := scalar(cfg.Strength, 1)
	bias := .5 - scalar(cfg.Threshold, .5)
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		direction := 1
		start, end := 0, w
		if cfg.Serpentine && y%2 == 1 {
			direction = -1
			start, end = w-1, -1
		}
		current := rows[y%depth]
		for x := start; x != end; x += direction {
			if !active(x, y) {
				continue
			}
			c := img.NRGBAAt(x, y)
			v := q.vector(c).add(current[x])
			for i := range v {
				v[i] = clamp(v[i] + bias)
			}
			index := q.nearest(v)
			chosen := q.colors[index]
			chosen.A = c.A
			img.SetNRGBA(x, y, chosen)
			errorVector := v.sub(q.vectors[index]).mul(strength)
			for _, e := range kernel {
				nx, ny := x+direction*e.x, y+e.y
				if nx < 0 || nx >= w || ny >= h || !diffusionPathActive(x, y, nx, ny, active) {
					continue
				}
				row := rows[ny%depth]
				row[nx] = row[nx].add(errorVector.mul(e.weight))
			}
		}
		clear(current)
	}
	return nil
}

// hilbertXY maps a Hilbert-curve index to a coordinate in a power-of-two square.
func hilbertXY(n, index int) (int, int) {
	x, y := 0, 0
	t := index
	for s := 1; s < n; s *= 2 {
		rx := 1 & (t / 2)
		ry := 1 & (t ^ rx)
		if ry == 0 {
			if rx == 1 {
				x = s - 1 - x
				y = s - 1 - y
			}
			x, y = y, x
		}
		x += s * rx
		y += s * ry
		t /= 4
	}
	return x, y
}
func riemersma(ctx context.Context, img *image.NRGBA, q quantizer, cfg Config, active func(int, int) bool) error {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	n := 1
	for n < max(w, h) {
		n *= 2
	}
	var history [16]vec3
	var weights [16]float64
	for i := range weights {
		weights[i] = math.Pow(16, float64(i)/15-1)
	}
	head := 0
	strength := scalar(cfg.Strength, 1)
	bias := .5 - scalar(cfg.Threshold, .5)
	// Thin rectangles need only their visible part of the padded square.
	// The traversal prunes Hilbert subtrees wholly outside the image before visiting pixels.
	visited := 0
	var visit func(int, int, int, int, int, int, int) error
	visit = func(x0, y0, xi, xj, yi, yj, level int) error {
		corners := [4][2]int{{x0, y0}, {x0 + xi, y0 + xj}, {x0 + yi, y0 + yj}, {x0 + xi + yi, y0 + xj + yj}}
		minx, maxx, miny, maxy := corners[0][0], corners[0][0], corners[0][1], corners[0][1]
		for _, p := range corners {
			minx = min(minx, p[0])
			maxx = max(maxx, p[0])
			miny = min(miny, p[1])
			maxy = max(maxy, p[1])
		}
		if maxx <= 0 || maxy <= 0 || minx >= w || miny >= h {
			return nil
		}
		if level == 0 {
			x, y := x0+(xi+yi)/2, y0+(xj+yj)/2
			// Integer division truncates negative vectors toward zero. The center must
			// use mathematical floor to stay inside a reflected unit cell.
			x = x0 + int(math.Floor(float64(xi+yi)/2))
			y = y0 + int(math.Floor(float64(xj+yj)/2))
			if x < 0 || y < 0 || x >= w || y >= h {
				return nil
			}
			visited++
			if visited%256 == 1 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if !active(x, y) {
				clear(history[:])
				return nil
			}
			c := img.NRGBAAt(x, y)
			source := q.vector(c)
			v := source
			for i := 0; i < 16; i++ {
				v = v.add(history[(head+i)%16].mul(weights[i] * strength))
			}
			for i := range v {
				v[i] = clamp(v[i] + bias)
			}
			index := q.nearest(v)
			chosen := q.colors[index]
			chosen.A = c.A
			img.SetNRGBA(x, y, chosen)
			// The original algorithm records source minus chosen. Recording adjusted
			// minus chosen would cause accumulated history to amplify errors excessively.
			history[head] = source.sub(q.vectors[index])
			head = (head + 1) % 16
			return nil
		}
		if err := visit(x0, y0, yi/2, yj/2, xi/2, xj/2, level-1); err != nil {
			return err
		}
		if err := visit(x0+xi/2, y0+xj/2, xi/2, xj/2, yi/2, yj/2, level-1); err != nil {
			return err
		}
		if err := visit(x0+xi/2+yi/2, y0+xj/2+yj/2, xi/2, xj/2, yi/2, yj/2, level-1); err != nil {
			return err
		}
		return visit(x0+xi/2+yi, y0+xj/2+yj, -yi/2, -yj/2, -xi/2, -xj/2, level-1)
	}
	level := 0
	for 1<<level < n {
		level++
	}
	return visit(0, 0, n, 0, 0, n, level)
}

// Every traversed point must remain in the selected, visible region. Wider
// kernels therefore cannot jump across a one-pixel transparent or masked gap.
func diffusionPathActive(x0, y0, x1, y1 int, active func(int, int) bool) bool {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for x0 != x1 || y0 != y1 {
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
		if !active(x0, y0) {
			return false
		}
	}
	return true
}
