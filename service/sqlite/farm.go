package sqlite

import (
	"database/sql"

	"github.com/imagvfx/coco/service"
)

// FarmService interacts with a database for coco.
type FarmService struct {
	db *sql.DB
}

// NewFarmService creates a new FarmService.
func NewFarmService(db *sql.DB) *FarmService {
	return &FarmService{db: db}
}

func (s *FarmService) UpdateAssign(a service.AssignUpdater) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateAssign(tx, a)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateAssign(tx *sql.Tx, a service.AssignUpdater) error {
	err := updateTask(tx, service.TaskUpdater{
		Job:            a.Job,
		Task:           a.Task,
		UpdateStatus:   a.UpdateTaskStatus,
		Status:         a.TaskStatus,
		UpdateRetry:    a.UpdateTaskRetry,
		Retry:          a.TaskRetry,
		UpdateAssignee: a.UpdateTaskAssignee,
		Assignee:       a.TaskAssignee,
	})
	if err != nil {
		return err
	}
	err = updateWorker(tx, service.WorkerUpdater{
		Addr:         a.Worker,
		UpdateStatus: a.UpdateWorkerStatus,
		Status:       a.WorkerStatus,
		UpdateTask:   a.UpdateWorkerTask,
		Job:          a.WorkerJob,
		Task:         a.WorkerTask,
	})
	if err != nil {
		return err
	}
	return nil
}
