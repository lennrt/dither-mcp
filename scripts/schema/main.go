// schema prints the MCP tool descriptions from this build.
package main

import (
	"encoding/json"
	"github.com/lennrt/dither-mcp/internal/app"
	"github.com/lennrt/dither-mcp/internal/mcpserver"
	"github.com/mark3labs/mcp-go/mcp"
	"os"
	"sort"
)

func main() {
	svc, err := app.New(".", false)
	if err != nil {
		panic(err)
	}
	defer svc.Close()
	s := mcpserver.New(svc)
	var list []mcp.Tool
	for _, tool := range s.ListTools() {
		list = append(list, tool.Tool)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err := e.Encode(list); err != nil {
		panic(err)
	}
}
