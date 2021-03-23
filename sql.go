package coco

type Services interface {
	FarmService() FarmService
	JobService() JobService
	WorkerService() WorkerService
}

type FarmService interface {
	UpdateAssign(AssignUpdater) error
}

type AssignUpdater struct {
	Task               TaskID
	UpdateTaskStatus   bool
	TaskStatus         TaskStatus
	UpdateTaskRetry    bool
	TaskRetry          int
	UpdateTaskAssignee bool
	TaskAssignee       string

	Worker             string
	UpdateWorkerStatus bool
	WorkerStatus       WorkerStatus
}

// JobService is an interface which let us use sqlite.JobService.
type JobService interface {
	AddJob(*SQLJob) (JobID, error)
	FindJobs(JobFilter) ([]*SQLJob, error)
	UpdateTask(TaskUpdater) error
}

// NopJobService is a JobService which does nothing.
// We need this for testing.
type NopJobService struct{}

// AddJob returns (0, nil) always.
func (s *NopJobService) AddJob(j *SQLJob) (JobID, error) {
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
	ID        JobID
	Target    string
	AutoRetry int
	Tasks     []*SQLTask
}

// SQLTask is a task information for sql database.
type SQLTask struct {
	ID             TaskID
	ParentNum      int
	Title          string
	Status         TaskStatus
	SerialSubtasks bool
	Commands       Commands
	Assignee       string
	Retry          int
}

// JobFilter is a job filter for searching jobs.
type JobFilter struct {
	Target string
}

// TaskUpdater has information for updating a task.
type TaskUpdater struct {
	ID             TaskID
	UpdateStatus   bool
	Status         TaskStatus
	UpdateAssignee bool
	Assignee       string
	UpdateRetry    bool
	Retry          int
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
	Task   *TaskID
}

// WorkerFilter is a job filter for searching workers.
type WorkerFilter struct {
	Addr string
}

// WorkerUpdater has information for updating a worker.
type WorkerUpdater struct {
	Addr         string
	UpdateStatus bool
	Status       WorkerStatus
	UpdateTask   bool
	Task         *TaskID
}
