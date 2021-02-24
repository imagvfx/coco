package sqlite

import (
	"database/sql"
	"fmt"
)

type DB struct {
	db   *sql.DB
	path string
}

func NewDB(path string) *DB {
	db := &DB{
		path: path,
	}
	return db
}

func (db *DB) Open() error {
	if db.path == "" {
		fmt.Errorf("db path required")
	}
	innerdb, err := sql.Open("sqlite3", db.path)
	if err != nil {
		return err
	}
	db.db = innerdb
	// Enable Write-Ahead Logging. See https://sqlite.org/wal.html
	if _, err := db.db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}
	// Eanble foreign key checks.
	if _, err := db.db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("foreign keys pragma: %w", err)
	}
	return nil
}

func (db *DB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}
