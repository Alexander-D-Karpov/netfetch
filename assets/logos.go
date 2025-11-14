package assets

import "embed"

//go:embed logos/*.json templates/*.html
var embeddedFS embed.FS

var LogosFS = embeddedFS

var TemplatesFS = embeddedFS
