package sqlite

import (
	"database/sql"
	"fmt"

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
			title TEXT,
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
			title,
			status,
			serial_subtasks,
			commands
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		ord,
		t.Num,
		t.ParentNum,
		t.Title,
		t.Status,
		t.SerialSubtasks,
		t.Commands,
	)
	if err != nil {
		return err
	}
	return nil
}

// FindJobs finds jobs those matched with given filter.
func (s *JobService) FindJobs(f coco.JobFilter) ([]*coco.SQLJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	jobs, err := findJobs(tx, f)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		err := attachTasks(tx, j)
		if err != nil {
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// findJobs finds jobs those matched with given filter.
func findJobs(tx *sql.Tx, f coco.JobFilter) ([]*coco.SQLJob, error) {
	wh := NewWhere()
	if f.Target != "" {
		wh.Add("target", f.Target)
	}

	q := fmt.Sprintf(`
		SELECT
			ord,
			target,
			auto_retry
		FROM jobs
		%v
		ORDER BY ord ASC
	`, wh.Stmt())

	rows, err := tx.Query(q, wh.Vals()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*coco.SQLJob, 0)
	for rows.Next() {
		j := &coco.SQLJob{}
		err := rows.Scan(
			&j.Order,
			&j.Target,
			&j.AutoRetry,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// attachTasks attach all tasks to it's job.
func attachTasks(tx *sql.Tx, j *coco.SQLJob) error {
	wh := NewWhere()
	wh.Add("ord", j.Order)

	q := fmt.Sprintf(`
		SELECT
			num,
			parent_num,
			title,
			status,
			serial_subtasks,
			commands
		FROM tasks
		%v
		ORDER BY num ASC
	`, wh.Stmt())
	rows, err := tx.Query(q, wh.Vals()...)
	if err != nil {
		return err
	}
	defer rows.Close()

	tasks := make([]*coco.SQLTask, 0)
	for rows.Next() {
		t := &coco.SQLTask{
			Order: j.Order,
		}
		err := rows.Scan(
			&t.Num,
			&t.ParentNum,
			&t.Title,
			&t.Status,
			&t.SerialSubtasks,
			&t.Commands,
		)
		if err != nil {
			return err
		}
		tasks = append(tasks, t)
	}
	j.Tasks = tasks
	return nil
}
