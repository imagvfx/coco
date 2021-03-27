package service

type Services interface {
	FarmService() FarmService
	JobService() JobService
	WorkerService() WorkerService
}

type FarmService interface {
	UpdateAssign(AssignUpdater) error
}

type AssignUpdater struct {
	Job                int
	Task               int
	UpdateTaskStatus   bool
	TaskStatus         int
	UpdateTaskRetry    bool
	TaskRetry          int
	UpdateTaskAssignee bool
	TaskAssignee       string

	Worker             string
	UpdateWorkerStatus bool
	WorkerStatus       int
	UpdateWorkerTask   bool
	WorkerJob          *int
	WorkerTask         *int
}

// JobService is an interface which let us use sqlite.JobService.
type JobService interface {
	AddJob(*Job) (int, error)
	FindJobs(JobFilter) ([]*Job, error)
	UpdateTask(TaskUpdater) error
}

// WorkerService is an interface which let us use sqlite.WorkerService.
type WorkerService interface {
	AddWorker(*Worker) error
	FindWorkers(WorkerFilter) ([]*Worker, error)
	UpdateWorker(WorkerUpdater) error
}

// Job is a job information for database service.
type Job struct {
	Job       int
	Target    string
	AutoRetry int
	Tasks     []*Task
}

// Task is a task information for database service.
type Task struct {
	Job            int
	Task           int
	ParentNum      int
	Title          string
	Status         int
	SerialSubtasks bool
	Commands       string
	Assignee       string
	Retry          int
}

// JobFilter is a job filter for searching jobs.
type JobFilter struct {
	Target string
}

// TaskUpdater has information for updating a task.
type TaskUpdater struct {
	Job            int
	Task           int
	UpdateStatus   bool
	Status         int
	UpdateAssignee bool
	Assignee       string
	UpdateRetry    bool
	Retry          int
}

// Worker is a worker information for database service.
type Worker struct {
	Addr   string
	Status int
	Job    *int
	Task   *int
}

// WorkerFilter is a job filter for searching workers.
type WorkerFilter struct {
	Addr string
}

// WorkerUpdater has information for updating a worker.
type WorkerUpdater struct {
	Addr         string
	UpdateStatus bool
	Status       int
	UpdateTask   bool
	Job          *int
	Task         *int
}
