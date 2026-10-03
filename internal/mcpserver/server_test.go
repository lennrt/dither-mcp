package mcpserver

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestProtocolWorkflow(t *testing.T) {
	root := t.TempDir()
	src := image.NewNRGBA(image.Rect(0, 0, 24, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 24; x++ {
			src.SetNRGBA(x, y, color.NRGBA{uint8(x * 10), uint8(y * 15), 90, 255})
		}
	}
	f, err := os.Create(filepath.Join(root, "source.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, src); err != nil {
		t.Fatal(err)
	}
	f.Close()
	svc, err := app.New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	c, err := client.NewInProcessClient(New(svc))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1"}
	if _, err = c.Initialize(ctx, init); err != nil {
		t.Fatal(err)
	}
	listed, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 13 {
		t.Fatalf("tools=%d", len(listed.Tools))
	}
	seen := map[string]bool{}
	validators := map[string]*jsonschema.Schema{}
	for _, tool := range listed.Tools {
		if seen[tool.Name] {
			t.Fatalf("duplicate tool %s", tool.Name)
		}
		seen[tool.Name] = true
		b, err := json.Marshal(tool)
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]any
		if err = json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		schema, ok := raw["inputSchema"].(map[string]any)
		if !ok || schema["type"] != "object" {
			t.Fatalf("missing object schema: %s", b)
		}
		if raw["outputSchema"] == nil {
			t.Fatalf("missing output schema: %s", tool.Name)
		}
		compiler := jsonschema.NewCompiler()
		uri := "urn:dither:" + tool.Name
		if err := compiler.AddResource(uri, raw["outputSchema"]); err != nil {
			t.Fatal(err)
		}
		compiled, err := compiler.Compile(uri)
		if err != nil {
			t.Fatal(err)
		}
		validators[tool.Name] = compiled
	}
	resources, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil || len(resources.Resources) != 2 {
		t.Fatalf("resources: %v %v", resources, err)
	}
	prompts, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil || len(prompts.Prompts) != 3 {
		t.Fatalf("prompts: %v %v", prompts, err)
	}
	call := func(name string, args any) *mcp.CallToolResult {
		t.Helper()
		r := mcp.CallToolRequest{}
		r.Params.Name = name
		r.Params.Arguments = args
		res, err := c.CallTool(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if !res.IsError {
			b, err := json.Marshal(res.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			var value any
			if err = json.Unmarshal(b, &value); err != nil {
				t.Fatal(err)
			}
			if err := validators[name].Validate(value); err != nil {
				t.Fatalf("%s result violates advertised schema: %v\n%s", name, err, b)
			}
		}
		return res
	}
	result := call("dither_render", map[string]any{"input": "source.png", "output": "out.png", "palette": "gameboy", "options": map[string]any{"algorithm": "atkinson", "width": 48}})
	if result.IsError {
		t.Fatalf("render error: %+v", result)
	}
	b, _ := json.Marshal(result.StructuredContent)
	var artifact app.Artifact
	if err := json.Unmarshal(b, &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Width != 48 || artifact.Height != 32 || len(artifact.SHA256) != 64 || artifact.Recipe == nil {
		t.Fatalf("artifact=%+v", artifact)
	}
	preview := call("dither_preview", map[string]any{"input": "out.png", "width": 24})
	if preview.IsError || len(preview.Content) != 2 {
		t.Fatalf("preview=%+v", preview)
	}
	if _, ok := preview.Content[1].(mcp.ImageContent); !ok {
		t.Fatalf("not image content: %T", preview.Content[1])
	}
	for _, args := range []any{map[string]any{"input": "source.png", "output": "out.png"}, map[string]any{"input": "../outside.png", "output": "bad.png"}, map[string]any{"input": "source.png", "output": "bad.png", "typo": 1}, map[string]any{"input": "source.png", "output": "bad.png", "options": map[string]any{"gamma": 0}}} {
		r := call("dither_render", args)
		if !r.IsError {
			t.Fatalf("expected tool error: %+v", args)
		}
	}
	if r := call("dither_catalog", map[string]any{}); r.IsError {
		t.Fatal("session did not recover after tool errors")
	}
	for _, step := range []struct {
		name string
		args any
	}{
		{"dither_palettes", map[string]any{}},
		{"dither_palettes", map[string]any{"query": "game boy", "category": "retro", "min_colors": 4, "max_colors": 4, "limit": 1}},
		{"dither_inspect", map[string]any{"input": "source.png"}},
		{"dither_palette_extract", map[string]any{"input": "source.png", "count": 4}},
		{"dither_compare", map[string]any{"input": "source.png", "output": "compare.png", "algorithms": []string{"atkinson", "bayer-4"}, "options": map[string]any{"width": 16}}},
		{"dither_animate", map[string]any{"input": "source.png", "output": "loop.gif", "frames": 4, "options": map[string]any{"width": 16}}},
		{"dither_batch", map[string]any{"items": []map[string]any{{"input": "source.png", "output": "batch.png"}, {"input": "missing.png", "output": "bad.png"}}}},
		{"dither_separate", map[string]any{"input": "source.png", "output": "plates.zip"}},
		{"dither_recipe_save", map[string]any{"output": "recipe.json", "recipe": artifact.Recipe}},
		{"dither_recipe_load", map[string]any{"input": "recipe.json"}},
	} {
		if r := call(step.name, step.args); r.IsError {
			t.Fatalf("%s: %v", step.name, r.Content)
		}
	}
}
