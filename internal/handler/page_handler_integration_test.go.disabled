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
)

type testApp struct {
	Router   *chi.Mux
	Repo     service.PageRepository
	Enforcer *casbin.Enforcer
}

// setupIntegrationTest initializes a full application stack for testing.
func setupIntegrationTest(t *testing.T) (*testApp, func()) {
	t.Helper()
	dsn := "file:memory?mode=memory&cache=shared"
	db, err := data.NewDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	schema1, _ := os.ReadFile("../../migrations/001_initial_schema.sql")
	db.MustExec(string(schema1))
	schema2, _ := os.ReadFile("../../migrations/002_create_casbin_rule_table.sql")
	db.MustExec(string(schema2))

	log := logger.New(config.LogConfig{Level: "debug", Format: "console"})
	viewService, _ := view.New(web.TemplateFS)
	pageRepository := data.NewSQLPageRepository(db)
	pageService := service.NewPageService(pageRepository)

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db.DB)
	sessionManager.Lifetime = 3 * time.Minute

	pageHandler := NewPageHandler(pageService, viewService, log)

	enforcer, _ := auth.NewEnforcer("sqlite3", dsn, "../../auth_model.conf")
	authzMiddleware := middleware.Authorizer(enforcer, sessionManager)
	errorMiddleware := middleware.Error(log, viewService)
	router := NewRouter(pageHandler, nil, authzMiddleware, errorMiddleware, sessionManager)

	app := &testApp{
		Router:   router,
		Repo:     pageRepository,
		Enforcer: enforcer,
	}

	teardown := func() {
		db.Close()
	}
	return app, teardown
}

func TestHandlers_Integration(t *testing.T) {
	app, teardown := setupIntegrationTest(t)
	defer teardown()

	// Seed policies
	app.Enforcer.AddPolicy("anonymous", "/view/*", "GET")
	app.Enforcer.AddPolicy("editor", "/edit/*", "GET")

	// Seed data
	app.Repo.CreatePage(context.Background(), &data.Page{Title: "TestPage", Content: "content"})

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
			app.Router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("want status %d; got %d", tc.wantStatus, rr.Code)
			}
			if tc.wantBody != "" && !strings.Contains(rr.Body.String(), tc.wantBody) {
				t.Errorf("body does not contain expected string '%s'", tc.wantBody)
			}
		})
	}
}
