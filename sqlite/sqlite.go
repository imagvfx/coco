package sqlite

import (
	"database/sql"
	"fmt"
)

func Open(path string) (*sql.DB, error) {
	if path == "" {
		fmt.Errorf("db path required")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Enable Write-Ahead Logging. See https://sqlite.org/wal.html
	if _, err := db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}
	// Eanble foreign key checks.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("foreign keys pragma: %w", err)
	}
	return db, nil
}
