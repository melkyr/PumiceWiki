//go:build integration

package data

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// setupCategoryTest creates a new in-memory SQLite database and a CategoryRepository for testing.
// It returns the repository and a teardown function to be deferred.
func setupCategoryTest(t *testing.T) (*CategoryRepository, func()) {
	t.Helper()

	// Use a non-shared in-memory database for complete test isolation.
	dsn := "file::memory:"
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to sqlite test database: %v", err)
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	schema := `
	CREATE TABLE categories (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id INTEGER,
		FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE,
		UNIQUE (name, parent_id)
	);`
	db.MustExec(schema)

	repo := NewCategoryRepository(db)

	teardown := func() {
		db.Close()
	}

	return repo, teardown
}

func TestCategoryRepository_SaveParent(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	category := &Category{Name: "Science"}
	id, err := repo.Save(category)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}
}

func TestCategoryRepository_SaveSubcategory(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	parent := &Category{Name: "Technology"}
	parentID, err := repo.Save(parent)
	if err != nil {
		t.Fatalf("failed to save parent category: %v", err)
	}

	child := &Category{Name: "Programming", ParentID: &parentID}
	childID, err := repo.Save(child)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if childID == 0 {
		t.Error("expected non-zero id")
	}
}

func TestCategoryRepository_FindByName(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	parent := &Category{Name: "Sports"}
	parentID, err := repo.Save(parent)
	if err != nil { t.Fatal(err) }

	child := &Category{Name: "Soccer", ParentID: &parentID}
	_, err = repo.Save(child)
	if err != nil { t.Fatal(err) }

	// Test finding parent
	found, err := repo.FindByName("Sports", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find category, but got nil")
	}
	if found.Name != "Sports" {
		t.Errorf("expected name 'Sports', got '%s'", found.Name)
	}

	// Test finding child
	found, err = repo.FindByName("Soccer", &parentID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find category, but got nil")
	}
	if found.Name != "Soccer" {
		t.Errorf("expected name 'Soccer', got '%s'", found.Name)
	}

	// Test not found
	found, err = repo.FindByName("Basketball", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, but found category: %v", found)
	}
}

func TestCategoryRepository_GetByID(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	category := &Category{Name: "Movies"}
	id, err := repo.Save(category)
	if err != nil { t.Fatal(err) }

	found, err := repo.GetByID(id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find category, but got nil")
	}
	if found.Name != "Movies" {
		t.Errorf("expected name 'Movies', got '%s'", found.Name)
	}

	// Test not found
	found, err = repo.GetByID(999)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, but found category: %v", found)
	}
}

func TestCategoryRepository_GetAll(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	_, err := repo.Save(&Category{Name: "Books"})
	if err != nil { t.Fatal(err) }
	_, err = repo.Save(&Category{Name: "Music"})
	if err != nil { t.Fatal(err) }

	categories, err := repo.GetAll()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(categories))
	}
}

func TestCategoryRepository_SearchByName(t *testing.T) {
	repo, teardown := setupCategoryTest(t)
	defer teardown()

	_, err := repo.Save(&Category{Name: "History"})
	if err != nil { t.Fatal(err) }
	_, err = repo.Save(&Category{Name: "Historical Fiction"})
	if err != nil { t.Fatal(err) }
	_, err = repo.Save(&Category{Name: "Art History"})
	if err != nil { t.Fatal(err) }

	results, err := repo.SearchByName("History")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// "History" and "Art History" should match. "Historical Fiction" should not.
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
