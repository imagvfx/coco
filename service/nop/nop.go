package nop

import "github.com/imagvfx/coco/service"

// JobService is a JobService which does nothing.
// We need this for testing.
type JobService struct{}

// AddJob returns (0, nil) always.
func (s *JobService) AddJob(j *service.Job) (int, error) {
	return 0, nil
}

// UpdateTask returns nil.
func (s *JobService) UpdateTask(service.TaskUpdater) error {
	return nil
}

// FindJobs returns (nil, nil).
func (s *JobService) FindJobs(f service.JobFilter) ([]*service.Job, error) {
	return nil, nil
}

// WorkerService is a WorkerService which does nothing.
// We need this for testing.
type WorkerService struct{}

// AddWorker returns (0, nil) always.
func (s *WorkerService) AddWorker(w *service.Worker) error {
	return nil
}

// UpdateWorker returns nil.
func (s *WorkerService) UpdateWorker(w service.WorkerUpdater) error {
	return nil
}

// FindWorkers returns (nil, nil).
func (s *WorkerService) FindWorkers(f service.WorkerFilter) ([]*service.Worker, error) {
	return nil, nil
}
