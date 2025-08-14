package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SQLPageRepository is a concrete implementation of the PageRepository interface using sqlx.
type SQLPageRepository struct {
	db *sqlx.DB
}

// NewSQLPageRepository creates a new SQLPageRepository.
func NewSQLPageRepository(db *sqlx.DB) *SQLPageRepository {
	return &SQLPageRepository{db: db}
}

// CreatePage inserts a new page into the database.
// Note: MariaDB (MySQL) does not support a RETURNING clause for inserts in the same
// way as PostgreSQL. This function inserts the data and assumes the database
// will correctly handle auto-incrementing IDs and default timestamps.
// The provided 'page' object is not updated with DB-generated values post-insert.
func (r *SQLPageRepository) CreatePage(ctx context.Context, page *Page) error {
	query := `INSERT INTO pages (title, content, author_id, category_id) VALUES (:title, :content, :author_id, :category_id)`
	_, err := r.db.NamedExecContext(ctx, query, page)
	if err != nil {
		return fmt.Errorf("failed to execute create page query: %w", err)
	}
	// To get the ID, a separate SELECT would be needed, but for now, we assume
	// the caller doesn't need the ID immediately after creation.
	return nil
}

// GetPageByTitle retrieves a single page from the database by its title.
func (r *SQLPageRepository) GetPageByTitle(ctx context.Context, title string) (*Page, error) {
	var page Page
	query := `SELECT id, title, content, author_id, created_at, updated_at, category_id FROM pages WHERE title = ?`
	if err := r.db.GetContext(ctx, &page, query, title); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("page with title '%s' not found", title)
		}
		return nil, fmt.Errorf("failed to get page by title: %w", err)
	}
	return &page, nil
}

// GetPageByID retrieves a single page from the database by its ID.
func (r *SQLPageRepository) GetPageByID(ctx context.Context, id int64) (*Page, error) {
	var page Page
	query := `SELECT id, title, content, author_id, created_at, updated_at, category_id FROM pages WHERE id = ?`
	if err := r.db.GetContext(ctx, &page, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("page with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get page by id: %w", err)
	}
	return &page, nil
}

// UpdatePage updates an existing page in the database.
func (r *SQLPageRepository) UpdatePage(ctx context.Context, page *Page) error {
	query := `UPDATE pages SET title = :title, content = :content, updated_at = :updated_at, category_id = :category_id WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, page)
	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no page found to update with id %d", page.ID)
	}
	return nil
}

// GetPagesByCategoryID retrieves all pages associated with a given category ID.
func (r *SQLPageRepository) GetPagesByCategoryID(ctx context.Context, categoryID int64) ([]*Page, error) {
	var pages []*Page
	query := `SELECT id, title, content, author_id, created_at, updated_at, category_id FROM pages WHERE category_id = ?`
	if err := r.db.SelectContext(ctx, &pages, query, categoryID); err != nil {
		return nil, fmt.Errorf("failed to get pages by category id: %w", err)
	}
	return pages, nil
}

// GetAllPages retrieves all pages from the database.
func (r *SQLPageRepository) GetAllPages(ctx context.Context) ([]*Page, error) {
	var pages []*Page
	query := `SELECT id, title, content, author_id, created_at, updated_at, category_id FROM pages`
	if err := r.db.SelectContext(ctx, &pages, query); err != nil {
		return nil, fmt.Errorf("failed to get all pages: %w", err)
	}
	return pages, nil
}

// DeletePage removes a page from the database by its ID.
func (r *SQLPageRepository) DeletePage(ctx context.Context, id int64) error {
	query := `DELETE FROM pages WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no page found to delete with id %d", id)
	}
	return nil
}
