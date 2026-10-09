// Package mcpapp embeds the self-contained MCP Apps studio resource.
// Node and development dependencies are not needed to run the Go binary.
package mcpapp

import _ "embed"

const URI = "ui://dither/studio.html"
const MIMEType = "text/html;profile=mcp-app"

//go:embed studio.html
var studioHTML string

// HTML returns the studio's single-file HTML resource.
func HTML() string { return studioHTML }
