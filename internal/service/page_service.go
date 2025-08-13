package service

import (
	"bytes"
	"context"
	"encoding/json"
	"go-wiki-app/internal/cache"
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
	cache     *cache.Cache
	sanitizer *bluemonday.Policy
	markdown  goldmark.Markdown
}

// NewPageService creates a new PageService with its dependencies.
func NewPageService(repo PageRepository, cache *cache.Cache) *PageService {
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
		cache:     cache,
		sanitizer: sanitizer,
		markdown:  markdown,
	}
}

// CreatePage handles the business logic for creating a new wiki page.
// It sanitizes the user-provided Markdown content before saving it to the database.
// It also invalidates the cache for the list of all pages.
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

	// Invalidate the cache for the list of all pages since a new page was added.
	s.cache.Delete("pages:all")

	return page, nil
}

// ViewPage retrieves a single page by its title. It uses a cache to speed up
// access to frequently viewed pages. If the page is not in the cache, it
// fetches it from the repository, stores it in the cache, and then returns it.
// It also processes the Markdown content into sanitized HTML.
func (s *PageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	// 1. Check the cache first.
	cacheKey := "page:" + title
	if cachedBytes, _ := s.cache.Get(cacheKey); cachedBytes != nil {
		var page data.Page
		if json.Unmarshal(cachedBytes, &page) == nil {
			// Cache hit, process markdown and return.
			s.processMarkdown(&page)
			return &page, nil
		}
	}

	// 2. Cache miss: Fetch from the repository.
	page, err := s.repo.GetPageByTitle(ctx, title)
	if err != nil {
		return nil, err
	}

	// 3. Store in cache for future requests.
	// We cache the raw page data from the DB, not the processed HTML.
	if bytesToCache, err := json.Marshal(page); err == nil {
		s.cache.Set(cacheKey, bytesToCache, 5*time.Minute) // TODO: Use configured TTL
	}

	// 4. Process markdown and return.
	s.processMarkdown(page)
	return page, nil
}

// UpdatePage handles the logic for updating an existing page.
// It sanitizes the content and updates the database. Crucially, it also
// invalidates the cache for the updated page and the list of all pages.
func (s *PageService) UpdatePage(ctx context.Context, id int64, title, content string) (*data.Page, error) {
	page, err := s.repo.GetPageByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Invalidate the cache for the old title, in case the title is being changed.
	s.cache.Delete("page:" + page.Title)
	// Invalidate the cache for the list of all pages.
	s.cache.Delete("pages:all")

	// Sanitize the user-provided content.
	sanitizedContent := s.sanitizer.Sanitize(content)

	// Update the fields
	page.Title = title
	page.Content = sanitizedContent
	page.UpdatedAt = time.Now()

	if err := s.repo.UpdatePage(ctx, page); err != nil {
		return nil, err
	}

	// Invalidate the cache for the new title as well.
	s.cache.Delete("page:" + page.Title)

	return page, nil
}

// processMarkdown is a helper function to convert a page's Markdown content
// into sanitized HTML. This logic is used for both cache hits and misses.
func (s *PageService) processMarkdown(page *data.Page) {
	var buf bytes.Buffer
	if err := s.markdown.Convert([]byte(page.Content), &buf); err == nil {
		sanitizedHTML := s.sanitizer.SanitizeBytes(buf.Bytes())
		page.HTMLContent = template.HTML(sanitizedHTML)
	}
}

// DeletePage handles the deletion of a page by its ID.
func (s *PageService) DeletePage(ctx context.Context, id int64) error {
	return s.repo.DeletePage(ctx, id)
}

// GetAllPages retrieves all pages, using a cache to avoid frequent database hits.
func (s *PageService) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	// 1. Check the cache first.
	cacheKey := "pages:all"
	if cachedBytes, _ := s.cache.Get(cacheKey); cachedBytes != nil {
		var pages []*data.Page
		if json.Unmarshal(cachedBytes, &pages) == nil {
			return pages, nil
		}
	}

	// 2. Cache miss: Fetch from the repository.
	pages, err := s.repo.GetAllPages(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Store in cache for future requests.
	if bytesToCache, err := json.Marshal(pages); err == nil {
		s.cache.Set(cacheKey, bytesToCache, 5*time.Minute) // TODO: Use configured TTL
	}

	return pages, nil
}
