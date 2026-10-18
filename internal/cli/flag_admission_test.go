package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/internal/app"
)

func flagAdmissionSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 32), G: uint8(y * 40), B: 100, A: 255})
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, source); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.png"), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInspectRejectsIgnoredFlags(t *testing.T) {
	root := flagAdmissionSource(t)
	for _, flags := range [][]string{{"--output", "ignored.png"}, {"--width", "100"}, {"--palette", "nonexistent"}} {
		var out, diagnostics bytes.Buffer
		args := append([]string{"inspect", "--root", root, "--input", "source.png"}, flags...)
		err := Run(context.Background(), args, nil, &out, &diagnostics)
		if err == nil || !strings.Contains(err.Error(), flags[0]) || !strings.Contains(err.Error(), "inspect") {
			t.Errorf("inspect silently ignored %v: error %v", flags, err)
		}
		if out.Len() != 0 {
			t.Errorf("unsupported inspect flag emitted a successful result: %s", out.Bytes())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "ignored.png")); !os.IsNotExist(err) {
		t.Fatalf("inspection wrote an artifact: %v", err)
	}
}

func TestUnsupportedFlagsFailBeforeWorkspaceAccess(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	for _, tc := range []struct {
		command string
		flag    string
		value   string
	}{
		{"catalog", "input", "missing.png"},
		{"palettes", "width", "0"},
		{"inspect", "output", "ignored.png"},
		{"render", "frames", "0"},
		{"render", "algorithms", "atkinson"},
		{"compare", "effect", "wave"},
		{"animate", "start", "0"},
		{"video", "effect", "wave"},
		{"video", "columns", "0"},
		{"separate", "columns", "0"},
		{"mcp", "serpentine", "false"},
		{"call", "output", "ignored.png"},
		{"call", "options", "not-json"},
	} {
		t.Run(tc.command+"/"+tc.flag, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			args := []string{tc.command, "--root", missing, "--" + tc.flag + "=" + tc.value}
			if tc.command == "call" {
				args = append(args, "dither_catalog", "{}")
			}
			err := Run(context.Background(), args, strings.NewReader(""), &out, &diagnostics)
			if err == nil || !strings.Contains(err.Error(), "--"+tc.flag) || !strings.Contains(err.Error(), tc.command) || strings.Contains(err.Error(), "open workspace") {
				t.Fatalf("unsupported flag did not fail before workspace access: %v", err)
			}
			if out.Len() != 0 {
				t.Fatalf("unsupported flag wrote to stdout: %s", out.Bytes())
			}
		})
	}
}

func TestCompareRejectsOverriddenAlgorithmFlag(t *testing.T) {
	for _, flags := range [][]string{{"--algorithm", "atkinson"}, {"--algorithms", "threshold,atkinson", "--algorithm", "bayer-8"}} {
		var out, diagnostics bytes.Buffer
		args := append([]string{"compare", "--root", filepath.Join(t.TempDir(), "missing")}, flags...)
		err := Run(context.Background(), args, nil, &out, &diagnostics)
		if err == nil || !strings.Contains(err.Error(), "--algorithm") || !strings.Contains(err.Error(), "--palettes") || strings.Contains(err.Error(), "open workspace") {
			t.Fatalf("overridden comparison algorithm was not explained: %v", err)
		}
	}
}

func TestSupportedCommandFlagsRemainAvailable(t *testing.T) {
	root := flagAdmissionSource(t)
	for _, args := range [][]string{
		{"catalog", "--allow-video=false"},
		{"palettes", "--query", "game boy", "--category", "retro", "--min-colors", "2", "--max-colors", "8", "--limit", "1", "--offset", "0"},
		{"inspect", "--input", "source.png"},
		{"render", "--input", "source.png", "--output", "render.png", "--algorithm", "atkinson", "--palette", "mono", "--width", "8", "--height", "6", "--pixel-scale", "1", "--seed", "0", "--serpentine=false", "--invert=false", "--grayscale=false", "--brightness", "0", "--contrast", "1", "--gamma", "1", "--saturation", "1", "--strength", "1", "--threshold", "0.5", "--format", "png", "--dpi", "72", "--options", `{"resize_filter":"nearest","brightness":0.1}`},
		{"call", "dither_recipe_save", `{"output":"style.json","recipe":{"version":1,"palette":"gameboy","options":{"algorithm":"atkinson"}}}`},
		{"render", "--input", "source.png", "--output", "recipe.png", "--recipe", "style.json", "--mask-input", "source.png"},
		{"separate", "--input", "source.png", "--output", "plates.zip", "--colors", "#000,#fff", "--algorithm", "threshold", "--format", "zip", "--dpi", "72"},
		{"compare", "--input", "source.png", "--output", "algorithms.png", "--algorithms", "threshold,atkinson", "--palette", "mono", "--width", "8", "--columns", "2"},
		{"compare", "--input", "source.png", "--output", "palettes.png", "--palettes", "mono,gameboy", "--algorithm", "threshold", "--width", "8", "--columns", "2"},
		{"animate", "--input", "source.png", "--output", "motion.png", "--effect", "pulse", "--frames", "2", "--fps", "12", "--columns", "2", "--amplitude", "0.1", "--width", "8"},
		{"call", "dither_inspect", `{"input":"source.png"}`},
	} {
		t.Run(args[0], func(t *testing.T) {
			withRoot := append([]string{args[0], "--root", root}, args[1:]...)
			var out, diagnostics bytes.Buffer
			if err := Run(context.Background(), withRoot, nil, &out, &diagnostics); err != nil {
				t.Fatalf("valid flags %v: %v", args, err)
			}
			if !json.Valid(out.Bytes()) || diagnostics.Len() != 0 {
				t.Fatalf("invalid CLI output: stdout %q, stderr %q", out.Bytes(), diagnostics.Bytes())
			}
			if args[0] == "render" && strings.Contains(strings.Join(args, " "), "--options") {
				var result app.Artifact
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Recipe.Options.Brightness != .1 || result.Recipe.Options.Contrast == nil || *result.Recipe.Options.Contrast != 1 || result.Recipe.Options.ResizeFilter != "nearest" {
					t.Fatalf("advanced options merge changed: %+v", result.Recipe.Options)
				}
			}
		})
	}
	// Video admission should reach the existing opt-in check without needing FFmpeg.
	var out, diagnostics bytes.Buffer
	err := Run(context.Background(), []string{"video", "--root", root, "--input", "missing.mp4", "--output", "movie.gif", "--frames", "2", "--fps", "12", "--start", "0", "--allow-video=false"}, nil, &out, &diagnostics)
	if err == nil || !strings.Contains(err.Error(), "video disabled") {
		t.Fatalf("valid video flags failed CLI admission: %v", err)
	}
}

func TestUnregisteredFlagErrorsIdentifyCommand(t *testing.T) {
	for _, tc := range []struct{ command, flag string }{
		{"render", "query"},
		{"inspect", "palettes"},
		{"palettes", "not-a-real-flag"},
	} {
		var out, diagnostics bytes.Buffer
		err := Run(context.Background(), []string{tc.command, "--root", filepath.Join(t.TempDir(), "missing"), "--" + tc.flag, "value"}, nil, &out, &diagnostics)
		if err == nil || !strings.Contains(err.Error(), tc.command+":") || !strings.Contains(err.Error(), tc.flag) {
			t.Errorf("%s --%s: error %v must identify command and flag", tc.command, tc.flag, err)
		}
		if out.Len() != 0 {
			t.Errorf("invalid flags emitted stdout: %s", out.Bytes())
		}
	}
}
