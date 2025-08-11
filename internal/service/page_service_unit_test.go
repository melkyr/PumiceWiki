//go:build unit

package service

import (
	"context"
	"errors"
	"go-wiki-app/internal/data"
	"testing"
)

// mockPageRepository is a mock implementation of the PageRepository interface.
// It allows us to control its behavior for testing purposes.
type mockPageRepository struct {
	// Fields to control mock behavior
	errToReturn  error
	pageToReturn *data.Page

	// Fields to record method calls
	createPageCalled bool
	getPageByTitleCalled bool
	getPageByIDCalled bool
	updatePageCalled bool
	deletePageCalled bool

	// Field to capture passed data
	lastPagePassed *data.Page
}

// Ensure mockPageRepository implements the PageRepository interface.
var _ PageRepository = (*mockPageRepository)(nil)

func (m *mockPageRepository) CreatePage(ctx context.Context, page *data.Page) error {
	m.createPageCalled = true
	m.lastPagePassed = page
	if m.errToReturn != nil {
		return m.errToReturn
	}
	// Simulate the DB populating the ID
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

func (m *mockPageRepository) UpdatePage(ctx context.Context, page *data.Page) error {
	m.updatePageCalled = true
	m.lastPagePassed = page
	return m.errToReturn
}

func (m *mockPageRepository) DeletePage(ctx context.Context, id int64) error {
	m.deletePageCalled = true
	return m.errToReturn
}

func TestPageService_CreatePage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		mockRepo := &mockPageRepository{}
		pageService := NewPageService(mockRepo)
		ctx := context.Background()
		title := "Test Page"
		content := "Hello <b>world</b>! <script>alert('xss')</script>"
		authorID := "user123"

		// Act
		page, err := pageService.CreatePage(ctx, title, content, authorID)

		// Assert
		if err != nil {
			t.Errorf("expected no error, but got %v", err)
		}
		if !mockRepo.createPageCalled {
			t.Error("expected repository's CreatePage to be called, but it wasn't")
		}
		if page == nil {
			t.Fatal("expected a page to be returned, but got nil")
		}
		if page.ID != 1 {
			t.Errorf("expected page ID to be 1, but got %d", page.ID)
		}

		// Assert that content was sanitized
		expectedSanitizedContent := "Hello <b>world</b>! "
		if mockRepo.lastPagePassed.Content != expectedSanitizedContent {
			t.Errorf("content was not sanitized correctly\ngot:  '%s'\nwant: '%s'", mockRepo.lastPagePassed.Content, expectedSanitizedContent)
		}
	})

	t.Run("failure", func(t *testing.T) {
		// Arrange
		mockRepo := &mockPageRepository{
			errToReturn: errors.New("database error"),
		}
		pageService := NewPageService(mockRepo)
		ctx := context.Background()

		// Act
		_, err := pageService.CreatePage(ctx, "title", "content", "author")

		// Assert
		if err == nil {
			t.Error("expected an error, but got nil")
		}
		if err.Error() != "database error" {
			t.Errorf("expected error 'database error', but got '%v'", err)
		}
	})
}

func TestPageService_ViewPage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		mockRepo := &mockPageRepository{
			pageToReturn: &data.Page{ID: 1, Title: "Test Page", Content: "Test Content"},
		}
		pageService := NewPageService(mockRepo)
		ctx := context.Background()

		// Act
		page, err := pageService.ViewPage(ctx, "Test Page")

		// Assert
		if err != nil {
			t.Errorf("expected no error, but got %v", err)
		}
		if !mockRepo.getPageByTitleCalled {
			t.Error("expected repository's GetPageByTitle to be called, but it wasn't")
		}
		if page == nil {
			t.Fatal("expected a page to be returned, but got nil")
		}
		if page.Title != "Test Page" {
			t.Errorf("expected page title to be 'Test Page', but got '%s'", page.Title)
		}
	})
}

func TestPageService_UpdatePage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		mockRepo := &mockPageRepository{
			pageToReturn: &data.Page{ID: 1, Title: "Old Title", Content: "Old Content"},
		}
		pageService := NewPageService(mockRepo)
		ctx := context.Background()
		newContent := "New <b>sanitized</b> content <script>nope</script>"

		// Act
		page, err := pageService.UpdatePage(ctx, 1, "New Title", newContent)

		// Assert
		if err != nil {
			t.Errorf("expected no error, but got %v", err)
		}
		if !mockRepo.getPageByIDCalled {
			t.Error("expected repository's GetPageByID to be called, but it wasn't")
		}
		if !mockRepo.updatePageCalled {
			t.Error("expected repository's UpdatePage to be called, but it wasn't")
		}
		if page.Content != "New <b>sanitized</b> content " {
			t.Errorf("content was not sanitized correctly\ngot:  '%s'", page.Content)
		}
	})
}

func TestPageService_DeletePage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		mockRepo := &mockPageRepository{}
		pageService := NewPageService(mockRepo)
		ctx := context.Background()

		// Act
		err := pageService.DeletePage(ctx, 1)

		// Assert
		if err != nil {
			t.Errorf("expected no error, but got %v", err)
		}
		if !mockRepo.deletePageCalled {
			t.Error("expected repository's DeletePage to be called, but it wasn't")
		}
	})
}
