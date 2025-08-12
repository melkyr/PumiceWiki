package handler

import (
	"context"
	"errors"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockPageService struct {
	ViewPageFunc   func(ctx context.Context, title string) (*data.Page, error)
	CreatePageFunc func(ctx context.Context, title, content, authorID string) (*data.Page, error)
	UpdatePageFunc func(ctx context.Context, id int64, title, content string) (*data.Page, error)
	GetAllPagesFunc func(ctx context.Context) ([]*data.Page, error)
}

func (m *mockPageService) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	return m.GetAllPagesFunc(ctx)
}

func (m *mockPageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	return m.ViewPageFunc(ctx, title)
}

func (m *mockPageService) CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error) {
	return m.CreatePageFunc(ctx, title, content, authorID)
}

func (m *mockPageService) UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error) {
	return m.UpdatePageFunc(ctx, id, title, content)
}

func TestViewHandler_Welcome(t *testing.T) {
	// Create a mock page service that returns an error when ViewPage is called
	pageService := &mockPageService{
		ViewPageFunc: func(ctx context.Context, title string) (*data.Page, error) {
			return nil, errors.New("page not found")
		},
	}

	// Create a new view service
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		t.Fatalf("failed to create view service: %v", err)
	}

	// Create a new page handler
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)

	// Create a new request
	req := httptest.NewRequest("GET", "/view/Home", nil)

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Create a new chi router and add the view handler
	r := chi.NewRouter()
	r.Get("/view/{title}", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.viewHandler(w, r)
	})

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check that the status code is 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the welcome page is rendered
	if !strings.Contains(rr.Body.String(), "Welcome to Go Wiki!") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}

func TestListHandler(t *testing.T) {
	// Create a mock page service that returns a list of pages
	pageService := &mockPageService{
		GetAllPagesFunc: func(ctx context.Context) ([]*data.Page, error) {
			return []*data.Page{
				{Title: "Page 1"},
				{Title: "Page 2"},
			}, nil
		},
	}

	// Create a new view service
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		t.Fatalf("failed to create view service: %v", err)
	}

	// Create a new page handler
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)

	// Create a new request
	req := httptest.NewRequest("GET", "/list", nil)

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Create a new chi router and add the list handler
	r := chi.NewRouter()
	r.Get("/list", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.listHandler(w, r)
	})

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check that the status code is 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the list page is rendered with the correct pages
	body := rr.Body.String()
	if !strings.Contains(body, "Page 1") {
		t.Errorf("handler returned unexpected body: got %v", body)
	}
	if !strings.Contains(body, "Page 2") {
		t.Errorf("handler returned unexpected body: got %v", body)
	}
}

func TestViewHandler_ViewPage(t *testing.T) {
	// Create a mock page service that returns a page when ViewPage is called
	pageService := &mockPageService{
		ViewPageFunc: func(ctx context.Context, title string) (*data.Page, error) {
			return &data.Page{Title: "Test Page", Content: "Test Content"}, nil
		},
	}

	// Create a new view service
	viewService, err := view.New(web.TemplateFS)
	if err != nil {
		t.Fatalf("failed to create view service: %v", err)
	}

	// Create a new page handler
	log := logger.New(config.LogConfig{Level: "info"})
	pageHandler := NewPageHandler(pageService, viewService, log)

	// Create a new request
	req := httptest.NewRequest("GET", "/view/Test%20Page", nil)

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Create a new chi router and add the view handler
	r := chi.NewRouter()
	r.Get("/view/{title}", func(w http.ResponseWriter, r *http.Request) {
		pageHandler.viewHandler(w, r)
	})

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check that the status code is 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the view page is rendered
	if !strings.Contains(rr.Body.String(), "Test Page") {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
}
