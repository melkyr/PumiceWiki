package handler

import (
	"errors"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"html/template"
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

// viewHandler handles requests to view a wiki page.
// It retrieves the page from the database and renders it using the "view.html" template.
// It includes special logic for the "Home" page to handle cases where it doesn't exist yet.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	userInfo := middleware.GetUserInfo(r.Context())

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		// If the page doesn't exist, we have special handling for the "Home" page.
		if title == "Home" {
			// If the user is anonymous, show the public welcome page.
			if userInfo.Subject == "anonymous" {
				if err := h.view.Render(w, r, "welcome.html", nil); err != nil {
					return &middleware.AppError{Error: err, Message: "Failed to render welcome page", Code: http.StatusInternalServerError}
				}
				return nil
			}
			// If the user is authenticated, show a default, non-db-backed "Home" page.
			// This ensures they see the standard layout and editor controls.
			page = &data.Page{
				Title:       "Home",
				Content:     "Welcome! This page is empty.",
				HTMLContent: template.HTML("<p>Welcome! This page is empty.</p>"),
			}
		} else {
			// For any other page that doesn't exist, return a 404 error.
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

// editHandler displays the form for editing a page. If the page does not exist,
// it pre-populates the form with the title from the URL, allowing for page creation.
// It explicitly forbids editing of the "Home" page.
func (h *PageHandler) editHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	// The "Home" page is special and cannot be edited through the standard editor.
	if title == "Home" {
		return &middleware.AppError{Error: errors.New("home page is not editable"), Message: "The Home page cannot be edited.", Code: http.StatusForbidden}
	}
	// Attempt to load the page. If it doesn't exist, create a new Page struct
	// to pass to the template for creation.
	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		page = &data.Page{Title: title}
	}

	userInfo := middleware.GetUserInfo(r.Context())
	data := map[string]interface{}{
		"Page":     page,
		"UserInfo": userInfo,
	}
	if err := h.view.Render(w, r, "edit.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render edit page", Code: http.StatusInternalServerError}
	}
	return nil
}

// listHandler displays a list of all pages in the wiki.
func (h *PageHandler) listHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	pages, err := h.pageService.GetAllPages(r.Context())
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to retrieve pages", Code: http.StatusInternalServerError}
	}

	userInfo := middleware.GetUserInfo(r.Context())
	data := map[string]interface{}{
		"Pages":    pages,
		"UserInfo": userInfo,
	}
	if err := h.view.Render(w, r, "list.html", data); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render list page", Code: http.StatusInternalServerError}
	}
	return nil
}

// saveHandler handles form submissions from the edit page. It can either create a new
// page or update an existing one. It distinguishes between create and update by
// checking if a page with the original title (from the URL) exists.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	originalTitle := chi.URLParam(r, "title")
	newTitle := r.FormValue("title")
	content := r.FormValue("content")
	userInfo := middleware.GetUserInfo(r.Context())
	authorID := userInfo.Subject

	// Attempt to load the page by its original title to see if it exists.
	page, err := h.pageService.ViewPage(r.Context(), originalTitle)
	if err != nil {
		// If it doesn't exist, create a new page with the title from the form.
		if _, err = h.pageService.CreatePage(r.Context(), newTitle, content, authorID); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to create page", Code: http.StatusInternalServerError}
		}
	} else {
		// If it exists, update it with the new title and content from the form.
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
