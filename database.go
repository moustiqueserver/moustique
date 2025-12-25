package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// Database handles persistent storage
type Database struct {
	mu     sync.RWMutex
	db     *sql.DB
	values map[string]string
	dbPath string
}

// NewDatabase creates a new database instance
func NewDatabase(path string) (*Database, error) {
	// Skapa katalogen om den inte finns (t.ex. för "data/app.db" skapar den "data/")
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv (
		key TEXT PRIMARY KEY,
		value TEXT
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &Database{
		db:     db,
		values: make(map[string]string),
		dbPath: path,
	}, nil
}

// LoadAll loads all values from database into memory
func (d *Database) LoadAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query("SELECT key, value FROM kv")
	if err != nil {
		return fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		d.values[key] = value
		count++
	}

	fmt.Printf("Loaded %d keys from SQLite\n", count)
	return rows.Err()
}

// SaveAll saves all in-memory values to database
func (d *Database) SaveAll() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for key, value := range d.values {
		if _, err := stmt.Exec(key, value); err != nil {
			return fmt.Errorf("failed to insert key %s: %w", key, err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Saved %d keys to SQLite\n", count)
	return nil
}

// SaveValue saves a single value (in-memory and DB)
func (d *Database) SaveValue(key string, value interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	d.values[key] = string(jsonData)

	// Also persist to DB immediately for important updates
	/*_, err = d.db.Exec("INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)",
		key, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save to db: %w", err)
	}*/

	return nil
}

// GetValue retrieves a value by key
func (d *Database) GetValue(key string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	value, exists := d.values[key]
	if !exists {
		return "", fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

// HasValue checks if a key exists
func (d *Database) HasValue(key string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	_, exists := d.values[key]
	return exists
}

// CountValues returns the number of stored values
func (d *Database) CountValues() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.values)
}

// GetKeys returns all keys
func (d *Database) GetKeys() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	keys := make([]string, 0, len(d.values))
	for key := range d.values {
		keys = append(keys, key)
	}
	return keys
}

// GetKeysByRegex returns keys matching a regex pattern
func (d *Database) GetKeysByRegex(re *regexp.Regexp) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var matches []string
	for key := range d.values {
		if re.MatchString(key) {
			matches = append(matches, key)
		}
	}
	return matches
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}
