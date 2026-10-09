// Package mcpserver exposes the local service over the Model Context Protocol.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

var Tools = []ToolInfo{
	{"dither_catalog", "Discover algorithm IDs, format roles, media capabilities, defaults, and enforced limits. Still images, animation, video, and print have separate capability sections.", true},
	{"dither_palettes", "Search palettes by query, category, and color count. The result includes exact hex colors, descriptions, provenance, and tags in stable ID order. The default page size is 32. The maximum is 256. Use next_offset with the same filters for the next page. The categories array lists all categories.", true},
	{"dither_inspect", "Inspect upright image dimensions, format, frames, alpha, byte size, SHA256, and input normalization. The normalization object reports stored dimensions, EXIF orientation, and the selected color declaration.", true},
	{"dither_preview", "Return a bounded PNG preview for visual inspection. The tool sends the preview to the MCP client.", true},
	{"dither_palette_extract", "Extract 2..256 representative sRGB colors deterministically from a normalized local image.", true},
	{"dither_render", "Apply a deterministic recipe to the first frame of a local image. The tool creates a new output and returns its dimensions, hash, and recipe. Optional source and mask SHA256 guards require the exact input bytes from a studio preview. A changed or unreadable guarded input creates no output and returns source_changed or mask_changed. Use dither_preview to inspect the result.", false},
	{"dither_compare", "Render 1..12 algorithms into a contact sheet with exact cell coordinates. Cells default to 320 pixels wide.", false},
	{"dither_batch", "Render 1..32 requests in sequence. The tool publishes each successful output separately and reports each failed item.", false},
	{"dither_animate", "Process source GIF frames or generate wave, orbit, pulse, noise, or palette-cycle motion. Export a GIF or PNG sprite sheet. GIF input defaults to source processing, which preserves frame count, delays, and loop count. Other input defaults to wave motion with 24 frames at 12 fps. Width defaults to 480 pixels when both dimensions are omitted or zero.", false},
	{"dither_video", "Process a bounded local MP4, MOV, WebM, Matroska, or AVI segment when local ffmpeg support is enabled. Export MP4, WebM, GIF, or a PNG sheet. The output contains no audio.", false},
	{"dither_separate", "Create a ZIP with one black-on-white PNG plate per palette ink and a manifest. The tool supports up to 16 inks. The recipe must omit masks and post-effects.", false},
	{"dither_recipe_save", "Validate a version-1 JSON recipe. Save it to a new relative path without replacing an existing file.", false},
	{"dither_recipe_load", "Read and validate a saved version-1 JSON recipe.", true},
	{"dither_studio", "Preview a dither recipe in memory. Supporting MCP Apps hosts open an interactive studio with algorithms, palettes, adjustments, and an explicit Save action. Other hosts receive a PNG and a replayable recipe. Metadata includes source and mask byte fingerprints, upright source dimensions after cropping, and export limits. No files are written by this tool. Preview dimensions are at most 1024 per side and default to fit within 512 by 512 without enlargement.", true},
}

func New(svc *app.Service) *server.MCPServer {
	s := server.NewMCPServer("dither-mcp", app.Version, server.WithToolCapabilities(false), server.WithResourceCapabilities(false, false), server.WithPromptCapabilities(false), server.WithRecovery(), server.WithInstructions(guide), server.WithTitle("Dither MCP"), server.WithExtensions(appsExtension()), server.WithToolFilter(appsToolFilter))
	add[app.Empty, app.CatalogResult](s, svc, Tools[0])
	add[app.PaletteQuery, app.PaletteList](s, svc, Tools[1])
	add[app.InputRequest, app.Inspection](s, svc, Tools[2])
	add[app.PreviewRequest, previewMetadata](s, svc, Tools[3])
	add[app.ExtractRequest, colorList](s, svc, Tools[4])
	add[app.RenderRequest, app.Artifact](s, svc, Tools[5])
	add[app.CompareRequest, app.CompareResult](s, svc, Tools[6])
	add[app.BatchRequest, app.BatchResult](s, svc, Tools[7])
	add[app.AnimateRequest, app.Artifact](s, svc, Tools[8])
	add[app.VideoRequest, app.Artifact](s, svc, Tools[9])
	add[app.RenderRequest, app.Artifact](s, svc, Tools[10])
	add[app.RecipeSaveRequest, app.Artifact](s, svc, Tools[11])
	add[app.InputRequest, app.Recipe](s, svc, Tools[12])
	add[app.StudioRequest, app.StudioMetadata](s, svc, Tools[13])
	addStudioResource(s)
	s.AddResource(mcp.NewResource("dither://capabilities", "Dither capabilities", mcp.WithMIMEType("application/json")), func(ctx context.Context, r mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		b, err := json.Marshal(svc.Catalog())
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{URI: r.Params.URI, MIMEType: "application/json", Text: string(b)}}, nil
	})
	s.AddResource(mcp.NewResource("dither://workflow", "Agent workflow guide", mcp.WithMIMEType("text/plain")), func(ctx context.Context, r mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{mcp.TextResourceContents{URI: r.Params.URI, MIMEType: "text/plain", Text: guide}}, nil
	})
	for _, p := range []struct{ name, description, text string }{
		{"art_director", "Compare looks, inspect previews, and save a reproducible recipe.", "1. Inspect the input.\n2. Discover algorithms and palettes.\n3. Create a small comparison with diffusion, ordered, and artistic algorithms.\n4. Preview the comparison.\n5. Explain the differences.\n6. Apply the user's direction.\n7. Preview the artifact.\n8. Save the recipe."},
		{"prepare_print", "Create spot-color print separations.", "1. Inspect the input.\n2. Choose at most 16 inks.\n3. Render at the intended dimensions.\n4. Preview the result.\n5. Generate dither_separate PNG plates at the requested DPI.\n6. Explain that the plates provide coverage masks without press calibration."},
		{"make_loop", "Create a deterministic GIF or sprite sheet.", "1. Inspect the input.\n2. Check the frame budgets.\n3. For an existing GIF, choose source to process its frames. Otherwise, choose wave, orbit, pulse, or palette-cycle.\n4. Render a small GIF at 12 fps.\n5. Inspect a preview of its first frame.\n6. Export the requested GIF or PNG sprite sheet.\n7. Report the frames and dimensions."},
	} {
		p := p
		s.AddPrompt(mcp.NewPrompt(p.name, mcp.WithPromptDescription(p.description)), func(ctx context.Context, r mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return mcp.NewGetPromptResult(p.description, []mcp.PromptMessage{mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(guide+"\n\n"+p.text))}), nil
		})
	}
	return s
}
func add[T any, O any](s *server.MCPServer, svc *app.Service, info ToolInfo) {
	tool := mcp.NewTool(info.Name, mcp.WithDescription(info.Description), mcp.WithInputSchema[T](), mcp.WithOutputSchema[O](), mcp.WithReadOnlyHintAnnotation(info.ReadOnly), mcp.WithDestructiveHintAnnotation(false), mcp.WithIdempotentHintAnnotation(info.ReadOnly), mcp.WithOpenWorldHintAnnotation(false))
	if info.Name == "dither_studio" {
		tool.Meta = studioToolMeta()
	}
	s.AddTool(tool, func(ctx context.Context, r mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var b []byte
		var err error
		if len(r.Params.RawArguments) > 0 {
			b = r.Params.RawArguments
		} else {
			args := r.Params.Arguments
			if args == nil {
				args = map[string]any{}
			}
			b, err = json.Marshal(args)
		}
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		v, err := svc.Do(ctx, info.Name, b)
		if err != nil {
			result := mcp.NewToolResultError(err.Error())
			var changed *app.InputChangedError
			if errors.As(err, &changed) {
				result = mcp.NewToolResultError(changed.Error())
				result.StructuredContent = map[string]any{"error": changed}
			}
			return result, nil
		}
		if studio, ok := v.(app.StudioResult); ok {
			return studioResult(ctx, studio), nil
		}
		if preview, ok := v.(app.PreviewResult); ok {
			metadata := map[string]any{"path": preview.Path, "width": preview.Width, "height": preview.Height, "mime_type": preview.MIMEType}
			b, _ := json.Marshal(metadata)
			result := mcp.NewToolResultStructured(metadata, string(b))
			result.Content = append(result.Content, mcp.NewImageContent(preview.Data, preview.MIMEType))
			return result, nil
		}
		text, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("encode result: %w", err)
		}
		return mcp.NewToolResultStructured(v, string(text)), nil
	})
}

const guide = `Still-image inputs normalize EXIF orientation and supported color declarations to upright sRGB before cropping, resizing, and palette extraction. Image masks use the same normalization. Inspect normalization metadata and use upright crop coordinates. Unsupported profiles return actionable errors. dither_catalog media.still.normalization lists profile support and limits.

Use dither_studio to explore a recipe in memory. MCP Apps hosts can display its optional interactive panel. Other hosts receive a PNG and structured recipe. Studio previews fit within 1024 pixels per side and default to 512. Saving is a separate dither_render call with a new output path. Replay the preview recipe with expected_source_sha256 from source_sha256 and, for an image mask, expected_mask_sha256 from mask_sha256. To export at another size, change only options.width and options.height. The studio's source_width and source_height describe the upright source after cropping. Export dimensions must fit export_limits. A source_changed or mask_changed error requires a fresh preview before export.

Paths are relative to the configured workspace. The service uses local files without URL imports, uploads, telemetry, or remote services. dither_preview sends image content to the MCP client. The client's data policy applies separately.

Choose a new output name if the path already exists. Place engine settings under options. Select a palette name or supply colors as a hex array. Omit adjustments to use neutral defaults. Explicit zero remains meaningful. A saved recipe must be used without inline options, palette, or colors.

Masks use the reduced pixel grid. Post-effects may produce colors outside the palette. The engine preserves alpha. JPEG and PBM cannot represent alpha. GIF uses binary alpha.

Use dither_palettes query, category, and color-count filters to discover a palette. Use next_offset with the same filters for more matches. Consult dither_catalog for actual IDs and budgets. Preview results to assess quality.

Calls run synchronously within defined limits. If the service observes cancellation before publication, it leaves no output. A batch publishes each successful item separately.`

type colorList struct {
	Colors []string `json:"colors"`
}
type previewMetadata struct {
	Path     string `json:"path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	MIMEType string `json:"mime_type"`
}
