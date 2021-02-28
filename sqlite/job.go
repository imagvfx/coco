package sqlite

import (
	"database/sql"

	"github.com/imagvfx/coco"
)

func CreateJobsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target TEXT NOT NULL,
			auto_retry INTEGER NOT NULL
		);
	`)
	return err
}

type JobService struct {
	db *sql.DB
}

func NewJobService(db *sql.DB) *JobService {
	return &JobService{db: db}
}

func (s *JobService) AddJob(j *coco.SQLJob) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()
	ord, err := addJob(tx, j)
	if err != nil {
		return -1, err
	}
	err = tx.Commit()
	if err != nil {
		return -1, err
	}
	return ord, nil
}

func addJob(tx *sql.Tx, j *coco.SQLJob) (int, error) {
	// Don't insert the job's id, it will be generated from db.
	result, err := tx.Exec(`
		INSERT INTO jobs (
			target,
			auto_retry
		)
		VALUES (?, ?)
	`,
		j.Target,
		j.AutoRetry,
	)
	if err != nil {
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	ord := int(id)
	return ord, nil
}
