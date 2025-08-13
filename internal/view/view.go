package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
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

	// Walk the templates/pages directory to find all page templates recursively
	var pages []string
	err = fs.WalkDir(templateFS, "templates/pages", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".html") {
			pages = append(pages, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk page templates: %w", err)
	}

	// For each page, parse it with the layout files
	for _, page := range pages {
		files := append(layouts, page)

		// The name of the template is its path relative to "templates/"
		// e.g., "pages/view.html" or "pages/htmx/category_search_results.html"
		name, err := filepath.Rel("templates", page)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", page, err)
		}

		// The name passed to template.New() becomes the name of the template,
		// which is how we refer to it when we want to execute a specific one.
		// We use the base name here so that in the template files, we can just
		// define the content block, and it will be merged with the base layout.
		ts, err := template.New(filepath.Base(page)).ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		// But we store it in the map with its full relative path name.
		v.templates[name] = ts
	}

	return v, nil
}

// Render executes a specific template by name.
func (v *View) Render(w io.Writer, r *http.Request, name string, data map[string]interface{}) error {
	ts, ok := v.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	// Set the Content-Type header to ensure middleware like compression works correctly.
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

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
