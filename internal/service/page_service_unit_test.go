//go:build unit

package service

import (
	"context"
	"errors"
	"go-wiki-app/internal/cache"
	"go-wiki-app/internal/config"
	"go-wiki-app/internal/data"
	"testing"
)

// newTestCache creates a new in-memory cache for testing.
func newTestCache(t *testing.T) (*cache.Cache, func()) {
	t.Helper()
	cfg := config.CacheConfig{
		FilePath: "file::memory:",
	}
	c, err := cache.New(cfg)
	if err != nil {
		t.Fatalf("failed to create test cache: %v", err)
	}
	teardown := func() {
		c.Close()
	}
	return c, teardown
}

// mockPageRepository is a mock implementation of the PageRepository interface.
type mockPageRepository struct {
	errToReturn   error
	pageToReturn  *data.Page
	pagesToReturn []*data.Page
	createPageCalled bool
	getPageByTitleCalled bool
	getPageByIDCalled bool
	getAllPagesCalled bool
	updatePageCalled bool
	deletePageCalled bool
	lastPagePassed *data.Page
}

var _ PageRepository = (*mockPageRepository)(nil)

func (m *mockPageRepository) CreatePage(ctx context.Context, page *data.Page) error {
	m.createPageCalled = true
	m.lastPagePassed = page
	if m.errToReturn != nil {
		return m.errToReturn
	}
	page.ID = 1
	return nil
}

func (m *mockPageRepository) GetPageByTitle(ctx context.Context, title string) (*data.Page, error) {
	m.getPageByTitleCalled = true
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	if m.pageToReturn != nil && m.pageToReturn.Title == title {
		return m.pageToReturn, nil
	}
	return nil, errors.New("page not found")
}

func (m *mockPageRepository) GetPageByID(ctx context.Context, id int64) (*data.Page, error) {
	m.getPageByIDCalled = true
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	if m.pageToReturn != nil && m.pageToReturn.ID == id {
		return m.pageToReturn, nil
	}
	return nil, errors.New("page not found")
}

func (m *mockPageRepository) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	m.getAllPagesCalled = true
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.pagesToReturn, nil
}

func (m *mockPageRepository) UpdatePage(ctx context.Context, page *data.Page) error {
	m.updatePageCalled = true
	m.lastPagePassed = page
	return m.errToReturn
}

func (m *mockPageRepository) DeletePage(ctx context.Context, id int64) error {
	m.deletePageCalled = true
	return m.errToReturn
}

func (m *mockPageRepository) GetPagesByCategoryID(ctx context.Context, categoryID int64) ([]*data.Page, error) {
	// For now, return an empty slice and no error.
	// This can be expanded if tests need more specific behavior.
	return []*data.Page{}, nil
}

// mockCategoryRepository is a mock implementation of the CategoryRepository interface.
type mockCategoryRepository struct {
	findByNameFunc func(name string, parentID *int64) (*data.Category, error)
	saveFunc       func(category *data.Category) (int64, error)
	getByIDFunc    func(id int64) (*data.Category, error)
	getAllFunc     func() ([]*data.Category, error)
	searchByNameFunc func(query string) ([]*data.Category, error)

	findByNameCalled   int
	saveCalled         int
	getByIDCalled      int
	getAllCalled       int
	searchByNameCalled int
	lastSavedCategory *data.Category
}

var _ CategoryRepository = (*mockCategoryRepository)(nil)

func (m *mockCategoryRepository) FindByName(name string, parentID *int64) (*data.Category, error) {
	m.findByNameCalled++
	if m.findByNameFunc != nil {
		return m.findByNameFunc(name, parentID)
	}
	return nil, nil
}

func (m *mockCategoryRepository) Save(category *data.Category) (int64, error) {
	m.saveCalled++
	m.lastSavedCategory = category
	if m.saveFunc != nil {
		return m.saveFunc(category)
	}
	return int64(m.saveCalled), nil
}

func (m *mockCategoryRepository) GetByID(id int64) (*data.Category, error) {
	m.getByIDCalled++
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return nil, nil
}

func (m *mockCategoryRepository) GetAll() ([]*data.Category, error) {
	m.getAllCalled++
	if m.getAllFunc != nil {
		return m.getAllFunc()
	}
	return []*data.Category{}, nil
}

func (m *mockCategoryRepository) SearchByName(query string) ([]*data.Category, error) {
	m.searchByNameCalled++
    if m.searchByNameFunc != nil {
        return m.searchByNameFunc(query)
    }
    return nil, nil
}

func TestPageService_CreatePage_WithCategories(t *testing.T) {
	t.Run("success with new categories", func(t *testing.T) {
		mockPageRepo := &mockPageRepository{}
		mockCategoryRepo := &mockCategoryRepository{}
		testCache, teardown := newTestCache(t)
		defer teardown()

		mockCategoryRepo.findByNameFunc = func(name string, parentID *int64) (*data.Category, error) {
			return nil, nil // Simulate categories not found
		}

		pageService := NewPageService(mockPageRepo, mockCategoryRepo, testCache)
		ctx := context.Background()

		_, err := pageService.CreatePage(ctx, "title", "content", "author", "Cat", "Subcat")
		if err != nil {
			t.Fatalf("CreatePage failed: %v", err)
		}

		if mockCategoryRepo.findByNameCalled != 2 {
			t.Errorf("expected FindByName to be called twice, got %d", mockCategoryRepo.findByNameCalled)
		}
		if mockCategoryRepo.saveCalled != 2 {
			t.Errorf("expected Save to be called twice, got %d", mockCategoryRepo.saveCalled)
		}
		if mockPageRepo.lastPagePassed.CategoryID == nil || *mockPageRepo.lastPagePassed.CategoryID != 2 {
			t.Errorf("expected page to be saved with CategoryID 2, got %v", mockPageRepo.lastPagePassed.CategoryID)
		}
	})
}

func TestPageService_GetCategoryTree(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPageRepo := &mockPageRepository{}
		mockCategoryRepo := &mockCategoryRepository{}
		testCache, teardown := newTestCache(t)
		defer teardown()

		parentID := int64(1)
		mockCategoryRepo.getAllFunc = func() ([]*data.Category, error) {
			return []*data.Category{
				{ID: 1, Name: "Science"},
				{ID: 2, Name: "Physics", ParentID: &parentID},
				{ID: 3, Name: "Arts"},
			}, nil
		}
		pageService := NewPageService(mockPageRepo, mockCategoryRepo, testCache)
		ctx := context.Background()

		tree, err := pageService.GetCategoryTree(ctx)
		if err != nil {
			t.Fatalf("GetCategoryTree failed: %v", err)
		}

		if len(tree) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(tree))
		}
		for _, node := range tree {
			if node.Parent.Name == "Science" {
				if len(node.Children) != 1 {
					t.Errorf("expected 1 child for Science, got %d", len(node.Children))
				}
			} else if node.Parent.Name == "Arts" {
				if len(node.Children) != 0 {
					t.Errorf("expected 0 children for Arts, got %d", len(node.Children))
				}
			}
		}
	})
}

func TestPageService_ViewPage_PopulatesCategories(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		testCache, teardown := newTestCache(t)
		defer teardown()

		catID := int64(2)
		parentCatID := int64(1)
		mockPageRepo := &mockPageRepository{
			pageToReturn: &data.Page{ID: 1, Title: "Test Page", CategoryID: &catID},
		}
		mockCategoryRepo := &mockCategoryRepository{}
		mockCategoryRepo.getByIDFunc = func(id int64) (*data.Category, error) {
			if id == 2 {
				return &data.Category{ID: 2, Name: "Subcat", ParentID: &parentCatID}, nil
			}
			if id == 1 {
				return &data.Category{ID: 1, Name: "Cat"}, nil
			}
			return nil, errors.New("not found")
		}
		pageService := NewPageService(mockPageRepo, mockCategoryRepo, testCache)
		ctx := context.Background()

		page, err := pageService.ViewPage(ctx, "Test Page")
		if err != nil {
			t.Fatalf("ViewPage failed: %v", err)
		}

		if page.CategoryName != "Cat" {
			t.Errorf("expected CategoryName 'Cat', got '%s'", page.CategoryName)
		}
		if page.SubcategoryName != "Subcat" {
			t.Errorf("expected SubcategoryName 'Subcat', got '%s'", page.SubcategoryName)
		}
	})
}
