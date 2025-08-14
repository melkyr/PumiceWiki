package handler

import (
	"errors"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"net/http"

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

// newTemplateData creates a map for template data and pre-populates it with common data.
func newTemplateData(r *http.Request) map[string]interface{} {
	data := make(map[string]interface{})
	data["UserInfo"] = middleware.GetUserInfo(r.Context())
	data["IsBasicMode"] = middleware.IsBasicMode(r.Context())
	return data
}

// viewHandler handles requests to view a wiki page.
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	templateData := newTemplateData(r)

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		if errors.Is(err, service.ErrAnonymousHome) {
			if err := h.view.Render(w, r, "pages/welcome.html", templateData); err != nil {
				return &middleware.AppError{Error: err, Message: "Failed to render welcome page", Code: http.StatusInternalServerError}
			}
			return nil
		}
		return &middleware.AppError{Error: err, Message: "Page not found", Code: http.StatusNotFound}
	}

	templateData["Page"] = page
	if err := h.view.Render(w, r, "pages/view.html", templateData); err != nil {
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
	// An error is expected if the page doesn't exist yet. We create a new page object in that case.
	if err != nil {
		// We don't want to show an edit page for the anonymous-home-page case.
		if errors.Is(err, service.ErrAnonymousHome) {
			return &middleware.AppError{Error: err, Message: "Page not found", Code: http.StatusNotFound}
		}
		page = &data.Page{Title: title}
	}

	templateData := newTemplateData(r)
	templateData["Page"] = page
	if err := h.view.Render(w, r, "pages/edit.html", templateData); err != nil {
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
	categoryTree, err := h.pageService.GetCategoryTree(r.Context())
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to retrieve category tree", Code: http.StatusInternalServerError}
	}
	templateData := newTemplateData(r)
	templateData["Pages"] = pages
	templateData["CategoryTree"] = categoryTree
	if err := h.view.Render(w, r, "pages/list.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render list page", Code: http.StatusInternalServerError}
	}
	return nil
}

// searchCategoriesHandler handles API requests to search for categories.
func (h *PageHandler) searchCategoriesHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	query := r.URL.Query().Get("q")
	categories, err := h.pageService.SearchCategories(r.Context(), query)
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to search for categories", Code: http.StatusInternalServerError}
	}
	templateData := newTemplateData(r)
	templateData["Categories"] = categories
	if err := h.view.Render(w, r, "pages/htmx/category_search_results.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render search results", Code: http.StatusInternalServerError}
	}
	return nil
}

// saveHandler handles form submissions from the edit page.
func (h *PageHandler) saveHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	originalTitle := chi.URLParam(r, "title")
	newTitle := r.FormValue("title")
	content := r.FormValue("content")
	category := r.FormValue("category")
	subcategory := r.FormValue("subcategory")
	authorID := middleware.GetUserInfo(r.Context()).Subject

	// Server-side validation to prevent editing "Home" page
	if originalTitle == "Home" || newTitle == "Home" {
		return &middleware.AppError{Error: errors.New("home page is not editable"), Message: "The Home page cannot be edited.", Code: http.StatusForbidden}
	}

	page, err := h.pageService.ViewPage(r.Context(), originalTitle)
	if err != nil {
		// If the page does not exist (and it's not the special anonymous home case), create it.
		if !errors.Is(err, service.ErrAnonymousHome) {
			if _, createErr := h.pageService.CreatePage(r.Context(), newTitle, content, authorID, category, subcategory); createErr != nil {
				return &middleware.AppError{Error: createErr, Message: "Failed to create page", Code: http.StatusInternalServerError}
			}
		} else {
			// This case indicates trying to save a page from a state that shouldn't be possible (e.g., anonymous user on home).
			return &middleware.AppError{Error: err, Message: "Cannot create page from this state", Code: http.StatusBadRequest}
		}
	} else {
		// If the page exists, update it.
		// The page object from ViewPage will have the ID we need.
		if _, updateErr := h.pageService.UpdatePage(r.Context(), page.ID, newTitle, content, category, subcategory); updateErr != nil {
			return &middleware.AppError{Error: updateErr, Message: "Failed to update page", Code: http.StatusInternalServerError}
		}
	}

	if r.Header.Get("HX-Request") == "true" && !middleware.IsBasicMode(r.Context()) {
		w.Header().Set("HX-Redirect", "/view/"+newTitle)
		return nil
	}

	http.Redirect(w, r, "/view/"+newTitle, http.StatusFound)
	return nil
}

func (h *PageHandler) viewByCategoryHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	categoryName := chi.URLParam(r, "categoryName")
	pages, err := h.pageService.GetPagesForCategory(r.Context(), categoryName)
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to get pages for category", Code: http.StatusNotFound}
	}
	templateData := newTemplateData(r)
	templateData["Title"] = "Category: " + categoryName
	templateData["Pages"] = pages
	if err := h.view.Render(w, r, "pages/category_view.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render category view", Code: http.StatusInternalServerError}
	}
	return nil
}

func (h *PageHandler) categoriesHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	categoryTree, err := h.pageService.GetCategoryTree(r.Context())
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to retrieve category tree", Code: http.StatusInternalServerError}
	}
	templateData := newTemplateData(r)
	templateData["CategoryTree"] = categoryTree
	if err := h.view.Render(w, r, "pages/categories.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render categories page", Code: http.StatusInternalServerError}
	}
	return nil
}

func (h *PageHandler) viewBySubcategoryHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	categoryName := chi.URLParam(r, "categoryName")
	subcategoryName := chi.URLParam(r, "subcategoryName")
	pages, err := h.pageService.GetPagesForSubcategory(r.Context(), categoryName, subcategoryName)
	if err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to get pages for subcategory", Code: http.StatusNotFound}
	}
	templateData := newTemplateData(r)
	templateData["Title"] = "Category: " + categoryName + " / " + subcategoryName
	templateData["Pages"] = pages
	if err := h.view.Render(w, r, "pages/category_view.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render category view", Code: http.StatusInternalServerError}
	}
	return nil
}
