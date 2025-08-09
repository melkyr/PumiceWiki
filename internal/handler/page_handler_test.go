//go:build integration

package handler

import (
	"context"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/service"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// setupTest initializes a full application stack and returns the router, the page repository, and a teardown function.
func setupTest(t *testing.T) (*chi.Mux, service.PageRepository, func()) {
	t.Helper()
	dsn := "file:memory?mode=memory&cache=shared"
	db, err := data.NewDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Manually apply migrations for the test database.
	schema1, err := os.ReadFile("../../migrations/001_initial_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read migration file 1: %v", err)
	}
	db.MustExec(string(schema1))

	schema2, err := os.ReadFile("../../migrations/002_create_casbin_rule_table.sql")
	if err != nil {
		t.Fatalf("Failed to read migration file 2: %v", err)
	}
	db.MustExec(string(schema2))

	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)

	// For tests, we don't have a real OIDC provider, so we pass nil for the authenticator.
	// This means we can't test the auth handlers directly in this suite.
	// We also pass a nil middleware function.
	// In a real app, we might create a mock authenticator and middleware.
	pageHandler := NewPageHandler(pageService, logger)
	router := NewRouter(pageHandler, nil, func(next http.Handler) http.Handler {
		return next // No-op middleware for tests
	})

	teardown := func() {
		db.Close()
	}
	return router, pageRepository, teardown
}

// TestSetup is a simple test to ensure that the test environment can be initialized without errors.
func TestSetup(t *testing.T) {
	router, _, teardown := setupTest(t)
	defer teardown()

	if router == nil {
		t.Fatal("Expected router to be not nil")
	}
	if len(router.Routes()) == 0 {
		t.Fatal("Expected router to have routes registered")
	}
}

// TestViewPageHandler tests the happy path and not-found case for the view handler.
func TestViewPageHandler(t *testing.T) {
	router, repo, teardown := setupTest(t)
	defer teardown()

	t.Run("existing page", func(t *testing.T) {
		testPage := &data.Page{Title: "TestPage", Content: "This is the content.", AuthorID: "tester"}
		if err := repo.CreatePage(context.Background(), testPage); err != nil {
			t.Fatalf("Failed to create test page: %v", err)
		}
		req := httptest.NewRequest("GET", "/view/TestPage", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
		}
		body, err := io.ReadAll(rr.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		if !strings.Contains(string(body), "<h1>TestPage</h1>") {
			t.Errorf("response body does not contain the page title")
		}
		if !strings.Contains(string(body), "<div>This is the content.</div>") {
			t.Errorf("response body does not contain the page content")
		}
	})

	t.Run("non-existent page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/view/NotFoundPage", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusFound)
		}
		location, err := rr.Result().Location()
		if err != nil {
			t.Fatalf("Failed to get redirect location: %v", err)
		}
		if location.Path != "/edit/NotFoundPage" {
			t.Errorf("handler redirected to wrong location: got %s want /edit/NotFoundPage", location.Path)
		}
	})
}

func TestSavePageHandler(t *testing.T) {
	router, repo, teardown := setupTest(t)
	defer teardown()

	t.Run("create new page", func(t *testing.T) {
		form := url.Values{}
		form.Add("content", "This is a new page.")
		req := httptest.NewRequest("POST", "/save/NewPage", strings.NewReader(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusFound)
		}
		page, err := repo.GetPageByTitle(context.Background(), "NewPage")
		if err != nil {
			t.Fatalf("Failed to get created page from repo: %v", err)
		}
		if page.Content != "This is a new page." {
			t.Errorf("page content is incorrect: got '%s' want '%s'", page.Content, "This is a new page.")
		}
	})

	t.Run("update existing page", func(t *testing.T) {
		existingPage := &data.Page{Title: "ExistingPage", Content: "Original content.", AuthorID: "tester"}
		if err := repo.CreatePage(context.Background(), existingPage); err != nil {
			t.Fatalf("Failed to create test page: %v", err)
		}

		form := url.Values{}
		form.Add("content", "Updated content.")
		req := httptest.NewRequest("POST", "/save/ExistingPage", strings.NewReader(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusFound)
		}
		updatedPage, err := repo.GetPageByTitle(context.Background(), "ExistingPage")
		if err != nil {
			t.Fatalf("Failed to get updated page from repo: %v", err)
		}
		if updatedPage.Content != "Updated content." {
			t.Errorf("page content was not updated: got '%s' want '%s'", updatedPage.Content, "Updated content.")
		}
	})
}
