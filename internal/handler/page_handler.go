package handler

import (
	"fmt"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/service"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PageHandler holds the dependencies for the page handlers.
// It depends on the PageService to perform business logic.
type PageHandler struct {
	pageService *service.PageService
	logger      *log.Logger
}

// NewPageHandler creates a new PageHandler with the given dependencies.
func NewPageHandler(ps *service.PageService, l *log.Logger) *PageHandler {
	return &PageHandler{
		pageService: ps,
		logger:      l,
	}
}

// viewHandler retrieves the page title from the URL, loads the page data via the service,
// and renders a template to display the page.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		h.logger.Printf("Error loading page '%s': %v", title, err)
		// If page not found, redirect to the edit page to allow creation.
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>%s</h1>", page.Title)
	fmt.Fprintf(w, "<div>%s</div>", page.Content)
	fmt.Fprintf(w, `<p><a href="/edit/%s">Edit this page</a></p>`, page.Title)
}

// editHandler loads a page and displays it in an HTML form for editing.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// Page doesn't exist, create a new Page struct for the template.
		page = &data.Page{Title: title}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Editing %s</h1>
<form action="/save/%s" method="POST">
	<textarea name="content" rows="20" cols="80">%s</textarea><br>
	<input type="submit" value="Save">
</form>`, page.Title, page.Title, page.Content)
}

// saveHandler handles the submission of the edit form.
// It determines whether to create a new page or update an existing one.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// For now, hardcode the author. This will come from the user's session later.
	authorID := "anonymous"

	// Check if the page already exists.
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// Page doesn't exist, so create it by calling the service.
		if _, err := h.pageService.CreatePage(r.Context(), title, content, authorID); err != nil {
			h.logger.Printf("Error creating page '%s': %v", title, err)
			http.Error(w, "Failed to save page", http.StatusInternalServerError)
			return
		}
	} else {
		// Page exists, so update it by calling the service.
		page.Content = content
		page.UpdatedAt = time.Now()
		if _, err := h.pageService.UpdatePage(r.Context(), page.ID, page.Title, page.Content); err != nil {
			h.logger.Printf("Error updating page '%s': %v", title, err)
			http.Error(w, "Failed to save page", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
