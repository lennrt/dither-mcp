package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"

	"github.com/lennrt/dither-mcp/engine"
)

const StudioMaxDimension = 1024
const StudioDefaultDimension = 512
const StudioMaxPNGBytes = 2 << 20

// StudioRequest describes an in-memory preview. It has no output path, so a
// preview call cannot publish an artifact. The returned recipe records its size.
type StudioRequest struct {
	Input     string        `json:"input" jsonschema:"Read a local still image relative to the workspace. GIF uses the first frame."`
	Palette   string        `json:"palette,omitempty" jsonschema:"Choose a built-in palette ID instead of colors. Omission selects mono."`
	Colors    []string      `json:"colors,omitempty" jsonschema:"Supply 2 through 256 unique opaque hex colors instead of palette."`
	Options   engine.Config `json:"options,omitempty" jsonschema:"Preview with these engine settings. Width and height, including inferred dimensions, must be at most 1024. Omit both to fit within 512 by 512 without enlargement. Seed must be a JavaScript-safe integer."`
	MaskInput string        `json:"mask_input,omitempty" jsonschema:"Read an optional local image mask with the same normalization and selection rules as dither_render."`
}

type StudioMetadata struct {
	Path         string             `json:"path"`
	Width        int                `json:"width"`
	Height       int                `json:"height"`
	MIMEType     string             `json:"mime_type"`
	Recipe       Recipe             `json:"recipe" jsonschema:"Replay this exact preview recipe with dither_render. Its dimensions describe the preview, not a full-resolution export."`
	MaskInput    string             `json:"mask_input,omitempty"`
	SourceSHA256 string             `json:"source_sha256" jsonschema:"SHA256 of the exact source file bytes decoded for this preview. Pass it as expected_source_sha256 when exporting."`
	MaskSHA256   string             `json:"mask_sha256,omitempty" jsonschema:"SHA256 of the exact mask file bytes decoded for this preview. Present when mask_input is set. Pass it as expected_mask_sha256 when exporting."`
	SourceWidth  int                `json:"source_width" jsonschema:"Width of the upright normalized source after the accepted crop and before preview resizing. Use this width for a source-size export."`
	SourceHeight int                `json:"source_height" jsonschema:"Height of the upright normalized source after the accepted crop and before preview resizing. Use this height for a source-size export."`
	ExportLimits StudioExportLimits `json:"export_limits" jsonschema:"The engine enforces these output dimensions and total pixel limit on exports."`
}

type StudioExportLimits struct {
	MaxWidth  int `json:"max_width"`
	MaxHeight int `json:"max_height"`
	MaxPixels int `json:"max_pixels"`
}

type StudioResult struct {
	StudioMetadata
	Data string `json:"data"`
}

func (r StudioResult) Request() StudioRequest {
	return StudioRequest{Input: r.Path, Palette: r.Recipe.Palette, Colors: r.Recipe.Colors, Options: r.Recipe.Options, MaskInput: r.MaskInput}
}

func (s *Service) studio(ctx context.Context, q StudioRequest) (StudioResult, error) {
	if q.Options.Width < 0 || q.Options.Width > StudioMaxDimension || q.Options.Height < 0 || q.Options.Height > StudioMaxDimension {
		return StudioResult{}, errors.New("studio width and height must be 0..1024. Use dither_render for larger output")
	}
	const maxSafeInteger = 1<<53 - 1
	if q.Options.Seed < -maxSafeInteger || q.Options.Seed > maxSafeInteger {
		return StudioResult{}, errors.New("studio seed must be an integer from -9007199254740991 through 9007199254740991")
	}
	cfg, recipe, maskSHA256, err := s.configWithMaskDigest(ctx, RenderRequest{Input: q.Input, Palette: q.Palette, Colors: q.Colors, Options: q.Options, MaskInput: q.MaskInput})
	if err != nil {
		return StudioResult{}, err
	}
	if err := engine.Validate(cfg); err != nil {
		return StudioResult{}, err
	}
	im, source, _, err := s.decode(ctx, q.Input)
	if err != nil {
		return StudioResult{}, err
	}
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	if crop := cfg.Crop; crop != nil {
		if crop.X >= w || crop.Y >= h || crop.Width > w-crop.X || crop.Height > h-crop.Y {
			return StudioResult{}, errors.New("crop must fit wholly inside the source image")
		}
		w, h = crop.Width, crop.Height
	}
	if cfg.Width == 0 && cfg.Height == 0 {
		scale := math.Min(1, float64(StudioDefaultDimension)/float64(max(w, h)))
		cfg.Width = max(1, int(math.Round(float64(w)*scale)))
		cfg.Height = max(1, int(math.Round(float64(h)*scale)))
	} else if cfg.Width == 0 {
		cfg.Width = max(1, int(math.Round(float64(w)*float64(cfg.Height)/float64(h))))
	} else if cfg.Height == 0 {
		cfg.Height = max(1, int(math.Round(float64(h)*float64(cfg.Width)/float64(w))))
	}
	if cfg.Width > StudioMaxDimension || cfg.Height > StudioMaxDimension {
		return StudioResult{}, fmt.Errorf("studio preview would be %dx%d. Choose a smaller width or height to fit within 1024 by 1024", cfg.Width, cfg.Height)
	}
	// Store the actual preview dimensions so Save can replay the same settings.
	recipe.Options.Width, recipe.Options.Height = cfg.Width, cfg.Height
	out, err := engine.Process(ctx, im, cfg)
	if err != nil {
		return StudioResult{}, err
	}
	buf := &boundedBuffer{max: StudioMaxPNGBytes}
	if err := engine.Encode("png", buf, out); err != nil {
		return StudioResult{}, fmt.Errorf("studio PNG preview: %w. Choose smaller dimensions", err)
	}
	if err := ctx.Err(); err != nil {
		return StudioResult{}, err
	}
	return StudioResult{
		StudioMetadata: StudioMetadata{
			Path: q.Input, Width: cfg.Width, Height: cfg.Height,
			MIMEType: "image/png", Recipe: recipe, MaskInput: q.MaskInput,
			SourceSHA256: digest(source), MaskSHA256: maskSHA256,
			SourceWidth: w, SourceHeight: h,
			ExportLimits: StudioExportLimits{MaxWidth: engine.MaxDimension, MaxHeight: engine.MaxDimension, MaxPixels: engine.MaxPixels},
		},
		Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}
