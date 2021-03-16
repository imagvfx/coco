package coco

// JobService is an interface which let us use sqlite.JobService.
type JobService interface {
	AddJob(*SQLJob) (int, error)
	FindJobs(JobFilter) ([]*SQLJob, error)
	UpdateTask(TaskUpdater) error
}

// NopJobService is a JobService which does nothing.
// We need this for testing.
type NopJobService struct{}

// AddJob returns (0, nil) always.
func (s *NopJobService) AddJob(j *SQLJob) (int, error) {
	return 0, nil
}

// UpdateTask returns nil.
func (s *NopJobService) UpdateTask(TaskUpdater) error {
	return nil
}

// FindJobs returns (nil, nil).
func (s *NopJobService) FindJobs(f JobFilter) ([]*SQLJob, error) {
	return nil, nil
}

// SQLJob is a job information for sql database.
type SQLJob struct {
	// TODO: Do we need ID here? It will be generated from a db.
	Order     int
	Target    string
	AutoRetry int
	Tasks     []*SQLTask
}

// JobFilter is a job filter for searching jobs.
type JobFilter struct {
	Target string
}

// TaskUpdater has information for updating a task.
type TaskUpdater struct {
	Order    int
	Num      int
	Status   *TaskStatus
	Assignee *string
}

// WorkerService is an interface which let us use sqlite.WorkerService.
type WorkerService interface {
	AddWorker(*SQLWorker) error
	FindWorkers(WorkerFilter) ([]*SQLWorker, error)
	UpdateWorker(WorkerUpdater) error
}

// NopWorkerService is a WorkerService which does nothing.
// We need this for testing.
type NopWorkerService struct{}

// AddWorker returns (0, nil) always.
func (s *NopWorkerService) AddWorker(w *SQLWorker) error {
	return nil
}

// UpdateWorker returns nil.
func (s *NopWorkerService) UpdateWorker(w WorkerUpdater) error {
	return nil
}

// FindWorkers returns (nil, nil).
func (s *NopWorkerService) FindWorkers(f WorkerFilter) ([]*SQLWorker, error) {
	return nil, nil
}

// SQLWorker is a job information for sql database.
type SQLWorker struct {
	Addr   string
	Status WorkerStatus
	Task   string
}

// WorkerFilter is a job filter for searching workers.
type WorkerFilter struct {
	Addr string
}

// WorkerUpdater has information for updating a worker.
type WorkerUpdater struct {
	Addr string

	Status *WorkerStatus
	Task   *string
}
