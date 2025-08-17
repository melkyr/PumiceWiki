package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-wiki-app/internal/cache"
	"go-wiki-app/internal/data"
	"go-wiki-app/internal/middleware"
	"html/template"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// lazyLoadRenderer is a custom renderer for images.
type lazyLoadRenderer struct {
	html.Config
}

// NewLazyLoadRenderer creates a new custom image renderer.
func NewLazyLoadRenderer() renderer.NodeRenderer {
	return &lazyLoadRenderer{
		Config: html.NewConfig(),
	}
}

// RegisterFuncs registers the renderer for the Image node.
func (r *lazyLoadRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *lazyLoadRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	_, _ = w.WriteString("<img src=\"")
	_, _ = w.Write(util.EscapeHTML(n.Destination))
	_, _ = w.WriteString("\" alt=\"")
	_, _ = w.Write(util.EscapeHTML(n.Text(source)))
	_, _ = w.WriteString("\" loading=\"lazy\"")
	if n.Title != nil {
		_, _ = w.WriteString(" title=\"")
		_, _ = w.Write(util.EscapeHTML(n.Title))
		_, _ = w.WriteString("\"")
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, nil)
	}
	_, _ = w.WriteString(">")
	return ast.WalkSkipChildren, nil
}

// PageRepository defines the interface for database operations on pages.
type PageRepository interface {
	CreatePage(ctx context.Context, page *data.Page) error
	GetPageByTitle(ctx context.Context, title string) (*data.Page, error)
	GetPageByID(ctx context.Context, id int64) (*data.Page, error)
	GetAllPages(ctx context.Context) ([]*data.Page, error)
	UpdatePage(ctx context.Context, page *data.Page) error
	DeletePage(ctx context.Context, id int64) error
	GetPagesByCategoryID(ctx context.Context, categoryID int64) ([]*data.Page, error)
}

// CategoryRepository defines the interface for database operations on categories.
type CategoryRepository interface {
	FindByName(name string, parentID *int64) (*data.Category, error)
	Save(category *data.Category) (int64, error)
	GetByID(id int64) (*data.Category, error)
	GetAll() ([]*data.Category, error)
	SearchByName(query string) ([]*data.Category, error)
}

// CategoryNode represents a parent category and its children.
type CategoryNode struct {
	Parent   *data.Category
	Children []*data.Category
}

// PageServicer defines the interface for interacting with pages.
type PageServicer interface {
	ViewPage(ctx context.Context, title string) (*data.Page, error)
	CreatePage(ctx context.Context, title, content, authorID, categoryName, subcategoryName string) (*data.Page, error)
	UpdatePage(ctx context.Context, id int64, title, content, categoryName, subcategoryName string) (*data.Page, error)
	GetAllPages(ctx context.Context) ([]*data.Page, error)
	DeletePage(ctx context.Context, id int64) error
	GetCategoryTree(ctx context.Context) ([]*CategoryNode, error)
	SearchCategories(ctx context.Context, query string) ([]*data.Category, error)
	GetPagesForCategory(ctx context.Context, categoryName string) ([]*data.Page, error)
	GetPagesForSubcategory(ctx context.Context, categoryName string, subcategoryName string) ([]*data.Page, error)
}

var ErrAnonymousHome = errors.New("anonymous user viewing non-existent home page")

// PageService provides business logic for managing pages.
type PageService struct {
	repo         PageRepository
	categoryRepo CategoryRepository
	cache        *cache.Cache
	sanitizer    *bluemonday.Policy
	markdown     goldmark.Markdown
}

// NewPageService creates a new PageService with its dependencies.
func NewPageService(repo PageRepository, categoryRepo CategoryRepository, cache *cache.Cache) *PageService {
	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowImages()
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(NewLazyLoadRenderer(), 100),
			),
		),
	)
	return &PageService{
		repo:         repo,
		categoryRepo: categoryRepo,
		cache:        cache,
		sanitizer:    sanitizer,
		markdown:     markdown,
	}
}

// CreatePage handles the business logic for creating a new wiki page.
func (s *PageService) CreatePage(ctx context.Context, title, content, authorID, categoryName, subcategoryName string) (*data.Page, error) {
	sanitizedContent := s.sanitizer.Sanitize(content)
	categoryID, err := s.getOrCreateCategories(ctx, categoryName, subcategoryName)
	if err != nil {
		return nil, err
	}
	page := &data.Page{
		Title:      title,
		Content:    sanitizedContent,
		AuthorID:   authorID,
		CategoryID: categoryID,
	}
	if err := s.repo.CreatePage(ctx, page); err != nil {
		return nil, err
	}
	s.cache.Delete("pages:all")
	return page, nil
}

// ViewPage retrieves a single page by its title.
func (s *PageService) ViewPage(ctx context.Context, title string) (*data.Page, error) {
	cacheKey := "page:" + title
	if cachedBytes, _ := s.cache.Get(cacheKey); cachedBytes != nil {
		var page data.Page
		if json.Unmarshal(cachedBytes, &page) == nil {
			s.processMarkdown(&page)
			return &page, nil
		}
	}
	page, err := s.repo.GetPageByTitle(ctx, title)
	if err != nil {
		if title == "Home" {
			userInfo := middleware.GetUserInfo(ctx)
			if userInfo.Subject == "anonymous" {
				return nil, ErrAnonymousHome
			}
			// Return a default page for logged-in users if Home doesn't exist
			page = &data.Page{
				Title:   "Home",
				Content: "Welcome! This page is empty.",
			}
		} else {
			return nil, fmt.Errorf("failed to get page from repo: %w", err)
		}
	} else {
		if err := s.populateCategoryNames(page); err != nil {
			// Log error but don't fail the request
		}
		if bytesToCache, err := json.Marshal(page); err == nil {
			s.cache.Set(cacheKey, bytesToCache, 5*time.Minute)
		}
	}
	s.processMarkdown(page)
	return page, nil
}

// UpdatePage handles the logic for updating an existing page.
func (s *PageService) UpdatePage(ctx context.Context, id int64, title, content, categoryName, subcategoryName string) (*data.Page, error) {
	page, err := s.repo.GetPageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cache.Delete("page:" + page.Title)
	s.cache.Delete("pages:all")
	sanitizedContent := s.sanitizer.Sanitize(content)
	categoryID, err := s.getOrCreateCategories(ctx, categoryName, subcategoryName)
	if err != nil {
		return nil, err
	}
	page.Title = title
	page.Content = sanitizedContent
	page.UpdatedAt = time.Now()
	page.CategoryID = categoryID
	if err := s.repo.UpdatePage(ctx, page); err != nil {
		return nil, err
	}
	s.cache.Delete("page:" + page.Title)
	return page, nil
}

// GetAllPages retrieves all pages.
func (s *PageService) GetAllPages(ctx context.Context) ([]*data.Page, error) {
	pages, err := s.repo.GetAllPages(ctx)
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		if err := s.populateCategoryNames(page); err != nil {
			// Log error but continue
		}
	}
	return pages, nil
}

// DeletePage handles the deletion of a page by its ID.
func (s *PageService) DeletePage(ctx context.Context, id int64) error {
	return s.repo.DeletePage(ctx, id)
}

// GetCategoryTree fetches all categories and organizes them into a tree structure.
func (s *PageService) GetCategoryTree(ctx context.Context) ([]*CategoryNode, error) {
	categories, err := s.categoryRepo.GetAll()
	if err != nil {
		return nil, err
	}
	var nodes []*CategoryNode
	parentMap := make(map[int64]*CategoryNode)
	for _, c := range categories {
		if c.ParentID == nil {
			node := &CategoryNode{Parent: c}
			nodes = append(nodes, node)
			parentMap[c.ID] = node
		}
	}
	for _, c := range categories {
		if c.ParentID != nil {
			if parentNode, ok := parentMap[*c.ParentID]; ok {
				parentNode.Children = append(parentNode.Children, c)
			}
		}
	}
	return nodes, nil
}

// SearchCategories searches for categories by name.
func (s *PageService) SearchCategories(ctx context.Context, query string) ([]*data.Category, error) {
	return s.categoryRepo.SearchByName(query)
}

// GetPagesForCategory retrieves all pages for a given category name.
func (s *PageService) GetPagesForCategory(ctx context.Context, categoryName string) ([]*data.Page, error) {
	parent, err := s.categoryRepo.FindByName(categoryName, nil)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("category '%s' not found", categoryName)
	}

	allCategories, err := s.categoryRepo.GetAll()
	if err != nil {
		return nil, err
	}

	var subCategoryIDs []int64
	for _, cat := range allCategories {
		if cat.ParentID != nil && *cat.ParentID == parent.ID {
			subCategoryIDs = append(subCategoryIDs, cat.ID)
		}
	}

	var allPages []*data.Page
	for _, id := range subCategoryIDs {
		pages, err := s.repo.GetPagesByCategoryID(ctx, id)
		if err != nil {
			return nil, err
		}
		allPages = append(allPages, pages...)
	}

	return allPages, nil
}

// GetPagesForSubcategory retrieves all pages for a given subcategory name.
func (s *PageService) GetPagesForSubcategory(ctx context.Context, categoryName string, subcategoryName string) ([]*data.Page, error) {
	parent, err := s.categoryRepo.FindByName(categoryName, nil)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("category '%s' not found", categoryName)
	}

	subCategory, err := s.categoryRepo.FindByName(subcategoryName, &parent.ID)
	if err != nil {
		return nil, err
	}
	if subCategory == nil {
		return nil, fmt.Errorf("subcategory '%s' not found in category '%s'", subcategoryName, categoryName)
	}

	return s.repo.GetPagesByCategoryID(ctx, subCategory.ID)
}

func (s *PageService) processMarkdown(page *data.Page) {
	var buf bytes.Buffer
	if err := s.markdown.Convert([]byte(page.Content), &buf); err == nil {
		sanitizedHTML := s.sanitizer.SanitizeBytes(buf.Bytes())
		page.HTMLContent = template.HTML(sanitizedHTML)
	}
}

func (s *PageService) getOrCreateCategories(ctx context.Context, categoryName, subcategoryName string) (*int64, error) {
	if categoryName == "" {
		categoryName = "NoCategory"
	}
	if subcategoryName == "" {
		subcategoryName = "NoSubCategory"
	}
	mainCategory, err := s.categoryRepo.FindByName(categoryName, nil)
	if err != nil {
		return nil, err
	}
	if mainCategory == nil {
		newCat := &data.Category{Name: categoryName}
		id, err := s.categoryRepo.Save(newCat)
		if err != nil {
			return nil, err
		}
		mainCategory = &data.Category{ID: id, Name: categoryName}
	}
	subCategory, err := s.categoryRepo.FindByName(subcategoryName, &mainCategory.ID)
	if err != nil {
		return nil, err
	}
	if subCategory == nil {
		newSubCat := &data.Category{Name: subcategoryName, ParentID: &mainCategory.ID}
		id, err := s.categoryRepo.Save(newSubCat)
		if err != nil {
			return nil, err
		}
		subCategory = &data.Category{ID: id, Name: subcategoryName, ParentID: &mainCategory.ID}
	}
	return &subCategory.ID, nil
}

func (s *PageService) populateCategoryNames(page *data.Page) error {
	if page.CategoryID == nil {
		page.CategoryName = "NoCategory"
		page.SubcategoryName = "NoSubCategory"
		return nil
	}
	subCategory, err := s.categoryRepo.GetByID(*page.CategoryID)
	if err != nil {
		page.CategoryName = "Unknown"
		page.SubcategoryName = "Unknown"
		return err
	}
	page.SubcategoryName = subCategory.Name
	if subCategory.ParentID != nil {
		parentCategory, err := s.categoryRepo.GetByID(*subCategory.ParentID)
		if err != nil {
			page.CategoryName = "Unknown"
			return err
		}
		page.CategoryName = parentCategory.Name
	} else {
		page.CategoryName = "Uncategorized"
	}
	return nil
}
