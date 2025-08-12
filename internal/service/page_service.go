package service

import (
	"bytes"
	"context"
	"go-wiki-app/internal/data"
	"html/template"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

// PageRepository defines the interface for database operations on pages.
type PageRepository interface {
	CreatePage(ctx context.Context, page *data.Page) error
	GetPageByTitle(ctx context.Context, title string) (*data.Page, error)
	GetPageByID(ctx context.Context, id int64) (*data.Page, error)
	GetAllPages(ctx context.Context) ([]*data.Page, error)
	UpdatePage(ctx context.Context, page *data.Page) error
	DeletePage(ctx context.Context, id int64) error
}

// PageServicer defines the interface for interacting with pages.
type PageServicer interface {
	ViewPage(ctx context.Context, title string) (*data.Page, error)
	CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error)
	UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error)
	GetAllPages(ctx context.Context) ([]*data.Page, error)
}

// PageService provides business logic for managing pages.
type PageService struct {
	repo      PageRepository
	sanitizer *bluemonday.Policy
	markdown  goldmark.Markdown
}

// NewPageService creates a new PageService with its dependencies.
func NewPageService(repo PageRepository) *PageService {
	// sanitizer is the policy for cleaning user-provided HTML to prevent XSS.
	// We start with the UGCPolicy which allows common formatting and links.
	sanitizer := bluemonday.UGCPolicy()
	// We explicitly allow images to be rendered.
	sanitizer.AllowImages()

	// markdown is the engine for converting Markdown text to HTML.
	markdown := goldmark.New(
		goldmark.WithExtensions(
		// Extensions like tables, strikethrough, etc., could be added here.
		),
	)

	return &PageService{
		repo:      repo,
		sanitizer: sanitizer,
		markdown:  markdown,
	}
}

// CreatePage handles the business logic for creating a new wiki page.
// It sanitizes the user-provided Markdown content before saving it to the database.
func (s *PageService) CreatePage(ctx context.Context, title, content, authorID string) (*data.Page, error) {
	// Note: We are sanitizing the raw Markdown here. This is a first line of
	// defense, but the primary sanitization happens after it's converted to HTML.
	sanitizedContent := s.sanitizer.Sanitize(content)

	page := &data.Page{
		Title:    title,
		Content:  sanitizedContent, // The raw, sanitized Markdown is stored.
		AuthorID: authorID,
	}

	if err := s.repo.CreatePage(ctx, page); err != nil {
		return nil, err
	}
	return page, nil
}

// ViewPage retrieves a single page by its title. It then converts the page's
// raw Markdown content into sanitized HTML, which is placed in the `HTMLContent`
// field for safe rendering in templates.
func (s *PageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	page, err := s.repo.GetPageByTitle(ctx, title)
	if err != nil {
		return nil, err
	}

	// 1. Convert the raw Markdown content from the DB into HTML.
	var buf bytes.Buffer
	if err := s.markdown.Convert([]byte(page.Content), &buf); err != nil {
		// If conversion fails, return the error but don't halt the application.
		return nil, err
	}

	// 2. Sanitize the generated HTML to prevent XSS attacks. This is the most
	//    important sanitization step, as it operates on the final HTML output.
	sanitizedHTML := s.sanitizer.SanitizeBytes(buf.Bytes())
	page.HTMLContent = template.HTML(sanitizedHTML)

	return page, nil
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

// GetAllPages retrieves all pages.
func (s *PageService) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	return s.repo.GetAllPages(ctx)
}
