package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/lennrt/dither-mcp/internal/mcpapp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const appsExtensionID = "io.modelcontextprotocol/ui"

func appsExtension() map[string]any {
	return map[string]any{appsExtensionID: map[string]any{"mimeTypes": []string{mcpapp.MIMEType}}}
}

func supportsApps(ctx context.Context) bool {
	session, ok := server.ClientSessionFromContext(ctx).(server.SessionWithClientInfo)
	if !ok {
		return false
	}
	value := session.GetClientCapabilities().Extensions[appsExtensionID]
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var capability struct {
		MIMETypes []string `json:"mimeTypes"`
	}
	if json.Unmarshal(data, &capability) != nil {
		return false
	}
	for _, mime := range capability.MIMETypes {
		if mime == mcpapp.MIMEType {
			return true
		}
	}
	return false
}

func studioToolMeta() *mcp.Meta {
	return mcp.NewMetaFromMap(map[string]any{"ui": map[string]any{"resourceUri": mcpapp.URI, "visibility": []string{"model", "app"}}})
}

// Filter copies descriptors before removing UI metadata. A client without Apps
// retains the same callable tool and its PNG, JSON, and recipe fallback.
func appsToolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	if supportsApps(ctx) {
		return tools
	}
	copyTools := append([]mcp.Tool(nil), tools...)
	for i := range copyTools {
		if copyTools[i].Name == "dither_studio" {
			copyTools[i].Meta = nil
		}
	}
	return copyTools
}

func studioResourceMeta() map[string]any {
	return map[string]any{"ui": map[string]any{
		"prefersBorder": true,
		"permissions":   map[string]any{},
		"csp": map[string]any{
			"connectDomains": []string{}, "resourceDomains": []string{},
			"frameDomains": []string{}, "baseUriDomains": []string{},
		},
	}}
}

func addStudioResource(s *server.MCPServer) {
	resource := mcp.NewResource(mcpapp.URI, "Dither studio", mcp.WithMIMEType(mcpapp.MIMEType), mcp.WithResourceDescription("Interactive algorithm and palette previews for MCP Apps hosts. All code is embedded. The app needs no external network or browser permissions."))
	resource.Meta = mcp.NewMetaFromMap(studioResourceMeta())
	s.AddResource(resource, func(ctx context.Context, r mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{URI: mcpapp.URI, MIMEType: mcpapp.MIMEType, Text: mcpapp.HTML(), Meta: studioResourceMeta()}}, nil
	})
}

func studioResult(ctx context.Context, preview app.StudioResult) *mcp.CallToolResult {
	text, _ := json.Marshal(preview.StudioMetadata)
	result := mcp.NewToolResultStructured(preview.StudioMetadata, string(text))
	result.Content = append(result.Content, mcp.NewImageContent(preview.Data, preview.MIMEType))
	if supportsApps(ctx) {
		result.Meta = mcp.NewMetaFromMap(map[string]any{"dither": map[string]any{
			"request": preview.Request(), "algorithms": engine.Catalog(), "palettes": engine.Palettes(),
		}})
	}
	return result
}
