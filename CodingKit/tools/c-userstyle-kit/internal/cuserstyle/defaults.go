package cuserstyle

import _ "embed"

//go:embed templates/c-kit.json
var defaultConfigJSON string

//go:embed templates/c-snippets.json
var defaultSnippetsJSON string
