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

// CreateTasksTable creates tasks table to a database if not exists.
// It is ok to call it multiple times.
func CreateTasksTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			job_id INTEGER NOT NULL,
			num INTEGER NOT NULL,
			parent_num INTEGER NOT NULL,
			status INTEGER NOT NULL,
			serial_subtasks BOOL NOT NULL,
			commands TEXT NOT NULL
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
	jobID, err := addJob(tx, j)
	if err != nil {
		return -1, err
	}
	for _, t := range j.Tasks {
		err := addTask(tx, jobID, t)
		if err != nil {
			return -1, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return -1, err
	}
	return jobID, nil
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
	jobID := int(id)
	return jobID, nil
}

// addTask adds a task to a database.
// It tasks a job ID, because the task doesn't know its job's id yet.
func addTask(tx *sql.Tx, jobID int, t *coco.SQLTask) error {
	_, err := tx.Exec(`
		INSERT INTO tasks (
			job_id,
			num,
			parent_num,
			status,
			serial_subtasks,
			commands
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		jobID,
		t.Num,
		t.ParentNum,
		t.Status,
		t.SerialSubtasks,
		t.Commands,
	)
	if err != nil {
		return err
	}
	return nil
}
