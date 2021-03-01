package sqlite

import (
	"database/sql"

	"github.com/imagvfx/coco"
)

// CreateJobsTable creates jobs table to a database if not exists.
// It is ok to call it multiple times.
func CreateJobsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			ord INTEGER PRIMARY KEY AUTOINCREMENT,
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
			ord INTEGER NOT NULL,
			num INTEGER NOT NULL,
			parent_num INTEGER NOT NULL,
			status INTEGER NOT NULL,
			serial_subtasks BOOL NOT NULL,
			commands TEXT NOT NULL,
			PRIMARY KEY (ord, num)
		);
	`)
	return err
}

// JobService interacts with a database for coco jobs.
type JobService struct {
	db *sql.DB
}

// NewJobService creates a new JobService.
func NewJobService(db *sql.DB) *JobService {
	return &JobService{db: db}
}

// AddJob adds a job and it's tasks into a database.
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
	for _, t := range j.Tasks {
		err := addTask(tx, ord, t)
		if err != nil {
			return -1, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return -1, err
	}
	return ord, nil
}

// addJob adds a job into a database.
func addJob(tx *sql.Tx, j *coco.SQLJob) (int, error) {
	// Don't insert the job's order number, it will be generated from db.
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

// addTask adds a task into a database.
// It tasks an order number of its job, because the task doesn't know it yet.
func addTask(tx *sql.Tx, ord int, t *coco.SQLTask) error {
	_, err := tx.Exec(`
		INSERT INTO tasks (
			ord,
			num,
			parent_num,
			status,
			serial_subtasks,
			commands
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		ord,
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
