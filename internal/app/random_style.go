package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
)

const MaxStyleSeed int64 = 1<<53 - 1

type RandomStyleRequest struct {
	Seed *int64 `json:"seed,omitempty" jsonschema:"Choose an integer from -9007199254740991 through 9007199254740991 to reproduce a style within the current engine and catalog version. Omit seed for a new random style. Zero is a reproducible seed; null is invalid."`
}

// UnmarshalJSON distinguishes an omitted seed from null, which is not a seed.
func (q *RandomStyleRequest) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Seed json.RawMessage `json:"seed"`
	}
	if err := StrictJSON(raw, &fields); err != nil {
		return err
	}
	q.Seed = nil
	if len(fields.Seed) == 0 {
		return nil
	}
	var seed int64
	if strings.TrimSpace(string(fields.Seed)) == "null" || json.Unmarshal(fields.Seed, &seed) != nil {
		return errors.New("seed must be an integer from -9007199254740991 through 9007199254740991, or omitted")
	}
	q.Seed = &seed
	return nil
}

type RandomStyleResult struct {
	Seed   int64  `json:"seed" jsonschema:"Pass this JavaScript-safe integer back to dither_random_style to reproduce the selection within the current engine and catalog version."`
	Recipe Recipe `json:"recipe" jsonschema:"A validated version-1 style recipe with no source, dimensions, pixel scale, resize filter, crop, or mask. Apply its palette and options to dither_studio or dither_render, or save it with dither_recipe_save."`
}

// styleRandom uses the same specified SplitMix64 mixing as the image engine,
// with a local stream so concurrent callers cannot affect one another.
type styleRandom uint64

func (r *styleRandom) next() uint64 {
	*r += 0x9e3779b97f4a7c15
	v := uint64(*r)
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}

func (r *styleRandom) pick(n int) int { return int(r.next() % uint64(n)) }

func randomStyle(ctx context.Context, q RandomStyleRequest) (RandomStyleResult, error) {
	if err := ctx.Err(); err != nil {
		return RandomStyleResult{}, err
	}
	var seed int64
	if q.Seed != nil {
		seed = *q.Seed
		if seed < -MaxStyleSeed || seed > MaxStyleSeed {
			return RandomStyleResult{}, errors.New("seed must be an integer from -9007199254740991 through 9007199254740991")
		}
	} else {
		value, err := rand.Int(rand.Reader, big.NewInt(2*MaxStyleSeed+1))
		if err != nil {
			return RandomStyleResult{}, fmt.Errorf("choose random style seed: %w", err)
		}
		seed = value.Int64() - MaxStyleSeed
	}
	rng := styleRandom(uint64(seed))
	algorithms, palettes := engine.Catalog(), engine.Palettes()
	algorithm := algorithms[rng.pick(len(algorithms))]
	palette := palettes[rng.pick(len(palettes))]
	// Discrete, moderate adjustments retain recognizable subject detail.
	// Texture uses at most one subtle post-effect; geometry stays untouched.
	options := engine.Config{
		Algorithm:  algorithm.ID,
		Seed:       seed,
		Brightness: float64(rng.pick(9)-4) / 50,
		Contrast:   engine.Float(float64(90+rng.pick(10)*5) / 100),
		Gamma:      engine.Float(float64(85+rng.pick(7)*5) / 100),
		Saturation: engine.Float(float64(75+rng.pick(7)*10) / 100),
		ColorSpace: []string{"srgb", "srgb", "linear-rgb"}[rng.pick(3)],
	}
	if algorithm.ID != "threshold" {
		options.Strength = engine.Float(float64(80+rng.pick(7)*5) / 100)
	}
	if algorithm.Family == "diffusion" {
		options.Serpentine = rng.pick(2) == 0
	}
	switch rng.pick(8) {
	case 0:
		options.Effects.Scanlines = float64(8+rng.pick(6)*2) / 100
	case 1:
		options.Effects.CRT = float64(8+rng.pick(6)*2) / 100
	case 2:
		options.Effects.Noise = float64(2+rng.pick(5)) / 100
	}
	recipe := Recipe{Version: 1, Palette: palette.ID, Options: options}
	if err := validateRecipe(ctx, recipe); err != nil {
		return RandomStyleResult{}, fmt.Errorf("generated style is invalid: %w", err)
	}
	return RandomStyleResult{Seed: seed, Recipe: recipe}, nil
}
