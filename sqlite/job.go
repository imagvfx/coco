package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

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
			commands,
			assignee,
			retry
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ord,
		t.Num,
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
	rows, err := tx.Query(`
		SELECT
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
		j.Order,
	)
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

func (s *JobService) UpdateTask(t coco.TaskUpdater) error {
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

func updateTask(tx *sql.Tx, t coco.TaskUpdater) error {
	keys := []string{}
	vals := []interface{}{}
	if t.Status != nil {
		keys = append(keys, "status = ?")
		vals = append(vals, t.Status)
	}
	if t.Assignee != nil {
		keys = append(keys, "assignee = ?")
		vals = append(vals, t.Assignee)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one parameter to update")
	}
	vals = append(vals, t.Order, t.Num)
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
