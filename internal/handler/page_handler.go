package handler

import (
	"fmt"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PageHandler holds the dependencies for the page handlers.
type PageHandler struct {
	pageService *service.PageService
	view        *view.View
	log         logger.Logger
}

// NewPageHandler creates a new PageHandler with the given dependencies.
func NewPageHandler(ps *service.PageService, v *view.View, log logger.Logger) *PageHandler {
	return &PageHandler{
		pageService: ps,
		view:        v,
		log:         log,
	}
}

// viewHandler retrieves the page title from the URL and renders the page.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		h.log.Warn(fmt.Sprintf("Page not found, redirecting to edit: %s", title))
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Page": page,
	}
	h.view.Render(w, "view.html", data)
}

// editHandler displays the form for editing a page.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// Page doesn't exist, create a new Page struct for the template.
		page = &data.Page{Title: title}
	}

	data := map[string]interface{}{
		"Page": page,
	}
	h.view.Render(w, "edit.html", data)
}

// saveHandler handles the form submission for creating or updating a page.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) {
	title := chi.URLParam(r, "title")
	content := r.FormValue("content")
	// In a real app, authorID would come from the user's session in the context.
	authorID := "anonymous"

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// Page doesn't exist, create it.
		if _, err := h.pageService.CreatePage(r.Context(), title, content, authorID); err != nil {
			h.log.Error(err, "Failed to create page")
			http.Error(w, "Failed to save page", http.StatusInternalServerError)
			return
		}
	} else {
		// Page exists, update it.
		page.Content = content
		page.UpdatedAt = time.Now()
		if _, err := h.pageService.UpdatePage(r.Context(), page.ID, page.Title, page.Content); err != nil {
			h.log.Error(err, "Failed to update page")
			http.Error(w, "Failed to save page", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
