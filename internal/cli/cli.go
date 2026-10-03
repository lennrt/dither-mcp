// Package cli implements a scriptable command-line interface to the MCP service.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/lennrt/dither-mcp/internal/mcpserver"
	"github.com/mark3labs/mcp-go/server"
)

const Help = `dither-mcp — your image studio, available to your agent

Usage: dither-mcp <command> [flags]

  mcp             Serve MCP over stdin/stdout (JSON-RPC only)
  catalog         Discover algorithms, formats, defaults, and limits
  palettes        Search palettes by style, category, and color count
  inspect         Inspect a local image before processing
  render          Dither one image to a new output file
  compare         Make an algorithm comparison contact sheet
  animate         Animate a still image or process an existing GIF
  video           Process a bounded local video (requires --allow-video)
  separate        Export spot-color PNG plates in a ZIP
  call            Call any tool: call [flags] dither_TOOL '{"input":"..."}'
  version         Print the development version
  help            Show this guide

Common flags (after command):
  --root .                 Workspace directory for all relative paths
  --input path             Local source file
  --output path            New artifact path
  --algorithm ID           Use catalog for exact IDs
  --palette NAME           Named palette (default mono)
  --colors '#112233,#fff'   Custom 2..256 colors, instead of palette
  --width N --height N     Resize, preserving aspect when one is omitted
  --pixel-scale N          Coarse grid with nearest-neighbor enlargement
  --seed N                 Deterministic noise seed
  --serpentine             Alternate diffusion scan direction
  --brightness N           Offset -1..1
  --contrast N --gamma N --saturation N --strength N --threshold N
  --invert --grayscale      Tone controls
  --format png             Override extension-based export format
  --dpi 300                PNG print-resolution metadata
  --recipe path            Saved recipe without inline palette/options
  --options '{...}'        Advanced engine options (crop, mask, effects)
  --mask-input path        Luminance/alpha mask image
  --algorithms A,B,C        Comparison algorithms
  --columns N              Contact-sheet or sprite-sheet columns
  --effect wave            source, wave, orbit, pulse, noise, palette-cycle
  --frames 24 --fps 12      Animation/video budget
  --amplitude 0.15         Motion strength 0..1 (zero selects default)
  --start 0                Video segment start in seconds
  --allow-video            Explicitly enable local ffmpeg/ffprobe

Palette discovery flags (palettes command):
  --query 'ocean night'     Case-insensitive AND search across metadata/colors
  --category ocean         Category slug from the discovery result
  --min-colors 2 --max-colors 8
  --limit 32 --offset 0     Stable ID ordering with at most 256 results

The CLI writes JSON results to stdout and diagnostics to stderr. Each output uses a new path.
Still images and masks normalize EXIF orientation and supported color metadata to upright sRGB.
Crop coordinates use the upright image. Run inspect to see normalization details.
For advanced tool schemas, connect an MCP client or read docs/mcp.md.
`

func Run(ctx context.Context, args []string, in io.Reader, out, errout io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, Help)
		return err
	}
	cmd := args[0]
	if cmd == "version" {
		_, err := fmt.Fprintln(out, "dither-mcp "+app.Version)
		return err
	}
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	f.SetOutput(errout)
	f.Usage = func() { fmt.Fprint(errout, Help) }
	var q app.RenderRequest
	var paletteQuery app.PaletteQuery
	var root, colors, algorithms, options, effect string
	var allow bool
	var frames, fps, columns int
	var amplitude, start float64
	f.StringVar(&root, "root", ".", "workspace directory")
	f.BoolVar(&allow, "allow-video", false, "enable ffmpeg")
	if cmd == "palettes" {
		f.StringVar(&paletteQuery.Query, "query", "", "palette search terms")
		f.StringVar(&paletteQuery.Category, "category", "", "palette category")
		f.IntVar(&paletteQuery.MinColors, "min-colors", 0, "minimum palette colors")
		f.IntVar(&paletteQuery.MaxColors, "max-colors", 0, "maximum palette colors")
		f.IntVar(&paletteQuery.Limit, "limit", 0, "page size (default 32, maximum 256)")
		f.IntVar(&paletteQuery.Offset, "offset", 0, "page offset")
	}
	f.StringVar(&q.Input, "input", "", "relative source path")
	f.StringVar(&q.Output, "output", "", "new relative output path")
	f.StringVar(&q.Recipe, "recipe", "", "relative recipe path")
	f.StringVar(&q.Palette, "palette", "", "palette name")
	f.StringVar(&colors, "colors", "", "comma-separated colors")
	f.StringVar(&q.Options.Algorithm, "algorithm", "", "algorithm ID")
	f.IntVar(&q.Options.Width, "width", 0, "width")
	f.IntVar(&q.Options.Height, "height", 0, "height")
	f.IntVar(&q.Options.PixelScale, "pixel-scale", 0, "pixel scale")
	f.Int64Var(&q.Options.Seed, "seed", 0, "seed")
	f.BoolVar(&q.Options.Serpentine, "serpentine", false, "serpentine scanning")
	f.BoolVar(&q.Options.Invert, "invert", false, "invert")
	f.BoolVar(&q.Options.Grayscale, "grayscale", false, "grayscale")
	f.Float64Var(&q.Options.Brightness, "brightness", 0, "brightness offset")
	f.StringVar(&q.Format, "format", "", "format")
	f.IntVar(&q.DPI, "dpi", 0, "PNG DPI")
	f.StringVar(&q.MaskInput, "mask-input", "", "mask image")
	f.StringVar(&options, "options", "", "advanced options JSON")
	f.StringVar(&algorithms, "algorithms", "floyd-steinberg,atkinson,bayer-8", "comparison IDs")
	f.StringVar(&effect, "effect", "", "motion effect")
	f.IntVar(&frames, "frames", 0, "frame count")
	f.IntVar(&fps, "fps", 0, "frame rate")
	f.IntVar(&columns, "columns", 0, "sheet columns")
	f.Float64Var(&amplitude, "amplitude", 0, "motion amplitude")
	f.Float64Var(&start, "start", 0, "video start seconds")
	for _, pair := range []struct {
		name string
		p    **float64
	}{{"contrast", &q.Options.Contrast}, {"gamma", &q.Options.Gamma}, {"saturation", &q.Options.Saturation}, {"strength", &q.Options.Strength}, {"threshold", &q.Options.Threshold}} {
		pair := pair
		f.Func(pair.name, "optional adjustment", func(s string) error {
			v, e := strconv.ParseFloat(s, 64)
			if e == nil {
				*pair.p = &v
			}
			return e
		})
	}
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if colors != "" {
		for _, c := range strings.Split(colors, ",") {
			q.Colors = append(q.Colors, strings.TrimSpace(c))
		}
	}
	if options != "" {
		if err := app.StrictJSON([]byte(options), &q.Options); err != nil {
			return err
		}
	}
	if cmd != "call" && f.NArg() != 0 {
		return errors.New("unexpected positional arguments. Put flags after the command")
	}
	svc, err := app.New(root, allow)
	if err != nil {
		return err
	}
	defer svc.Close()
	if cmd == "mcp" {
		return server.NewStdioServer(mcpserver.New(svc)).Listen(ctx, &lineLimitReader{source: in}, out)
	}
	var name string
	var request any
	switch cmd {
	case "catalog":
		name = "dither_catalog"
		request = app.Empty{}
	case "palettes":
		name = "dither_palettes"
		request = paletteQuery
	case "inspect":
		name = "dither_inspect"
		request = app.InputRequest{Input: q.Input}
	case "render", "separate":
		name = "dither_" + cmd
		request = q
	case "compare":
		name = "dither_compare"
		request = app.CompareRequest{RenderRequest: q, Algorithms: strings.Split(algorithms, ","), Columns: columns}
	case "animate":
		name = "dither_animate"
		request = app.AnimateRequest{RenderRequest: q, Effect: effect, Frames: frames, FPS: fps, Amplitude: amplitude, Columns: columns}
	case "video":
		name = "dither_video"
		request = app.VideoRequest{RenderRequest: q, Frames: frames, FPS: fps, Start: start}
	case "call":
		if f.NArg() != 2 {
			return errors.New("usage: call [--root DIR] TOOL '{JSON}'")
		}
		name = f.Arg(0)
		request = json.RawMessage(f.Arg(1))
	default:
		return fmt.Errorf("unknown command %q. Run dither-mcp help", cmd)
	}
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	result, err := svc.Do(ctx, name, b)
	if err != nil {
		return err
	}
	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	return e.Encode(result)
}
