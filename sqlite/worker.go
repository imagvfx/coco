package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/coco"
)

// CreateWorkersTable creates jobs table to a database if not exists.
// It is ok to call it multiple times.
func CreateWorkersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS workers (
			addr TEXT PRIMARY KEY,
			status INT NOT NULL,
			job INT NOT NULL,
			task INT NOT NULL
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
			job,
			task
		)
		VALUES (?, ?, ?, ?)
	`,
		w.Addr,
		w.Status,
		w.Task[0],
		w.Task[1],
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
	keys := []string{}
	vals := []interface{}{}
	if w.Addr != "" {
		keys = append(keys, "addr = ?")
		vals = append(vals, w.Addr)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}

	rows, err := tx.Query(`
		SELECT
			addr,
			status,
			job,
			task
		FROM workers
		`+where+`
		ORDER BY addr ASC
	`,
		vals...,
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
			&w.Task[0],
			&w.Task[1],
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
	keys := []string{}
	vals := []interface{}{}
	if w.Status != nil {
		keys = append(keys, "status = ?")
		vals = append(vals, w.Status)
	}
	if w.Task != nil {
		keys = append(keys, "job = ?")
		vals = append(vals, (*w.Task)[0])
		keys = append(keys, "task = ?")
		vals = append(vals, (*w.Task)[1])
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one parameter to update")
	}
	vals = append(vals, w.Addr)
	_, err := tx.Exec(`
		UPDATE workers
		SET `+strings.Join(keys, ", ")+`
		WHERE
			addr = ?
	`,
		vals...,
	)
	return err
}
