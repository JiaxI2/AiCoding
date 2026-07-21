// Package skilltemplates owns the immutable external Skill scaffold bundled
// into aicoding skill init. The template is not an AiCoding Skill source.
package skilltemplates

import "embed"

// Files contains the runtime-neutral Skill authoring template.
//
//go:embed SKILL.tmpl
var Files embed.FS
