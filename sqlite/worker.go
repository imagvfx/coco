package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/imagvfx/coco"
)

// CreateWorkersTable creates jobs table to a database if not exists.
// It is ok to call it multiple times.
func CreateWorkersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS workers (
			addr TEXT PRIMARY KEY,
			status INT NOT NULL,
			task TEXT NOT NULL
		);
	`)
	return err
}

// WorkerService interacts with a database for coco workers.
type WorkerService struct {
	db *sql.DB
}

// NewWorkerService creates a new WorkerService.
func NewWorkerService(db *sql.DB) *WorkerService {
	return &WorkerService{db: db}
}

// AddWorker adds a worker into a database.
func (s *WorkerService) AddWorker(w *coco.SQLWorker) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addWorker(tx, w)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// addWorker adds a job into a database.
func addWorker(tx *sql.Tx, w *coco.SQLWorker) error {
	// Don't insert the job's order number, it will be generated from db.
	_, err := tx.Exec(`
		INSERT INTO workers (
			addr,
			status,
			task
		)
		VALUES (?, ?, ?)
	`,
		w.Addr,
		w.Status,
		w.Task,
	)
	return err
}

// FindAllWorkers finds jobs those matched with given filter.
func (s *WorkerService) FindWorkers(f coco.WorkerFilter) ([]*coco.SQLWorker, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	workers, err := findWorkers(tx, f)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return workers, nil
}

// findWorkers finds workers those matched with given filter.
func findWorkers(tx *sql.Tx, w coco.WorkerFilter) ([]*coco.SQLWorker, error) {
	wh := NewWhere()
	if w.Addr != "" {
		wh.Add("addr", w.Addr)
	}

	rows, err := tx.Query(`
		SELECT
			addr,
			status,
			task
		FROM workers
		`+wh.Stmt()+`
		ORDER BY addr ASC
	`,
		wh.Vals()...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workers := make([]*coco.SQLWorker, 0)
	for rows.Next() {
		w := &coco.SQLWorker{}
		err := rows.Scan(
			&w.Addr,
			&w.Status,
			&w.Task,
		)
		if err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, nil
}

func (s *WorkerService) UpdateWorker(w coco.WorkerUpdater) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateWorker(tx, w)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateWorker(tx *sql.Tx, w coco.WorkerUpdater) error {
	setStmt := "SET "
	setVals := make([]interface{}, 0)
	if w.Status != nil {
		setStmt += "status = ?"
		setVals = append(setVals, w.Status)
	}
	if w.Task != nil {
		if len(setVals) != 0 {
			setStmt += ", "
		}
		setStmt += "task = ?"
		setVals = append(setVals, w.Task)
	}
	if len(setVals) == 0 {
		return fmt.Errorf("need at least one parameter to update")
	}
	vals := append(setVals, w.Addr)
	_, err := tx.Exec(`
		UPDATE workers
		`+setStmt+`
		WHERE
			addr = ?
	`,
		vals...,
	)
	return err
}
