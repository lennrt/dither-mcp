package mcpserver

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
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
	if len(listed.Tools) != 15 {
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
		if tool.Name == "dither_compare" {
			inputURI := uri + ":input"
			if err := compiler.AddResource(inputURI, schema); err != nil {
				t.Fatal(err)
			}
			inputSchema, err := compiler.Compile(inputURI)
			if err != nil {
				t.Fatal(err)
			}
			if err := inputSchema.Validate(map[string]any{"input": "source.png", "output": "palettes.png", "palettes": []any{"gameboy", "cga"}}); err != nil {
				t.Fatalf("palette comparison violates advertised input schema: %v", err)
			}
		}
		if tool.Name == "dither_random_style" {
			if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint || tool.Annotations.IdempotentHint == nil || *tool.Annotations.IdempotentHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
				t.Fatalf("random style annotations misrepresent file access or freshness: %+v", tool.Annotations)
			}
			inputURI := uri + ":input"
			if err := compiler.AddResource(inputURI, schema); err != nil {
				t.Fatal(err)
			}
			inputSchema, err := compiler.Compile(inputURI)
			if err != nil {
				t.Fatal(err)
			}
			for _, args := range []map[string]any{{}, {"seed": 0}, {"seed": app.MaxStyleSeed}, {"seed": -app.MaxStyleSeed}} {
				if err := inputSchema.Validate(args); err != nil {
					t.Fatalf("valid random style request violates advertised schema: %v", err)
				}
			}
			for _, args := range []map[string]any{{"seed": nil}, {"seed": 1.5}, {"seed": "1"}, {"seed": app.MaxStyleSeed + 1}, {"seed": -app.MaxStyleSeed - 1}, {"output": "unexpected.json"}} {
				if err := inputSchema.Validate(args); err == nil {
					t.Fatalf("invalid random style request satisfies advertised schema: %+v", args)
				}
			}
		}
	}
	resources, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil || len(resources.Resources) != 3 {
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
	styleResult := call("dither_random_style", map[string]any{"seed": 42})
	if styleResult.IsError {
		t.Fatalf("random style failed: %+v", styleResult.Content)
	}
	styleWire, _ := json.Marshal(styleResult.StructuredContent)
	var style app.RandomStyleResult
	if err := json.Unmarshal(styleWire, &style); err != nil || style.Seed != 42 || style.Recipe.Options.Seed != 42 {
		t.Fatalf("random style metadata: %+v / %v", style, err)
	}
	directStyle, err := svc.Do(ctx, "dither_random_style", []byte(`{"seed":42}`))
	if err != nil || !reflect.DeepEqual(style, directStyle) {
		t.Fatalf("MCP and service style generation disagree: %v", err)
	}
	if files, err := os.ReadDir(root); err != nil || len(files) != 1 {
		t.Fatalf("random style wrote files: %v", err)
	}
	if preview := call("dither_studio", map[string]any{"input": "source.png", "palette": style.Recipe.Palette, "options": style.Recipe.Options}); preview.IsError || len(preview.Content) < 2 {
		t.Fatalf("generated recipe did not preview through MCP: %+v", preview.Content)
	}
	for _, args := range []map[string]any{{"seed": nil}, {"seed": 1.5}, {"seed": "1"}, {"seed": app.MaxStyleSeed + 1}, {"seed": -app.MaxStyleSeed - 1}, {"input": "source.png"}} {
		if result := call("dither_random_style", args); !result.IsError {
			t.Fatalf("random style accepted invalid arguments: %+v", args)
		}
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
	paletteArgs := map[string]any{"input": "source.png", "output": "palettes.png", "palettes": []string{"gameboy", "cga"}, "options": map[string]any{"algorithm": "atkinson", "width": 16}, "columns": 2}
	paletteResult := call("dither_compare", paletteArgs)
	if paletteResult.IsError {
		t.Fatalf("palette comparison failed: %v", paletteResult.Content)
	}
	wire, _ := json.Marshal(paletteResult.StructuredContent)
	var comparison app.CompareResult
	if err := json.Unmarshal(wire, &comparison); err != nil {
		t.Fatal(err)
	}
	if comparison.Artifact.Recipe != nil || len(comparison.Cells) != 2 || comparison.Cells[0].Palette != "gameboy" || comparison.Cells[1].Palette != "cga" || comparison.Cells[0].Algorithm != "atkinson" {
		t.Fatalf("MCP lost palette comparison metadata: %+v", comparison)
	}
	paletteArgs["output"] = "palettes-direct.png"
	raw, _ := json.Marshal(paletteArgs)
	direct, err := svc.Do(ctx, "dither_compare", raw)
	if err != nil {
		t.Fatal(err)
	}
	if direct.(app.CompareResult).Artifact.SHA256 != comparison.Artifact.SHA256 {
		t.Fatal("MCP and shared service produced different palette comparison pixels")
	}
	paletteArgs["output"] = "palettes-invalid.png"
	paletteArgs["algorithms"] = []string{"atkinson"}
	if result := call("dither_compare", paletteArgs); !result.IsError {
		t.Fatal("MCP accepted both comparison modes")
	}
	if _, err := os.Stat(filepath.Join(root, "palettes-invalid.png")); !os.IsNotExist(err) {
		t.Fatal("invalid MCP comparison published output")
	}
	for _, step := range []struct {
		name string
		args any
	}{
		{"dither_palettes", map[string]any{}},
		{"dither_random_style", map[string]any{}},
		{"dither_random_style", map[string]any{"seed": 0}},
		{"dither_random_style", map[string]any{"seed": -app.MaxStyleSeed}},
		{"dither_random_style", map[string]any{"seed": app.MaxStyleSeed}},
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
