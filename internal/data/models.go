package data

import (
	"html/template"
	"time"
)

// Page represents a single wiki page in the database.
type Page struct {
	ID              int64         `db:"id"`
	Title           string        `db:"title"`
	Content         string        `db:"content"`
	HTMLContent     template.HTML `db:"-"`
	AuthorID        string        `db:"author_id"`
	CreatedAt       time.Time     `db:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"`
	CategoryID      *int64        `db:"category_id"`
	CategoryName    string        `db:"-"`
	SubcategoryName string        `db:"-"`
}

// Category represents a category for wiki pages.
type Category struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	ParentID *int64 `db:"parent_id"`
}
