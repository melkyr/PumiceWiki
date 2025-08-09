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
// It uses a RETURNING clause to get the generated ID and timestamps and updates the provided page object.
func (r *SQLPageRepository) CreatePage(ctx context.Context, page *Page) error {
	query := `INSERT INTO pages (title, content, author_id) VALUES (:title, :content, :author_id) RETURNING *`

	// Use NamedQuery which is suitable for RETURNING clauses.
	rows, err := r.db.NamedQueryContext(ctx, query, page)
	if err != nil {
		return fmt.Errorf("failed to execute create page query: %w", err)
	}
	defer rows.Close()

	// The RETURNING clause should give us exactly one row.
	if rows.Next() {
		// Scan the returned data back into the page struct.
		if err := rows.StructScan(page); err != nil {
			return fmt.Errorf("failed to scan returned data from create page: %w", err)
		}
	} else {
		return fmt.Errorf("no row returned after insert")
	}

	return nil
}

// GetPageByTitle retrieves a single page from the database by its title.
func (r *SQLPageRepository) GetPageByTitle(ctx context.Context, title string) (*Page, error) {
	var page Page
	query := `SELECT * FROM pages WHERE title = ?`
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
	query := `SELECT * FROM pages WHERE id = ?`
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
	query := `UPDATE pages SET title = :title, content = :content, updated_at = :updated_at WHERE id = :id`
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
