package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens a db at path.
// It will check stat of the db file before open it.
// It returns an error if the check or openning of the db failed.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		fmt.Errorf("db path required")
	}
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Enable Write-Ahead Logging. See https://sqlite.org/wal.html
	if _, err := db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	// Enable foreign key checks.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("foreign keys pragma: %w", err)
	}
	return db, nil
}

// Create creates a new initialized db.
// It returns an error if failed to create the db.
func Create(path string) (*sql.DB, error) {
	if path == "" {
		fmt.Errorf("db path required")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	err = CreateJobsTable(tx)
	if err != nil {
		return nil, err
	}
	err = CreateTasksTable(tx)
	if err != nil {
		return nil, err
	}
	err = CreateWorkersTable(tx)
	if err != nil {
		return nil, err
	}
	return db, tx.Commit()
}
