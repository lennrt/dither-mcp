package engine

import (
	"context"
	"image"
	"math"
	"sort"
	"strconv"
	"strings"
)

// splitmix is a specified integer hash, independent of Go's random algorithms.
func splitmix(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}
func coordinateNoise(x, y int, seed int64) float64 {
	v := splitmix(uint64(seed) ^ uint64(x)*0xd6e8feb86659fd93 ^ uint64(y)*0xa5a3564e27f886f7)
	return (float64(v>>11) + .5) / 9007199254740992
}
func fract(v float64) float64 { return v - math.Floor(v) }

// bayer returns [0,n*n) ranks for recursive power-of-two dispersed-dot tiles.
func bayer(n, x, y int) int {
	value := 0
	for bit := 1; bit < n; bit *= 2 {
		a, b := 0, 0
		if x&bit != 0 {
			a = 1
		}
		if y&bit != 0 {
			b = 1
		}
		value = value*4 + ((a^b)*2 + b)
	}
	return value
}

type thresholdTile struct {
	size   int
	values []float64
}

func (t thresholdTile) at(x, y int) float64 { return t.values[(y%t.size)*t.size+x%t.size] }
func rankedTile(n int, score func(int, int) float64) thresholdTile {
	type point struct {
		index int
		score float64
	}
	points := make([]point, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := y*n + x
			points[i] = point{i, score(x, y)}
		}
	}
	sort.SliceStable(points, func(i, j int) bool { return points[i].score < points[j].score })
	t := thresholdTile{n, make([]float64, n*n)}
	for rank, p := range points {
		t.values[p.index] = (float64(rank) + .5) / float64(n*n)
	}
	return t
}
func explicitTile(n int, divisor float64, values ...int) thresholdTile {
	t := thresholdTile{n, make([]float64, len(values))}
	for i, v := range values {
		t.values[i] = float64(v) / divisor
	}
	return t
}

func screenTile(id string) thresholdTile {
	switch id {
	case "clustered-4":
		return explicitTile(4, 9, 4, 2, 7, 5, 3, 1, 8, 6, 7, 5, 4, 2, 8, 6, 3, 1)
	case "clustered-8":
		return explicitTile(8, 33, 13, 7, 8, 14, 17, 21, 22, 18, 6, 1, 3, 9, 28, 31, 29, 23, 5, 2, 4, 10, 27, 32, 30, 24, 16, 12, 11, 15, 20, 26, 25, 19, 17, 21, 22, 18, 13, 7, 8, 14, 28, 31, 29, 23, 6, 1, 3, 9, 27, 32, 30, 24, 5, 2, 4, 10, 20, 26, 25, 19, 16, 12, 11, 15)
	case "halftone-4", "halftone-8", "halftone-16":
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "halftone-"))
		return rankedTile(n, func(x, y int) float64 {
			dx, dy := float64(x)+.5-float64(n)/2, float64(y)+.5-float64(n)/2
			return dx*dx + dy*dy
		})
	case "crosshatch":
		return rankedTile(8, func(x, y int) float64 { a, b := (x+y)%8, (x-y+8)%8; return float64(min(a, 8-a, b, 8-b)) })
	case "diamond":
		return rankedTile(8, func(x, y int) float64 { return math.Abs(float64(x)+.5-4) + math.Abs(float64(y)+.5-4) })
	case "spiral":
		return rankedTile(16, func(x, y int) float64 {
			dx, dy := float64(x)+.5-8, float64(y)+.5-8
			angle := math.Atan2(dy, dx)
			return fract(math.Hypot(dx, dy)*.18 - angle/(2*math.Pi))
		})
	case "dots":
		return rankedTile(12, func(x, y int) float64 {
			dx, dy := float64(x%4)+.5-2, float64(y%4)+.5-2
			return dx*dx + dy*dy + .01*float64(bayer(4, x/4, y/4))
		})
	case "waves":
		return rankedTile(16, func(x, y int) float64 {
			return math.Abs(math.Sin(2*math.Pi*float64(y)/8 + .8*math.Sin(2*math.Pi*float64(x)/16)))
		})
	}
	return thresholdTile{}
}

func ordered(ctx context.Context, img *image.NRGBA, q quantizer, cfg Config, id string, active func(int, int) bool) error {
	tile := screenTile(id)
	if id == "blue-noise" {
		var err error
		tile, err = blueNoiseTile(ctx, cfg.Seed)
		if err != nil {
			return err
		}
	}
	n := 0
	if strings.HasPrefix(id, "bayer-") {
		n, _ = strconv.Atoi(strings.TrimPrefix(id, "bayer-"))
	}
	strength := scalar(cfg.Strength, 1)
	bias := .5 - scalar(cfg.Threshold, .5)
	for y := 0; y < img.Rect.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := 0; x < img.Rect.Dx(); x++ {
			if !active(x, y) {
				continue
			}
			c := img.NRGBAAt(x, y)
			v := q.vector(c)
			for k := range v {
				v[k] = clamp(v[k] + bias)
			}
			index := 0
			if id == "threshold" {
				index = q.nearest(v)
			} else {
				t := .5
				switch {
				case n > 0:
					t = (float64(bayer(n, x%n, y%n)) + .5) / float64(n*n)
				case tile.size > 0:
					t = tile.at(x, y)
				default:
					switch id {
					case "random":
						t = coordinateNoise(x, y, cfg.Seed)
					case "interleaved-gradient":
						t = fract(52.9829189 * fract(.06711056*float64(x)+.00583715*float64(y)+float64(uint64(cfg.Seed)&0xffff)/65536))
					case "arithmetic-add":
						t = (float64(((x+y*237)*119)&255) + .5) / 256
					case "arithmetic-xor":
						t = (float64(((x^(y*149))*123)&255) + .5) / 256
					case "horizontal-lines":
						t = (float64(y%8) + .5) / 8
					case "vertical-lines":
						t = (float64(x%8) + .5) / 8
					case "diagonal-lines":
						t = (float64((x+y)%8) + .5) / 8
					case "checkerboard":
						t = 1.0 / 3
						if (x+y)%2 == 1 {
							t = 2.0 / 3
						}
					}
				}
				index = q.pair(v, t, strength)
			}
			chosen := q.colors[index]
			chosen.A = c.A
			img.SetNRGBA(x, y, chosen)
		}
	}
	return nil
}

// blueNoiseTile implements Ulichney's void-and-cluster construction on a small
// periodic tile. The Gaussian density uses wrapped distance (sigma=1.5).
// Three ranking phases prevent the bright-end clumping of progressive point insertion.
// Work and storage remain fixed. Each recipe regenerates the tile to avoid an
// unbounded cache of seeds in long-running MCP servers.
func blueNoiseTile(ctx context.Context, seed int64) (thresholdTile, error) {
	const n = 16
	const total = n * n
	influence := make([]float64, total*total)
	for a := 0; a < total; a++ {
		if a%32 == 0 {
			if err := ctx.Err(); err != nil {
				return thresholdTile{}, err
			}
		}
		for b := 0; b < total; b++ {
			dx, dy := abs(a%n-b%n), abs(a/n-b/n)
			dx = min(dx, n-dx)
			dy = min(dy, n-dy)
			influence[a*total+b] = math.Exp(-float64(dx*dx+dy*dy) / (2 * 1.5 * 1.5))
		}
	}
	occupied := make([]bool, total)
	density := make([]float64, total)
	toggle := func(index int, add bool) {
		occupied[index] = add
		sign := -1.0
		if add {
			sign = 1
		}
		for j := 0; j < total; j++ {
			density[j] += sign * influence[index*total+j]
		}
	}
	selectPixel := func(want bool, largest bool) int {
		best := -1
		score := math.Inf(1)
		if largest {
			score = math.Inf(-1)
		}
		for i, on := range occupied {
			if on != want {
				continue
			}
			if best < 0 || (largest && density[i] > score) || (!largest && density[i] < score) {
				best = i
				score = density[i]
			}
		}
		return best
	}
	indices := make([]int, total)
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		a, b := splitmix(uint64(seed)^uint64(indices[i])), splitmix(uint64(seed)^uint64(indices[j]))
		if a == b {
			return indices[i] < indices[j]
		}
		return a < b
	})
	const initial = total / 10
	for _, i := range indices[:initial] {
		toggle(i, true)
	}
	for iteration := 0; iteration < 4096; iteration++ {
		if iteration%64 == 0 {
			if err := ctx.Err(); err != nil {
				return thresholdTile{}, err
			}
		}
		cluster := selectPixel(true, true)
		toggle(cluster, false)
		void := selectPixel(false, false)
		toggle(void, true)
		if void == cluster {
			break
		}
	}
	prototype := append([]bool(nil), occupied...)
	ranks := make([]int, total)
	for rank := initial - 1; rank >= 0; rank-- {
		cluster := selectPixel(true, true)
		ranks[cluster] = rank
		toggle(cluster, false)
	}
	// Restore the optimized prototype. Fill its voids up to half occupancy.
	clear(density)
	clear(occupied)
	for i, on := range prototype {
		if on {
			toggle(i, true)
		}
	}
	for rank := initial; rank < total/2; rank++ {
		void := selectPixel(false, false)
		ranks[void] = rank
		toggle(void, true)
	}
	// Rank the bright half by removing clusters from the inverse pattern.
	inverse := append([]bool(nil), occupied...)
	clear(density)
	clear(occupied)
	for i, on := range inverse {
		if !on {
			toggle(i, true)
		}
	}
	for rank := total / 2; rank < total; rank++ {
		cluster := selectPixel(true, true)
		ranks[cluster] = rank
		toggle(cluster, false)
	}
	tile := thresholdTile{n, make([]float64, total)}
	for i, r := range ranks {
		tile.values[i] = (float64(r) + .5) / total
	}
	return tile, nil
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
