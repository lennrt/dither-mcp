package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lennrt/dither-mcp/internal/app"
)

func TestCLI(t *testing.T) {
	root := t.TempDir()
	f, err := os.Create(filepath.Join(root, "source.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	var out, diagnostics bytes.Buffer
	err = Run(context.Background(), []string{"render", "--root", root, "--input", "source.png", "--output", "result.png", "--algorithm", "atkinson", "--palette", "mono", "--width", "16", "--contrast", "0"}, strings.NewReader(""), &out, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.Len() != 0 {
		t.Fatal(diagnostics.String())
	}
	var result map[string]any
	if err = json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("stdout not JSON: %v", err)
	}
	if result["width"] != float64(16) {
		t.Fatal(result)
	}
	if err := Run(context.Background(), []string{"render", "--root", root, "--input", "source.png", "--output", "result.png"}, nil, &out, &diagnostics); err == nil {
		t.Fatal("overwrite accepted")
	}
	for _, args := range [][]string{{"nonsense"}, {"call", "--root", root}, {"render", "--root", root, "unexpected"}, {"render", "--root", root, "--gamma", "no"}} {
		if err := Run(context.Background(), args, nil, &out, &diagnostics); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	out.Reset()
	if err := Run(context.Background(), []string{"help"}, nil, &out, &diagnostics); err != nil || !strings.Contains(out.String(), "MCP") {
		t.Fatal("help broken")
	}
}
func TestStdioNoBanner(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}` + "\n"
	var out, diagnostics bytes.Buffer
	err := Run(context.Background(), []string{"mcp", "--root", t.TempDir()}, strings.NewReader(input), &out, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("no response")
	}
	for _, line := range lines {
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			t.Fatalf("stdout pollution: %q", line)
		}
		if raw["jsonrpc"] != "2.0" {
			t.Fatal(raw)
		}
	}
}

func TestMCPMessageLimit(t *testing.T) {
	r := &lineLimitReader{source: strings.NewReader(strings.Repeat("x", maxMessageBytes+1) + "\n")}
	b, err := io.ReadAll(r)
	if err == nil || len(b) > maxMessageBytes {
		t.Fatalf("limit: len=%d err=%v", len(b), err)
	}
	r = &lineLimitReader{source: strings.NewReader(strings.Repeat("x", maxMessageBytes) + "\n" + strings.Repeat("x", maxMessageBytes) + "\n")}
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatalf("valid bounded lines rejected: %v", err)
	}
}

func TestPaletteDiscoveryCLI(t *testing.T) {
	root := t.TempDir()
	var out, diagnostics bytes.Buffer
	args := []string{"palettes", "--root", root, "--query", "GAME BOY", "--category", "retro", "--min-colors", "4", "--max-colors", "4", "--limit", "1"}
	if err := Run(context.Background(), args, nil, &out, &diagnostics); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Palettes []struct {
			ID string `json:"id"`
		} `json:"palettes"`
		Count int `json:"count"`
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Limit != 1 || result.Palettes[0].ID != "gameboy" {
		t.Fatalf("unexpected discovery: %s", out.Bytes())
	}
	out.Reset()
	if err := Run(context.Background(), []string{"palettes", "--root", root, "--limit", "257"}, nil, &out, &diagnostics); err == nil {
		t.Fatal("invalid limit accepted")
	}
}

func TestPaletteComparisonCLI(t *testing.T) {
	root := t.TempDir()
	source := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 32), G: uint8(y * 40), B: 100, A: 255})
		}
	}
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, source); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.png"), imageData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostics bytes.Buffer
	args := []string{"compare", "--root", root, "--input", "source.png", "--output", "palettes.png", "--palettes", "gameboy,cga", "--algorithm", "atkinson", "--width", "4", "--columns", "2"}
	if err := Run(context.Background(), args, nil, &out, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if diagnostics.Len() != 0 {
		t.Fatal(diagnostics.String())
	}
	var result app.CompareResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Artifact.Width != 8 || result.Artifact.Height != 3 || result.Artifact.Recipe != nil || len(result.Cells) != 2 || result.Cells[0].Palette != "gameboy" || result.Cells[1].Palette != "cga" || result.Cells[0].Algorithm != "atkinson" {
		t.Fatalf("incorrect CLI comparison: %+v", result)
	}
	out.Reset()
	if err := Run(context.Background(), []string{"call", "--root", root, "dither_compare", `{"input":"source.png","output":"call.png","palettes":["gameboy","cga"],"options":{"algorithm":"atkinson","width":4},"columns":2}`}, nil, &out, &diagnostics); err != nil {
		t.Fatal(err)
	}
	var called app.CompareResult
	if err := json.Unmarshal(out.Bytes(), &called); err != nil {
		t.Fatal(err)
	}
	if result.Artifact.SHA256 != called.Artifact.SHA256 {
		t.Fatal("compare flags and tool-shaped call produced different pixels")
	}
	for _, flags := range [][]string{
		{"--palettes", "gameboy", "--algorithms", "atkinson"},
		{"--palettes", "gameboy", "--algorithms", "floyd-steinberg,atkinson,bayer-8"},
		{"--palettes", ""},
		{"--palettes", "gameboy", "--palette", "mono"},
		{"--palettes", "gameboy", "--colors", "#000,#fff"},
		{"--palettes", "gameboy", "--recipe", "recipe.json"},
	} {
		out.Reset()
		args := append([]string{"compare", "--root", root, "--input", "source.png", "--output", "invalid.png"}, flags...)
		if err := Run(context.Background(), args, nil, &out, &diagnostics); err == nil || out.Len() != 0 {
			t.Fatalf("invalid comparison flags succeeded: %v", flags)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "invalid.png")); !os.IsNotExist(err) {
		t.Fatal("invalid comparison flags published output")
	}
	out.Reset()
	if err := Run(context.Background(), []string{"compare", "--root", root, "--input", "source.png", "--output", "algorithms.png", "--width", "4"}, nil, &out, &diagnostics); err != nil {
		t.Fatal(err)
	}
	var algorithms app.CompareResult
	if err := json.Unmarshal(out.Bytes(), &algorithms); err != nil {
		t.Fatal(err)
	}
	if algorithms.Artifact.Recipe == nil || len(algorithms.Cells) != 3 || algorithms.Cells[0].Algorithm != "floyd-steinberg" || algorithms.Cells[1].Algorithm != "atkinson" || algorithms.Cells[2].Algorithm != "bayer-8" || algorithms.Cells[0].Palette != "" {
		t.Fatalf("default algorithm mode changed: %+v", algorithms)
	}
}
