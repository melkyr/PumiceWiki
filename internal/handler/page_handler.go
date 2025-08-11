package handler

import (
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
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
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	if title == "Home" {
		_, err := h.pageService.ViewPage(r.Context(), title)
		if err != nil {
			http.Redirect(w, r, "/edit/Home", http.StatusFound)
			return nil
		}
	}

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Page not found", Code: http.StatusNotFound}
	}

	data := map[string]interface{}{
		"Page": page,
	}
	if err := h.view.Render(w, r, "view.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render view", Code: http.StatusInternalServerError}
	}
	return nil
}

// editHandler displays the form for editing a page.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		page = &data.Page{Title: title}
	}

	data := map[string]interface{}{
		"Page": page,
	}
	if err := h.view.Render(w, r, "edit.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render edit page", Code: http.StatusInternalServerError}
	}
	return nil
}

// saveHandler handles the form submission for creating or updating a page.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	content := r.FormValue("content")
	userInfo := middleware.GetUserInfo(r.Context())
	authorID := userInfo.Subject

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		if page, err = h.pageService.CreatePage(r.Context(), title, content, authorID); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to create page", Code: http.StatusInternalServerError}
		}
	} else {
		page.Content = content
		page.UpdatedAt = time.Now()
		if _, err := h.pageService.UpdatePage(r.Context(), page.ID, page.Title, page.Content); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to update page", Code: http.StatusInternalServerError}
		}
	}

	// If it's an HTMX request, render the form again with a success message.
	if r.Header.Get("HX-Request") == "true" && !middleware.IsBasicMode(r.Context()) {
		data := map[string]interface{}{
			"Page":    page,
			"Message": "Saved!",
		}
		// Render the partial view for HTMX requests
		if err := h.view.Render(w, r, "htmx/edit_form.html", data); err !=.Nil() {
			return &middleware.AppError{Error: err, Message: "Failed to render HTMX view", Code: http.StatusInternalServerError}
		}
		return nil
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
	return nil
}
