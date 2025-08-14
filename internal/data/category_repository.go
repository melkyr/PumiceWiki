package data

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

// CategoryRepository handles database operations for categories.
type CategoryRepository struct {
	DB *sqlx.DB
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(db *sqlx.DB) *CategoryRepository {
	return &CategoryRepository{DB: db}
}

// FindByName finds a category by name and parent ID.
func (r *CategoryRepository) FindByName(name string, parentID *int64) (*Category, error) {
	var category Category
	var err error
	query := "SELECT id, name, parent_id FROM categories WHERE name = ? AND parent_id "
	if parentID == nil {
		query += "IS NULL"
		err = r.DB.Get(&category, query, name)
	} else {
		query += "= ?"
		err = r.DB.Get(&category, query, name, *parentID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &category, nil
}

// SearchByName searches for categories by name.
func (r *CategoryRepository) SearchByName(query string) ([]*Category, error) {
	var categories []*Category
	err := r.DB.Select(&categories, "SELECT id, name, parent_id FROM categories WHERE name LIKE ?", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// GetAll retrieves all categories from the database.
func (r *CategoryRepository) GetAll() ([]*Category, error) {
	var categories []*Category
	err := r.DB.Select(&categories, "SELECT id, name, parent_id FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// Save creates a new category and returns its ID.
func (r *CategoryRepository) Save(category *Category) (int64, error) {
	res, err := r.DB.NamedExec("INSERT INTO categories (name, parent_id) VALUES (:name, :parent_id)", category)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetByID finds a category by its ID.
func (r *CategoryRepository) GetByID(id int64) (*Category, error) {
	var category Category
	err := r.DB.Get(&category, "SELECT id, name, parent_id FROM categories WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &category, nil
}
