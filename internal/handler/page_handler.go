package handler

import (
	"errors"
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
	pageService service.PageServicer
	view        *view.View
	log         logger.Logger
}

// NewPageHandler creates a new PageHandler with the given dependencies.
func NewPageHandler(ps service.PageServicer, v *view.View, log logger.Logger) *PageHandler {
	return &PageHandler{
		pageService: ps,
		view:        v,
		log:         log,
	}
}

// viewHandler retrieves the page title from the URL and renders the page.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	userInfo := middleware.GetUserInfo(r.Context())

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// If page is not found, check if it's the home page and user is anonymous
		if title == "Home" && userInfo.Subject == "anonymous" {
			if err := h.view.Render(w, r, "welcome.html", nil); err != nil {
				return &middleware.AppError{Error: err, Message: "Failed to render welcome page", Code: http.StatusInternalServerError}
			}
			return nil
		}

		// For authenticated users on home, or any user on a different non-existent page
		if title == "Home" {
			page = &data.Page{Title: "Home", Content: "Welcome! This page is empty."}
		} else {
			return &middleware.AppError{Error: err, Message: "Page not found", Code: http.StatusNotFound}
		}
	}

	data := map[string]interface{}{
		"Page":     page,
		"UserInfo": userInfo,
	}
	if err := h.view.Render(w, r, "view.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render view", Code: http.StatusInternalServerError}
	}
	return nil
}

// editHandler displays the form for editing a page.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	if title == "Home" {
		return &middleware.AppError{Error: errors.New("home page is not editable"), Message: "The Home page cannot be edited.", Code: http.StatusForbidden}
	}
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
func (h *PageHandler) listHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	pages, err := h.pageService.GetAllPages(r.Context())
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to retrieve pages", Code: http.StatusInternalServerError}
	}

	data := map[string]interface{}{
		"Pages": pages,
	}
	if err := h.view.Render(w, r, "list.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render list page", Code: http.StatusInternalServerError}
	}
	return nil
}

func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	originalTitle := chi.URLParam(r, "title")
	newTitle := r.FormValue("title")
	content := r.FormValue("content")
	userInfo := middleware.GetUserInfo(r.Context())
	authorID := userInfo.Subject

	// Check if we are creating a new page or updating an existing one
	page, err := h.pageService.ViewPage(r.Context(), originalTitle)
	if err != nil {
		// Create new page
		if page, err = h.pageService.CreatePage(r.Context(), newTitle, content, authorID); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to create page", Code: http.StatusInternalServerError}
		}
	} else {
		// Update existing page
		page.Content = content
		page.UpdatedAt = time.Now()
		if _, err := h.pageService.UpdatePage(r.Context(), page.ID, newTitle, page.Content); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to update page", Code: http.StatusInternalServerError}
		}
	}

	// For HTMX requests, redirect using a header
	if r.Header.Get("HX-Request") == "true" && !view.IsBasicMode(r.Context()) {
		w.Header().Set("HX-Redirect", "/view/"+newTitle)
		return nil
	}

	// For standard requests, use a normal redirect
	http.Redirect(w, r, "/view/"+newTitle, http.StatusFound)
	return nil
}
