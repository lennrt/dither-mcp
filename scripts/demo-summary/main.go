// Command demo-summary formats real CLI artifact JSON for terminal recordings.
// The tapes show this filter explicitly. It reads tool results from stdin.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lennrt/dither-mcp/internal/app"
)

type artifact struct {
	Operation string `json:"operation"`
	Engine    string `json:"engine_version"`
	Path      string `json:"path"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Frames    int    `json:"frames"`
	Bytes     int    `json:"bytes"`
	SHA256    string `json:"sha256"`
	Recipe    struct {
		Version int      `json:"version"`
		Palette string   `json:"palette"`
		Colors  []string `json:"colors"`
		Options struct {
			Algorithm  string `json:"algorithm"`
			PixelScale int    `json:"pixel_scale"`
		} `json:"options"`
	} `json:"recipe"`
}

func fail(err error) { fmt.Fprintln(os.Stderr, "dither-summary:", err); os.Exit(1) }
func main() {
	palettes := flag.Bool("palettes", false, "Summarize a palette discovery response")
	inspect := flag.Bool("inspect", false, "Summarize image inspection and input normalization")
	flag.Parse()
	if flag.NArg() != 0 || *palettes && *inspect {
		fail(fmt.Errorf("use one summary mode without positional arguments"))
	}
	d := json.NewDecoder(io.LimitReader(os.Stdin, 1<<20))
	var raw json.RawMessage
	if err := d.Decode(&raw); err != nil {
		fail(err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		fail(fmt.Errorf("expected exactly one JSON result"))
	}
	if *inspect {
		var result app.Inspection
		if err := json.Unmarshal(raw, &result); err != nil {
			fail(err)
		}
		if result.Path == "" || result.Width < 1 || result.Height < 1 || result.Normalization.WorkingSpace != "srgb" {
			fail(fmt.Errorf("result does not contain complete inspection metadata"))
		}
		fmt.Printf("IMAGE  %s · %s · %d × %d upright\n", result.Path, result.Format, result.Width, result.Height)
		fmt.Printf("SOURCE %d × %d stored · EXIF orientation %d\n", result.Normalization.StoredWidth, result.Normalization.StoredHeight, result.Normalization.EXIFOrientation)
		fmt.Printf("COLOR  %s → sRGB · converted %t\n", result.Normalization.ColorSource, result.Normalization.ColorConverted)
		fmt.Printf("FILE   %d bytes · %d frame(s) · alpha %t\n", result.Bytes, result.Frames, result.HasAlpha)
		fmt.Printf("SHA256 %s\n", result.SHA256)
		return
	}
	if *palettes {
		var p struct {
			Total    int `json:"total"`
			Matched  int `json:"matched"`
			Count    int `json:"count"`
			Offset   int `json:"offset"`
			Palettes []struct {
				ID       string   `json:"id"`
				Category string   `json:"category"`
				Colors   []string `json:"colors"`
			} `json:"palettes"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			fail(err)
		}
		if p.Total < 1 || p.Count != len(p.Palettes) || p.Matched < p.Count || p.Offset < 0 {
			fail(fmt.Errorf("result does not contain complete palette discovery metadata"))
		}
		fmt.Printf("PALETTES  %d in the library · %d matching · %d shown\n", p.Total, p.Matched, p.Count)
		for _, palette := range p.Palettes {
			if palette.ID == "" || palette.Category == "" || len(palette.Colors) < 2 {
				fail(fmt.Errorf("result contains an incomplete palette"))
			}
			fmt.Printf("%-25s %2d colors · %s\n", palette.ID, len(palette.Colors), palette.Category)
			fmt.Printf("  %s\n", strings.Join(palette.Colors, " "))
		}
		return
	}
	var envelope struct {
		Artifact json.RawMessage `json:"artifact"`
		Cells    []struct {
			Algorithm string `json:"algorithm"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		fail(err)
	}
	if len(envelope.Artifact) > 0 {
		raw = envelope.Artifact
	}
	var a artifact
	if err := json.Unmarshal(raw, &a); err != nil {
		fail(err)
	}
	if a.Path == "" || len(a.SHA256) != 64 || a.Bytes < 1 {
		fail(fmt.Errorf("result does not contain a complete artifact"))
	}
	palette := a.Recipe.Palette
	if palette == "" {
		palette = strings.Join(a.Recipe.Colors, ", ")
	}
	fmt.Printf("TOOL     %s\nLOCAL    %s\nIMAGE    %d × %d · %s · %d frame(s)\nBYTES    %d\n", a.Operation, a.Path, a.Width, a.Height, strings.ToUpper(a.Format), a.Frames, a.Bytes)
	fmt.Printf("RECIPE   v%d · %s · %s", a.Recipe.Version, a.Recipe.Options.Algorithm, palette)
	if a.Recipe.Options.PixelScale > 1 {
		fmt.Printf(" · %d× pixels", a.Recipe.Options.PixelScale)
	}
	fmt.Println()
	if len(envelope.Cells) > 0 {
		names := make([]string, len(envelope.Cells))
		for i, c := range envelope.Cells {
			names[i] = c.Algorithm
		}
		fmt.Printf("CELLS    %d · %s\n", len(names), strings.Join(names, ", "))
	}
	fmt.Printf("SHA256   %s\nENGINE   %s\n", a.SHA256, a.Engine)
}
