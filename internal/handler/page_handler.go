package handler

import (
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PageHandler holds the dependencies for the page handlers.
type PageHandler struct {
	pageService *service.PageService
	view        *view.View
	logger      *log.Logger
}

// NewPageHandler creates a new PageHandler with the given dependencies.
func NewPageHandler(ps *service.PageService, v *view.View, l *log.Logger) *PageHandler {
	return &PageHandler{
		pageService: ps,
		view:        v,
		logger:      l,
	}
}

// viewHandler retrieves the page title from the URL, loads the page data via the service,
// and renders a template to display the page.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Page": page,
	}
	h.view.Render(w, "view.html", data)
}

// editHandler loads a page and displays it in an HTML form for editing.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		page = &data.Page{Title: title}
	}

	data := map[string]interface{}{
		"Page": page,
	}
	h.view.Render(w, "edit.html", data)
}

// saveHandler handles the submission of the edit form.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	content := r.FormValue("content")
	authorID := "anonymous" // This will be replaced by user from context later

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// Page doesn't exist, create it.
		if _, err := h.pageService.CreatePage(r.Context(), title, content, authorID); err != nil {
			h.logger.Printf("Error creating page '%s': %v", title, err)
			http.Error(w, "Failed to save page", http.StatusInternalServerError)
			return
		}
	} else {
		// Page exists, update it.
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
