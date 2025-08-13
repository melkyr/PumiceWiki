package cache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Cache provides a SQLite-based caching mechanism.
type Cache struct {
	db *sqlx.DB
}

// New creates a new Cache instance.
// It opens the SQLite database at the given file path and ensures the
// cache table is created.
func New(filePath string) (*Cache, error) {
	db, err := sqlx.Connect("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite cache: %w", err)
	}

	// For a cache, WAL mode is generally better for concurrency.
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, fmt.Errorf("failed to set WAL mode on sqlite cache: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS cache (
		key TEXT PRIMARY KEY,
		value BLOB,
		expires_at INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON cache (expires_at);
	`
	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache schema: %w", err)
	}

	return &Cache{db: db}, nil
}

// Get retrieves an item from the cache. It returns nil if the item is not found or is expired.
func (c *Cache) Get(key string) ([]byte, error) {
	var item struct {
		Value     []byte `db:"value"`
		ExpiresAt int64  `db:"expires_at"`
	}
	query := `SELECT value, expires_at FROM cache WHERE key = ?`
	err := c.db.Get(&item, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error for a cache miss.
		}
		return nil, fmt.Errorf("failed to get item from cache: %w", err)
	}

	// Check for expiration
	if time.Now().Unix() > item.ExpiresAt {
		// Item has expired, delete it from the cache (best effort)
		_ = c.Delete(key)
		return nil, nil // Treat as a cache miss
	}

	return item.Value, nil
}

// Set adds an item to the cache with a specific TTL (time-to-live).
func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl).Unix()
	query := `INSERT OR REPLACE INTO cache (key, value, expires_at) VALUES (?, ?, ?)`
	_, err := c.db.Exec(query, key, value, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to set item in cache: %w", err)
	}
	return nil
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) error {
	query := `DELETE FROM cache WHERE key = ?`
	_, err := c.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete item from cache: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}
