package service

import (
	"context"
	"go-wiki-app/internal/data"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// PageRepository defines the interface for database operations on pages.
type PageRepository interface {
	CreatePage(ctx context.Context, page *data.Page) error
	GetPageByTitle(ctx context.Context, title string) (*data.Page, error)
	GetPageByID(ctx context.Context, id int64) (*data.Page, error)
	UpdatePage(ctx context.Context, page *data.Page) error
	DeletePage(ctx context.Context, id int64) error
}

// PageServicer defines the interface for interacting with pages.
type PageServicer interface {
	ViewPage(ctx context.Context, title string) (*data.Page, error)
	CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error)
	UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error)
}

// PageService provides business logic for managing pages.
type PageService struct {
	repo      PageRepository
	sanitizer *bluemonday.Policy
}

// NewPageService creates a new PageService with the given repository.
func NewPageService(repo PageRepository) *PageService {
	// Create a new bluemonday policy for user-generated content.
	// UGCPolicy is a good starting point as it allows basic formatting
	// like links, lists, bold, etc., while stripping out dangerous HTML.
	sanitizer := bluemonday.UGCPolicy()

	return &PageService{
		repo:      repo,
		sanitizer: sanitizer,
	}
}

// CreatePage handles the creation of a new wiki page.
// It sanitizes the content before passing it to the repository.
func (s *PageService) CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error) {
	// Sanitize the user-provided content to prevent XSS attacks.
	sanitizedContent := s.sanitizer.Sanitize(content)

	page := &data.Page{
		Title:    title,
		Content:  sanitizedContent,
		AuthorID: authorID,
	}

	if err := s.repo.CreatePage(ctx, page); err != nil {
		return nil, err
	}
	return page, nil
}

// ViewPage retrieves a single page by its title.
func (s *PageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	return s.repo.GetPageByTitle(ctx, title)
}

// UpdatePage handles the logic for updating an existing page.
// It sanitizes the content before passing it to the repository.
func (s *PageService) UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error) {
	page, err := s.repo.GetPageByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Sanitize the user-provided content.
	sanitizedContent := s.sanitizer.Sanitize(content)

	// Update the fields
	page.Title = title
	page.Content = sanitizedContent
	page.UpdatedAt = time.Now()

	if err := s.repo.UpdatePage(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// DeletePage handles the deletion of a page by its ID.
func (s *PageService) DeletePage(ctx context.Context, id int64) error {
	return s.repo.DeletePage(ctx, id)
}
