package data

import "time"

// Page represents a single wiki page in the database.
type Page struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	AuthorID  string    `db:"author_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
