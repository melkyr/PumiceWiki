package web

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var templateFS embed.FS

//go:embed all:static
var staticFS embed.FS

// TemplateFS provides access to the embedded template files.
var TemplateFS fs.FS = templateFS

// StaticFS provides access to the embedded static asset files.
var StaticFS fs.FS = staticFS
