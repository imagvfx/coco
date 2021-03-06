package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/coco/service"
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
			title TEXT NOT NULL,
			status INTEGER NOT NULL,
			serial_subtasks BOOL NOT NULL,
			commands TEXT NOT NULL,
			assignee TEXT NOT NULL,
			retry INTEGER NOT NULL,
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
func (s *JobService) AddJob(j *service.Job) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()
	id, err := addJob(tx, j)
	if err != nil {
		return -1, err
	}
	for _, t := range j.Tasks {
		t.Job = id
		err := addTask(tx, t)
		if err != nil {
			return -1, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return -1, err
	}
	return id, nil
}

// addJob adds a job into a database.
func addJob(tx *sql.Tx, j *service.Job) (int, error) {
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
	return int(id), nil
}

// addTask adds a task into a database.
// It tasks an order number of its job, because the task doesn't know it yet.
func addTask(tx *sql.Tx, t *service.Task) error {
	_, err := tx.Exec(`
		INSERT INTO tasks (
			ord,
			num,
			parent_num,
			title,
			status,
			serial_subtasks,
			commands,
			assignee,
			retry
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.Job,
		t.Task,
		t.ParentNum,
		t.Title,
		t.Status,
		t.SerialSubtasks,
		t.Commands,
		t.Assignee,
		t.Retry,
	)
	if err != nil {
		return err
	}
	return nil
}

// FindJobs finds jobs those matched with given filter.
func (s *JobService) FindJobs(f service.JobFilter) ([]*service.Job, error) {
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
func findJobs(tx *sql.Tx, f service.JobFilter) ([]*service.Job, error) {
	keys := []string{}
	vals := []interface{}{}
	if f.Target != "" {
		keys = append(keys, "target = ?")
		vals = append(vals, f.Target)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}

	rows, err := tx.Query(`
		SELECT
			ord,
			target,
			auto_retry
		FROM jobs
		`+where+`
		ORDER BY ord ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*service.Job, 0)
	for rows.Next() {
		j := &service.Job{}
		err := rows.Scan(
			&j.Job,
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
func attachTasks(tx *sql.Tx, j *service.Job) error {
	rows, err := tx.Query(`
		SELECT
			ord,
			num,
			parent_num,
			title,
			status,
			serial_subtasks,
			commands
		FROM tasks
		WHERE
			ord = ?
		ORDER BY num ASC
	`,
		j.Job,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	tasks := make([]*service.Task, 0)
	for rows.Next() {
		t := &service.Task{}
		err := rows.Scan(
			&t.Job,
			&t.Task,
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

func (s *JobService) UpdateTask(t service.TaskUpdater) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateTask(tx, t)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateTask(tx *sql.Tx, t service.TaskUpdater) error {
	keys := []string{}
	vals := []interface{}{}
	if t.UpdateStatus {
		keys = append(keys, "status = ?")
		vals = append(vals, t.Status)
	}
	if t.UpdateAssignee {
		keys = append(keys, "assignee = ?")
		vals = append(vals, t.Assignee)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one parameter to update")
	}
	vals = append(vals, t.Job, t.Task)
	_, err := tx.Exec(`
		UPDATE tasks
		SET `+strings.Join(keys, ", ")+`
		WHERE
			ord = ? AND
			num = ?
	`,
		vals...,
	)
	return err
}
