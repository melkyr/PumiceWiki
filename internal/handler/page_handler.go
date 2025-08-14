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
func (h *PageHandler) viewHandler(w http.ResponseWriter, r *http.Request) *middleware.AppError {
	title := chi.URLParam(r, "title")
	userInfo := middleware.GetUserInfo(r.Context())

	page, err := h.pageService.ViewPage(r.Context(), title)
	if err != nil {
		if title == "Home" {
			templateData := map[string]interface{}{"IsBasicMode": middleware.IsBasicMode(r.Context())}
			if userInfo.Subject == "anonymous" {
				if err := h.view.Render(w, r, "pages/welcome.html", templateData); err != nil {
					return &middleware.AppError{Error: err, Message: "Failed to render welcome page", Code: http.StatusInternalServerError}
				}
				return nil
			}
			page = &data.Page{
				Title:       "Home",
				Content:     "Welcome! This page is empty.",
				HTMLContent: template.HTML("<p>Welcome! This page is empty.</p>"),
			}
		} else {
			return &middleware.AppError{Error: err, Message: "Page not found", Code: http.StatusNotFound}
		}
	}

	templateData := map[string]interface{}{
		"Page":        page,
		"UserInfo":    userInfo,
		"IsBasicMode": middleware.IsBasicMode(r.Context()),
	}
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
	if err != nil {
		page = &data.Page{Title: title}
	}

	templateData := map[string]interface{}{
		"Page":        page,
		"UserInfo":    middleware.GetUserInfo(r.Context()),
		"IsBasicMode": middleware.IsBasicMode(r.Context()),
	}
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
	templateData := map[string]interface{}{
		"Pages":        pages,
		"UserInfo":     middleware.GetUserInfo(r.Context()),
		"CategoryTree": categoryTree,
		"IsBasicMode":  middleware.IsBasicMode(r.Context()),
	}
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
	templateData := map[string]interface{}{
		"Categories":  categories,
		"IsBasicMode": middleware.IsBasicMode(r.Context()),
	}
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

	page, err := h.pageService.ViewPage(r.Context(), originalTitle)
	if err != nil {
		if _, err = h.pageService.CreatePage(r.Context(), newTitle, content, authorID, category, subcategory); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to create page", Code: http.StatusInternalServerError}
		}
	} else {
		if _, err := h.pageService.UpdatePage(r.Context(), page.ID, newTitle, content, category, subcategory); err != nil {
			return &middleware.AppError{Error: err, Message: "Failed to update page", Code: http.StatusInternalServerError}
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
	templateData := map[string]interface{}{
		"Title":       "Category: " + categoryName,
		"Pages":       pages,
		"UserInfo":    middleware.GetUserInfo(r.Context()),
		"IsBasicMode": middleware.IsBasicMode(r.Context()),
	}
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
	templateData := map[string]interface{}{
		"UserInfo":     middleware.GetUserInfo(r.Context()),
		"CategoryTree": categoryTree,
		"IsBasicMode":  middleware.IsBasicMode(r.Context()),
	}
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
	templateData := map[string]interface{}{
		"Title":       "Category: " + categoryName + " / " + subcategoryName,
		"Pages":       pages,
		"UserInfo":    middleware.GetUserInfo(r.Context()),
		"IsBasicMode": middleware.IsBasicMode(r.Context()),
	}
	if err := h.view.Render(w, r, "pages/category_view.html", templateData); err != nil {
		return &middleware.AppError{Error: err, Message: "Failed to render category view", Code: http.StatusInternalServerError}
	}
	return nil
}
