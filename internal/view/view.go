package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
)

// View represents a collection of parsed HTML templates.
type View struct {
	templates map[string]*template.Template
}

// New creates a new View by parsing all templates from the given filesystem.
func New(templateFS fs.FS) (*View, error) {
	v := &View{
		templates: make(map[string]*template.Template),
	}

	// First, get all the layout files
	layouts, err := fs.Glob(templateFS, "templates/layouts/*.html")
	if err != nil {
		return nil, err
	}

	// Then, get all the page files
	pages, err := fs.Glob(templateFS, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}

	// For each page, parse it with the layout files
	for _, page := range pages {
		files := append(layouts, page)
		// The name of the template is the base name of the page file
		name := filepath.Base(page)
		// Parse the files
		ts, err := template.New(name).ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		v.templates[name] = ts
	}

	return v, nil
}

import "go-wiki-app/internal/middleware"
// Render executes a specific template by name.
func (v *View) Render(w io.Writer, r *http.Request, name string, data map[string]interface{}) error {
	ts, ok := v.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	// Add the IsBasicMode flag to the data map.
	if data == nil {
		data = make(map[string]interface{})
	}
	data["IsBasicMode"] = middleware.IsBasicMode(r.Context())

	// Execute the template into a buffer first to catch any errors
	// before writing to the response writer.
	buf := new(bytes.Buffer)
	err := ts.Execute(buf, data)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}
