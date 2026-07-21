// Package mcptemplates owns the immutable component manifest scaffold bundled
// into aicoding mcp init.
package mcptemplates

import "embed"

// Files contains the MCP component manifest template.
//
//go:embed component.tmpl.json
var Files embed.FS
