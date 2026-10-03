// mcp-demo calls the local MCP server through stdio. It uses no language model or network API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(filepath.Join(root, "demos"), ".mcp-session-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	relative, err := filepath.Rel(root, dir)
	if err != nil {
		return err
	}
	c, err := client.NewStdioMCPClient(filepath.Join(root, "bin", "dither-mcp"), nil, "mcp", "--root", root)
	if err != nil {
		return err
	}
	defer c.Close()
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "dither-demo", Version: "1"}
	session, err := c.Initialize(ctx, init)
	if err != nil {
		return err
	}
	fmt.Printf("CONNECTED  %s %s · MCP over stdio\n", session.ServerInfo.Name, session.ServerInfo.Version)
	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}
	resources, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return err
	}
	prompts, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return err
	}
	fmt.Printf("DISCOVER   %d tools · %d resources · %d prompts\n", len(tools.Tools), len(resources.Resources), len(prompts.Prompts))
	call := func(name string, args map[string]any) (map[string]any, error) {
		r := mcp.CallToolRequest{}
		r.Params.Name = name
		r.Params.Arguments = args
		result, err := c.CallTool(ctx, r)
		if err != nil {
			return nil, err
		}
		if result.IsError {
			return nil, fmt.Errorf("%s: %v", name, result.Content)
		}
		var out map[string]any
		b, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &out)
		return out, err
	}
	catalog, err := call("dither_catalog", map[string]any{})
	if err != nil {
		return err
	}
	fmt.Printf("CATALOG    %d algorithms · bounded local processing\n", len(catalog["algorithms"].([]any)))
	source := "docs/assets/source/moon-garden.png"
	meta, err := call("dither_inspect", map[string]any{"input": source})
	if err != nil {
		return err
	}
	fmt.Printf("INSPECT    moon-garden.png · %.0f × %.0f\n", meta["width"], meta["height"])
	comparison, err := call("dither_compare", map[string]any{"input": source, "output": filepath.ToSlash(filepath.Join(relative, "compare.png")), "palette": "gameboy", "options": map[string]any{"width": 240}, "algorithms": []string{"floyd-steinberg", "atkinson", "bayer-8"}})
	if err != nil {
		return err
	}
	fmt.Printf("COMPARE    3 treatments · %d indexed cells\n", len(comparison["cells"].([]any)))
	output := filepath.ToSlash(filepath.Join(relative, "atkinson.png"))
	artifact, err := call("dither_render", map[string]any{"input": source, "output": output, "palette": "gameboy", "options": map[string]any{"algorithm": "atkinson", "width": 480, "pixel_scale": 2}})
	if err != nil {
		return err
	}
	fmt.Printf("RENDER     Atkinson / Game Boy · %.0f × %.0f\n", artifact["width"], artifact["height"])
	preview := mcp.CallToolRequest{}
	preview.Params.Name = "dither_preview"
	preview.Params.Arguments = map[string]any{"input": output, "width": 320}
	p, err := c.CallTool(ctx, preview)
	if err != nil {
		return err
	}
	if p.IsError || len(p.Content) != 2 {
		return fmt.Errorf("MCP preview result does not contain the expected image content")
	}
	fmt.Println("PREVIEW    The server returned a PNG image to the client.")
	_, err = call("dither_recipe_save", map[string]any{"output": filepath.ToSlash(filepath.Join(relative, "recipe.json")), "recipe": artifact["recipe"]})
	if err != nil {
		return err
	}
	fmt.Println("RECIPE     The server saved a versioned JSON recipe.")
	sum := artifact["sha256"].(string)
	fmt.Printf("VERIFY     SHA256 %s…\n", sum[:20])
	repeat := mcp.CallToolRequest{}
	repeat.Params.Name = "dither_render"
	repeat.Params.Arguments = map[string]any{"input": source, "output": output}
	blocked, err := c.CallTool(ctx, repeat)
	if err != nil {
		return err
	}
	if !blocked.IsError {
		return fmt.Errorf("render accepted an existing output path")
	}
	fmt.Println("GUARD      The server preserved the existing output.")
	fmt.Println("DONE       The client completed the local MCP session.")
	return nil
}
