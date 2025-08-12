//go:build integration

package handler

import (
	"context"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type testApp struct {
	Router   *chi.Mux
	Repo     service.PageRepository
	Enforcer *casbin.Enforcer
}

var testAppInstance *testApp

func TestMain(m *testing.M) {
	// Setup
	dsn := "file::memory:?cache=shared"
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		panic("Failed to connect to sqlite test database: " + err.Error())
	}

	// Run migrations
	sqliteSchema := `
CREATE TABLE pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	db.MustExec(sqliteSchema)
	schema2, _ := os.ReadFile("../../migrations/002_create_casbin_rule_table.up.sql")
	db.MustExec(string(schema2))
	schema3, _ := os.ReadFile("../../migrations/003_create_sessions_table.up.sql")
	db.MustExec(string(schema3))

	log := logger.New(config.LogConfig{Level: "debug", Format: "console"})
	viewService, _ := view.New(web.TemplateFS)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db.DB)
	sessionManager.Lifetime = 3 * time.Minute

	pageHandler := NewPageHandler(pageService, viewService, log)
	seoHandler := NewSeoHandler(pageService)

	enforcer, _ := auth.NewEnforcer("sqlite3", dsn, "../../auth_model.conf")
	authzMiddleware := middleware.Authorizer(enforcer, sessionManager)
	errorMiddleware := middleware.Error(log, viewService)
	router := NewRouter(pageHandler, nil, seoHandler, authzMiddleware, errorMiddleware, sessionManager)

	testAppInstance = &testApp{
		Router:   router,
		Repo:     pageRepository,
		Enforcer: enforcer,
	}

	// Run tests
	exitCode := m.Run()

	// Teardown
	db.Close()
	os.Exit(exitCode)
}

func TestHandlers_Integration(t *testing.T) {
	// Seed policies
	testAppInstance.Enforcer.AddPolicy("anonymous", "/view/*", "GET")
	testAppInstance.Enforcer.AddPolicy("editor", "/edit/*", "GET")

	// Seed data for this test
	page := &data.Page{Title: "TestPage", Content: "content"}
	err := testAppInstance.Repo.CreatePage(context.Background(), page)
	if err != nil {
		t.Fatalf("failed to create page for test: %v", err)
	}

	// For debugging:
	_, err = testAppInstance.Repo.GetPageByTitle(context.Background(), "TestPage")
	if err != nil {
		t.Fatalf("Failed to get page right after creating it: %v", err)
	}

	testCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "View Existing Page",
			method:     "GET",
			path:       "/view/TestPage",
			wantStatus: http.StatusOK,
			wantBody:   "<h2>TestPage</h2>",
		},
		{
			name:       "View Non-Existent Page (Not Found Error)",
			method:     "GET",
			path:       "/view/NotFoundPage",
			wantStatus: http.StatusNotFound,
			wantBody:   "Error 404",
		},
		{
			name:       "Edit Page without permission (Forbidden Error)",
			method:     "GET",
			path:       "/edit/TestPage",
			wantStatus: http.StatusForbidden,
			wantBody:   "Forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			testAppInstance.Router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("want status %d; got %d", tc.wantStatus, rr.Code)
			}
			if tc.wantBody != "" && !strings.Contains(rr.Body.String(), tc.wantBody) {
				t.Errorf("body does not contain expected string '%s'", tc.wantBody)
			}
		})
	}
}

func TestCompression_Integration(t *testing.T) {
	// Seed policies
	testAppInstance.Enforcer.AddPolicy("anonymous", "/view/*", "GET")

	// Seed data for this test
	page := &data.Page{Title: "CompressPage", Content: "this is some content to be compressed"}
	testAppInstance.Repo.CreatePage(context.Background(), page)

	req := httptest.NewRequest("GET", "/view/CompressPage", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	testAppInstance.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, rr.Code)
	}

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("want Content-Encoding header to be 'gzip'; got '%s'", rr.Header().Get("Content-Encoding"))
	}
}
