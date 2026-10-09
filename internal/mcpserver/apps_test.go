package mcpserver

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/lennrt/dither-mcp/internal/mcpapp"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestAppsNegotiationAndFallback(t *testing.T) {
	for _, tt := range []struct {
		name       string
		extensions map[string]any
		apps       bool
	}{
		{"apps", appsExtension(), true},
		{"plain", nil, false},
		{"other-mime", map[string]any{appsExtensionID: map[string]any{"mimeTypes": []string{"text/html"}}}, false},
		{"malformed", map[string]any{appsExtensionID: map[string]any{"mimeTypes": mcpapp.MIMEType}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			im := image.NewNRGBA(image.Rect(0, 0, 20, 12))
			im.SetNRGBA(1, 1, color.NRGBA{200, 180, 90, 255})
			f, err := os.Create(filepath.Join(root, "source.png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, im); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			svc, err := app.New(root, false)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			server := New(svc)
			c, err := client.NewInProcessClient(server)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := c.Start(ctx); err != nil {
				t.Fatal(err)
			}
			init := mcp.InitializeRequest{}
			init.Params.ProtocolVersion = "2025-11-25"
			init.Params.ClientInfo = mcp.Implementation{Name: "apps-test", Version: "1"}
			init.Params.Capabilities.Extensions = tt.extensions
			initialized, err := c.Initialize(ctx, init)
			if err != nil {
				t.Fatal(err)
			}
			if initialized.Capabilities.Extensions[appsExtensionID] == nil {
				t.Fatal("server did not advertise Apps")
			}
			listed, err := c.ListTools(ctx, mcp.ListToolsRequest{})
			if err != nil {
				t.Fatal(err)
			}
			var studio mcp.Tool
			for _, tool := range listed.Tools {
				if tool.Name == "dither_studio" {
					studio = tool
				}
			}
			if studio.Name == "" {
				t.Fatal("studio tool unavailable")
			}
			if (studio.Meta != nil) != tt.apps {
				t.Fatalf("metadata negotiation: %+v", studio.Meta)
			}
			if !*studio.Annotations.ReadOnlyHint || *studio.Annotations.DestructiveHint || *studio.Annotations.OpenWorldHint {
				t.Fatal("incorrect studio annotations")
			}
			if tt.apps {
				ui := studio.Meta.AdditionalFields["ui"].(map[string]any)
				if ui["resourceUri"] != mcpapp.URI {
					t.Fatal("wrong UI resource")
				}
			}
			request := mcp.CallToolRequest{}
			request.Params.Name = "dither_studio"
			request.Params.Arguments = map[string]any{"input": "source.png", "palette": "gameboy"}
			result, err := c.CallTool(ctx, request)
			if err != nil || result.IsError {
				t.Fatalf("studio call: %v %+v", err, result)
			}
			if len(result.Content) != 2 {
				t.Fatal("missing core fallback")
			}
			if _, ok := result.Content[1].(mcp.ImageContent); !ok {
				t.Fatal("missing native PNG")
			}
			var text map[string]any
			if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &text); err != nil || text["recipe"] == nil || text["path"] != "source.png" || text["source_width"] != float64(20) || text["source_height"] != float64(12) || text["export_limits"] == nil {
				t.Fatal("missing replayable JSON fallback")
			}
			if digest, ok := text["source_sha256"].(string); !ok || len(digest) != 64 {
				t.Fatal("missing source fingerprint in core fallback")
			}
			if (result.Meta != nil) != tt.apps {
				t.Fatal("private rendering metadata not negotiated")
			}
			if tt.apps {
				b, _ := json.Marshal(result.Meta)
				var meta struct {
					Dither struct {
						Request    app.StudioRequest
						Algorithms []any
						Palettes   []any
					}
				}
				if err := json.Unmarshal(b, &meta); err != nil {
					t.Fatal(err)
				}
				if len(meta.Dither.Algorithms) != 41 || len(meta.Dither.Palettes) != 256 || meta.Dither.Request.Options.Width != 20 || meta.Dither.Request.Options.Height != 12 || meta.Dither.Request.Palette != "gameboy" {
					t.Fatal("incomplete studio metadata")
				}
			}
			b, _ := json.Marshal(studio)
			var descriptor map[string]any
			if err := json.Unmarshal(b, &descriptor); err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource("urn:studio", descriptor["outputSchema"]); err != nil {
				t.Fatal(err)
			}
			schema, err := compiler.Compile("urn:studio")
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(text); err != nil {
				t.Fatalf("studio output violates schema: %v", err)
			}
			read := mcp.ReadResourceRequest{}
			read.Params.URI = mcpapp.URI
			resource, err := c.ReadResource(ctx, read)
			if err != nil || len(resource.Contents) != 1 {
				t.Fatalf("app resource: %+v %v", resource, err)
			}
			html := resource.Contents[0].(mcp.TextResourceContents)
			if html.URI != mcpapp.URI || html.MIMEType != mcpapp.MIMEType || !strings.Contains(html.Text, "<!doctype html>") && !strings.Contains(html.Text, "<!DOCTYPE html>") {
				t.Fatal("invalid HTML resource")
			}
			data, _ := json.Marshal(html.Meta)
			var policy struct {
				UI struct {
					Permissions map[string]any
					CSP         map[string][]string
				}
			}
			if err := json.Unmarshal(data, &policy); err != nil {
				t.Fatal(err)
			}
			if len(policy.UI.Permissions) != 0 || len(policy.UI.CSP) != 4 {
				t.Fatal("unexpected app permissions")
			}
			for key, domains := range policy.UI.CSP {
				if key == "resourceDomains" {
					if len(domains) != 1 || domains[0] != "data:" {
						t.Fatal("app must declare only inline data resources")
					}
					continue
				}
				if len(domains) != 0 {
					t.Fatal("app requested external network access")
				}
			}
			files, _ := os.ReadDir(root)
			if len(files) != 1 {
				t.Fatal("studio wrote an artifact")
			}
			// A fallback descriptor must never mutate the globally registered tool.
			if server.ListTools()["dither_studio"].Tool.Meta == nil {
				t.Fatal("negotiation mutated shared metadata")
			}
		})
	}
}

func TestStudioExportGuardsReturnStructuredMCPError(t *testing.T) {
	for _, kind := range []string{"source", "mask"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			im := image.NewNRGBA(image.Rect(0, 0, 20, 12))
			im.SetNRGBA(1, 1, color.NRGBA{R: 200, A: 255})
			for _, path := range []string{"source.png", "mask.png"} {
				f, err := os.Create(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, im); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
			}
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
			init.Params.ClientInfo = mcp.Implementation{Name: "guard-test", Version: "1"}
			if _, err := c.Initialize(ctx, init); err != nil {
				t.Fatal(err)
			}
			call := func(name string, arguments any) *mcp.CallToolResult {
				t.Helper()
				request := mcp.CallToolRequest{}
				request.Params.Name, request.Params.Arguments = name, arguments
				result, err := c.CallTool(ctx, request)
				if err != nil {
					t.Fatal(err)
				}
				return result
			}
			preview := call("dither_studio", map[string]any{"input": "source.png", "mask_input": "mask.png"})
			if preview.IsError {
				t.Fatal(preview)
			}
			b, _ := json.Marshal(preview.StructuredContent)
			var metadata app.StudioMetadata
			if err := json.Unmarshal(b, &metadata); err != nil {
				t.Fatal(err)
			}
			path := kind + ".png"
			if err := os.WriteFile(filepath.Join(root, path), []byte("changed image bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			render := call("dither_render", app.RenderRequest{Input: metadata.Path, Output: "exports/blocked.png", Palette: metadata.Recipe.Palette, Colors: metadata.Recipe.Colors, Options: metadata.Recipe.Options, MaskInput: metadata.MaskInput, ExpectedSourceSHA256: metadata.SourceSHA256, ExpectedMaskSHA256: metadata.MaskSHA256})
			if !render.IsError || len(render.Content) != 1 || !strings.HasPrefix(render.Content[0].(mcp.TextContent).Text, kind+"_changed:") {
				t.Fatalf("missing stable fallback error: %+v", render)
			}
			b, _ = json.Marshal(render.StructuredContent)
			var failure struct {
				Error app.InputChangedError `json:"error"`
			}
			if err := json.Unmarshal(b, &failure); err != nil || failure.Error.Code != kind+"_changed" || failure.Error.Path != path || len(failure.Error.ExpectedSHA256) != 64 || len(failure.Error.ActualSHA256) != 64 || failure.Error.Message == "" {
				t.Fatalf("missing structured mismatch details: %s", b)
			}
			if _, err := os.Stat(filepath.Join(root, "exports")); !os.IsNotExist(err) {
				t.Fatal("MCP mismatch created an output directory or artifact")
			}
		})
	}
}
