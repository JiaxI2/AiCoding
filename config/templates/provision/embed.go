// Package provisiontemplates owns the immutable documentation skeleton bundled
// into aicoding provision.
package provisiontemplates

import "embed"

// Files contains the target-relative docs tree under docs/.
//
//go:embed docs
var Files embed.FS
