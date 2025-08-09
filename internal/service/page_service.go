package service

import (
	"context"
	"go-wiki-app/internal/data"
	"time"
)

// PageRepository defines the interface for database operations on pages.
// This allows us to decouple the service from the database implementation,
// making the service easier to test and maintain.
type PageRepository interface {
	CreatePage(ctx context.Context, page *data.Page) error
	GetPageByTitle(ctx context.Context, title string) (*data.Page, error)
	GetPageByID(ctx context.Context, id int64) (*data.Page, error)
	UpdatePage(ctx context.Context, page *data.Page) error
	DeletePage(ctx context.Context, id int64) error
}

// PageService provides business logic for managing pages.
type PageService struct {
	repo PageRepository
}

// NewPageService creates a new PageService with the given repository.
func NewPageService(repo PageRepository) *PageService {
	return &PageService{repo: repo}
}

// CreatePage handles the creation of a new wiki page.
// It contains business logic for validation before passing it to the repository.
func (s *PageService) CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error) {
	// In a real application, you would add more validation here.
	// For example, checking for empty title/content, sanitizing input, etc.

	page := &data.Page{
		Title:    title,
		Content:  content,
		AuthorID: authorID,
	}

	if err := s.repo.CreatePage(ctx, page); err != nil {
		return nil, err
	}

	// The repository's CreatePage method is expected to populate the ID and timestamps.
	return page, nil
}

// ViewPage retrieves a single page by its title.
func (s *PageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	return s.repo.GetPageByTitle(ctx, title)
}

// UpdatePage handles the logic for updating an existing page.
func (s *PageService) UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error) {
	// First, retrieve the existing page to ensure it exists.
	page, err := s.repo.GetPageByID(ctx, id)
	if err != nil {
		return nil, err // Repository should return a specific error for not found.
	}

	// Update the fields
	page.Title = title
	page.Content = content
	page.UpdatedAt = time.Now() // The service layer can be responsible for timestamps

	if err := s.repo.UpdatePage(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// DeletePage handles the deletion of a page by its ID.
func (s *PageService) DeletePage(ctx context.Context, id int64) error {
	// You might add business logic here, e.g., checking if the user has permission
	// before attempting to delete. For now, we delegate directly to the repository.
	return s.repo.DeletePage(ctx, id)
}
