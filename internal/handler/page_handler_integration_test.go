//go:build integration

package handler

import (
	"context"
	"go-wiki-app/internal/auth"
	"go-wiki-app/internal/cache"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/logger"
	"go-wiki-app/internal/middleware"
	"go-wiki-app/internal/service"
	"go-wiki-app/internal/view"
	"go-wiki-app/web"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	Router         *chi.Mux
	DB             *sqlx.DB
	PageRepo       *data.SQLPageRepository
	CategoryRepo   *data.CategoryRepository
	Enforcer       casbin.IEnforcer
	SessionManager *scs.SessionManager
}

var testAppInstance *testApp

func TestMain(m *testing.M) {
	dsn := "file:integration_test.db?cache=shared&mode=memory"
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		panic("Failed to connect to sqlite test database: " + err.Error())
	}

	// Migrations
	pagesSchema := `
	CREATE TABLE pages (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL UNIQUE,
		content TEXT NOT NULL,
		author_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		category_id INTEGER
	);`
	db.MustExec(pagesSchema)

	categoriesSchema := `
	CREATE TABLE categories (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id INTEGER,
		FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE,
		UNIQUE (name, parent_id)
	);`
	db.MustExec(categoriesSchema)

	casbinSchema, _ := os.ReadFile("../../migrations/002_create_casbin_rule_table.up.sql")
	db.MustExec(string(casbinSchema))
	sessionsSchema, _ := os.ReadFile("../../migrations/003_create_sessions_table.up.sql")
	db.MustExec(string(sessionsSchema))

	log := logger.New(config.LogConfig{Level: "debug", Format: "console"})
	viewService, _ := view.New(web.TemplateFS)
	testCache, _ := cache.New(config.CacheConfig{FilePath: "file::memory:"})

	pageRepository := data.NewSQLPageRepository(db)
	categoryRepository := data.NewCategoryRepository(db)
	pageService := service.NewPageService(pageRepository, categoryRepository, testCache)

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
		Router:         router,
		DB:             db,
		PageRepo:       pageRepository,
		CategoryRepo:   categoryRepository,
		Enforcer:       enforcer,
		SessionManager: sessionManager,
	}

	exitCode := m.Run()

	testCache.Close()
	db.Close()
	os.Exit(exitCode)
}

func getAuthenticatedCookie(t *testing.T) *http.Cookie {
	t.Helper()

	var cookie *http.Cookie

	// Create a dummy handler that sets the session
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// The Authorizer middleware uses "user_subject" from the session
		testAppInstance.SessionManager.Put(ctx, "user_subject", "test-editor")
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the dummy handler with the session middleware
	wrappedHandler := testAppInstance.SessionManager.LoadAndSave(handler)

	// Call the handler to get the session cookie
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) > 0 {
		cookie = cookies[0]
	} else {
		t.Fatal("failed to get session cookie")
	}

	return cookie
}

func TestSavePage_WithCategories_Integration(t *testing.T) {
	testAppInstance.DB.MustExec("DELETE FROM pages")
	testAppInstance.DB.MustExec("DELETE FROM categories")

	// Grant permissions for the test
	testAppInstance.Enforcer.AddPolicy("editor", "/save/NewCategorizedPage", "POST")
	testAppInstance.Enforcer.AddRoleForUser("test-editor", "editor")

	cookie := getAuthenticatedCookie(t)

	form := url.Values{}
	form.Add("title", "NewCategorizedPage")
	form.Add("content", "Some content")
	form.Add("category", "IntegrationTests")
	form.Add("subcategory", "Passing")

	req := httptest.NewRequest("POST", "/save/NewCategorizedPage", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	testAppInstance.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("want status %d; got %d", http.StatusFound, rr.Code)
	}

	page, err := testAppInstance.PageRepo.GetPageByTitle(context.Background(), "NewCategorizedPage")
	if err != nil {
		t.Fatalf("Failed to retrieve saved page: %v", err)
	}
	if page.CategoryID == nil {
		t.Fatal("Page was saved with a nil CategoryID")
	}

	subCategory, err := testAppInstance.CategoryRepo.GetByID(*page.CategoryID)
	if err != nil {
		t.Fatalf("Failed to retrieve subcategory: %v", err)
	}
	if subCategory.Name != "Passing" {
		t.Errorf("Expected subcategory name 'Passing', got '%s'", subCategory.Name)
	}
}
