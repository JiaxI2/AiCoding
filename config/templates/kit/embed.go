// Package kittemplates owns the immutable scaffolds bundled into aicoding kit init.
package kittemplates

import "embed"

// Files contains the first-party and external-wrapper Kit templates.
//
//go:embed *.tmpl.json *.tmpl.md
var Files embed.FS
