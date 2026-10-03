package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestStdioProtocolEras(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			svc, err := app.New(t.TempDir(), false)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			input, send := io.Pipe()
			receive, output := io.Pipe()
			defer send.Close()
			defer receive.Close()
			defer input.Close()
			defer output.Close()
			stop := context.AfterFunc(ctx, func() {
				_ = send.CloseWithError(ctx.Err())
				_ = receive.CloseWithError(ctx.Err())
			})
			defer stop()
			go func() { _ = server.NewStdioServer(New(svc)).Listen(ctx, input, output) }()
			reader := bufio.NewReader(receive)
			id := 0
			rpc := func(method string, params map[string]any) map[string]any {
				t.Helper()
				id++
				if version == "2026-07-28" {
					params["_meta"] = map[string]any{
						mcp.MetaKeyProtocolVersion:    version,
						mcp.MetaKeyClientInfo:         map[string]any{"name": "dither-wire-test", "version": "1"},
						mcp.MetaKeyClientCapabilities: map[string]any{},
						"allow_video":                 true,
					}
				}
				b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := send.Write(append(b, '\n')); err != nil {
					t.Fatal(err)
				}
				line, err := reader.ReadBytes('\n')
				if err != nil {
					t.Fatal(err)
				}
				var result map[string]any
				if err := json.Unmarshal(line, &result); err != nil {
					t.Fatalf("stdout contains non-JSON protocol output: %s", line)
				}
				if result["id"] != float64(id) {
					t.Fatalf("response ID mismatch: %v", result)
				}
				return result
			}
			if version == "2026-07-28" {
				discovered := rpc("server/discover", map[string]any{})
				result, ok := discovered["result"].(map[string]any)
				if !ok || result["resultType"] != "complete" || result["_meta"] == nil {
					t.Fatalf("modern discovery lacks result metadata: %v", discovered)
				}
				versions, ok := result["supportedVersions"].([]any)
				if !ok || len(versions) == 0 || versions[0] != version || result["instructions"] == nil {
					t.Fatalf("discovery capabilities: %v", result)
				}
			} else {
				init := rpc("initialize", map[string]any{"protocolVersion": version, "clientInfo": map[string]any{"name": "legacy-wire-test", "version": "1"}, "capabilities": map[string]any{}})
				if init["error"] != nil {
					t.Fatalf("legacy initialization: %v", init)
				}
				if _, err := send.Write([]byte("{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n")); err != nil {
					t.Fatal(err)
				}
			}
			listed := rpc("tools/list", map[string]any{})
			result := listed["result"].(map[string]any)
			tools := result["tools"].([]any)
			if len(tools) != 13 {
				t.Fatalf("tools=%d", len(tools))
			}
			if version == "2026-07-28" && result["resultType"] != "complete" {
				t.Fatal("modern tools/list lacks resultType")
			}
			called := rpc("tools/call", map[string]any{"name": "dither_catalog", "arguments": map[string]any{}})
			result = called["result"].(map[string]any)
			structured := result["structuredContent"].(map[string]any)
			if structured["video_enabled"] != false || structured["media"] == nil {
				t.Fatal("request metadata changed service settings or media discovery is absent")
			}
			content := result["content"].([]any)
			text := content[0].(map[string]any)["text"].(string)
			var fallback map[string]any
			if err := json.Unmarshal([]byte(text), &fallback); err != nil || fallback["media"] == nil {
				t.Fatal("structured results need serialized JSON text for older clients")
			}
			invalid := rpc("tools/call", map[string]any{"name": "dither_catalog", "arguments": map[string]any{"extra": true}})
			if invalid["result"].(map[string]any)["isError"] != true {
				t.Fatal("argument validation must produce a tool execution error")
			}
			unknown := rpc("tools/call", map[string]any{"name": "missing_tool", "arguments": map[string]any{}})
			if unknown["error"] == nil {
				t.Fatal("an unknown tool must produce a protocol error")
			}
		})
	}
}
