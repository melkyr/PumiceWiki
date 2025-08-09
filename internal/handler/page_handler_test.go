//go:build integration

package handler

import (
	"context"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
)

type testApp struct {
	Router  *chi.Mux
	Repo    service.PageRepository
	Enforcer *casbin.Enforcer
}

// setupTest initializes a full application stack for testing.
func setupTest(t *testing.T) (*testApp, func()) {
	t.Helper()
	dsn := "file:memory?mode=memory&cache=shared"
	db, err := data.NewDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Manually apply migrations.
	schema1, _ := os.ReadFile("../../migrations/001_initial_schema.sql")
	db.MustExec(string(schema1))
	schema2, _ := os.ReadFile("../../migrations/002_create_casbin_rule_table.sql")
	db.MustExec(string(schema2))

	// Init layers.
	logger := log.New(os.Stdout, "TEST ", log.LstdFlags)
	viewService, _ := view.New(web.TemplateFS)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)
	pageHandler := NewPageHandler(pageService, viewService, logger)

	// Init auth components for the test.
	// We pass a nil authenticator because we are not testing the OIDC login flow itself,
	// only the authorization rules for an anonymous user.
	enforcer, _ := auth.NewEnforcer("sqlite3", dsn, "../../auth_model.conf")
	authzMiddleware := middleware.Authorizer(enforcer, nil)
	router := NewRouter(pageHandler, nil, authzMiddleware)

	app := &testApp{
		Router:  router,
		Repo:    pageRepository,
		Enforcer: enforcer,
	}

	teardown := func() {
		db.Close()
	}
	return app, teardown
}

// seedPolicies is a helper to add policies for testing.
func seedPolicies(t *testing.T, e *casbin.Enforcer) {
	t.Helper()
	policies := [][]string{
		{"anonymous", "/view/TestPage", "GET"},
		{"anonymous", "/auth/login", "GET"},
		{"editor", "/edit/TestPage", "GET"},
		{"editor", "/save/TestPage", "POST"},
	}
	for _, p := range policies {
		if _, err := e.AddPolicy(p); err != nil {
			t.Fatalf("Failed to add policy %v: %v", p, err)
		}
	}
}

func TestAuthzMiddleware(t *testing.T) {
	app, teardown := setupTest(t)
	defer teardown()

	// Seed specific policies for this test
	seedPolicies(t, app.Enforcer)

	// Create a page for viewing
	app.Repo.CreatePage(context.Background(), &data.Page{Title: "TestPage", Content: "content"})

	testCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"Anonymous can view page", "GET", "/view/TestPage", http.StatusOK},
		{"Anonymous cannot edit page", "GET", "/edit/TestPage", http.StatusForbidden},
		{"Anonymous cannot save page", "POST", "/save/TestPage", http.StatusForbidden},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.method == "POST" {
				form := url.Values{}
				form.Add("content", "new content")
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(form.Encode()))
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			}

			rr := httptest.NewRecorder()
			app.Router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tc.wantStatus)
			}
		})
	}
}
