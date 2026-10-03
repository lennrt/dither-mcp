package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
