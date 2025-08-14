package handler

import (
	"context"
	"errors"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockPageService struct {
	ViewPageFunc           func(ctx context.Context, title string) (*data.Page, error)
	CreatePageFunc         func(ctx context.Context, title, content, authorID, categoryName, subcategoryName string) (*data.Page, error)
	UpdatePageFunc         func(ctx context.Context, id int64, title, content, categoryName, subcategoryName string) (*data.Page, error)
	GetAllPagesFunc        func(ctx context.Context) ([]*data.Page, error)
	DeletePageFunc         func(ctx context.Context, id int64) error
	GetCategoryTreeFunc    func(ctx context.Context) ([]*service.CategoryNode, error)
	SearchCategoriesFunc   func(ctx context.Context, query string) ([]*data.Category, error)
	GetPagesForCategoryFunc func(ctx context.Context, categoryName string) ([]*data.Page, error)
	GetPagesForSubcategoryFunc func(ctx context.Context, categoryName string, subcategoryName string) ([]*data.Page, error)
}

func (m *mockPageService) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	return m.GetAllPagesFunc(ctx)
}

func (m *mockPageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	return m.ViewPageFunc(ctx, title)
}

func (m *mockPageService) CreatePage(ctx context.Context, title, content, authorID, categoryName, subcategoryName string) (*data.Page, error) {
	return m.CreatePageFunc(ctx, title, content, authorID, categoryName, subcategoryName)
}

func (m *mockPageService) UpdatePage(ctx context.Context, id int64, title, content, categoryName, subcategoryName string) (*data.Page, error) {
	return m.UpdatePageFunc(ctx, id, title, content, categoryName, subcategoryName)
}

func (m *mockPageService) DeletePage(ctx context.Context, id int64) error {
	return m.DeletePageFunc(ctx, id)
}

func (m *mockPageService) GetCategoryTree(ctx context.Context) ([]*service.CategoryNode, error) {
	return m.GetCategoryTreeFunc(ctx)
}

func (m *mockPageService) SearchCategories(ctx context.Context, query string) ([]*data.Category, error) {
	return m.SearchCategoriesFunc(ctx, query)
}

func (m *mockPageService) GetPagesForCategory(ctx context.Context, categoryName string) ([]*data.Page, error) {
	if m.GetPagesForCategoryFunc != nil {
		return m.GetPagesForCategoryFunc(ctx, categoryName)
	}
	return nil, nil
}

func (m *mockPageService) GetPagesForSubcategory(ctx context.Context, categoryName string, subcategoryName string) ([]*data.Page, error) {
	if m.GetPagesForSubcategoryFunc != nil {
		return m.GetPagesForSubcategoryFunc(ctx, categoryName, subcategoryName)
	}
	return nil, nil
}

func TestViewHandler_Welcome(t *testing.T) {
	pageService := &mockPageService{
		ViewPageFunc: func(ctx context.Context, title string) (*data.Page, error) {
			return nil, errors.New("page not found")
		},
	}
	viewService, _ := view.New(web.TemplateFS)
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)
	req := httptest.NewRequest("GET", "/view/Home", nil)
	rr := httptest.NewRecorder()
	r := chi.NewRouter()
	r.Get("/view/{title}", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.viewHandler(w, r)
	})
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "Welcome to Go Wiki!") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestListHandler(t *testing.T) {
	pageService := &mockPageService{
		GetAllPagesFunc: func(ctx context.Context) ([]*data.Page, error) {
			return []*data.Page{{Title: "Page 1"}, {Title: "Page 2"}}, nil
		},
		GetCategoryTreeFunc: func(ctx context.Context) ([]*service.CategoryNode, error) {
			return []*service.CategoryNode{}, nil // Return empty tree for this test
		},
	}
	viewService, _ := view.New(web.TemplateFS)
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)
	req := httptest.NewRequest("GET", "/list", nil)
	rr := httptest.NewRecorder()
	r := chi.NewRouter()
	r.Get("/list", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.listHandler(w, r)
	})
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Page 1") {
		t.Errorf("handler returned unexpected body: got %v", body)
	}
}

func TestViewHandler_ViewPage(t *testing.T) {
	pageService := &mockPageService{
		ViewPageFunc: func(ctx context.Context, title string) (*data.Page, error) {
			return &data.Page{Title: "Test Page", Content: "Test Content"}, nil
		},
	}
	viewService, _ := view.New(web.TemplateFS)
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)
	req := httptest.NewRequest("GET", "/view/Test%20Page", nil)
	rr := httptest.NewRecorder()
	r := chi.NewRouter()
	r.Get("/view/{title}", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.viewHandler(w, r)
	})
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "Test Page") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}
