package main

type TaskStatus int

const (
	TaskWaiting = TaskStatus(iota)
	TaskRunning
	TaskFailed
	TaskDone
)

// Task has a command and/or subtasks that will be run by workers.
type Task struct {
	// Title is human readable title for task.
	// Empty Title is allowed.
	Title string

	// job is a job the task is belong to.
	job *Job

	// parent is a parent task of the task.
	// it will be nil, if the task is a root task.
	parent *Task

	// num is internal order of the task in a job.
	num int

	// status indicates task status using in the farm.
	// It should not be set from user.
	status TaskStatus

	// Priority is a priority hint for the task.
	// Priority set to zero makes it inherit nearest parent that has non-zero priority.
	// If there isn't non-zero priority parent, it will use the job's priority.
	// Negative values will be considered as false value, and treated as zero.
	//
	// NOTE: Use CalcPriority to get the real priority of the task.
	Priority int

	// Subtasks contains subtasks to be run.
	// Subtasks could be nil or empty.
	Subtasks []*Task

	// When true, a subtask will be launched after the prior task finished.
	// When false, a subtask will be launched right after the prior task started.
	SerialSubtasks bool

	// Commands will be executed after SubTasks run.
	// Commands are guaranteed that they run serially from a same worker.
	Commands []Command
}

func (t *Task) Status() TaskStatus {
	return t.status
}

func (t *Task) CalcPriority() int {
	tt := t
	for tt != nil {
		p := tt.Priority
		if p != 0 {
			return p
		}
		tt = tt.parent
	}
	return t.job.DefaultPriority
}

func walkTaskFn(t *Task, fn func(t *Task)) {
	fn(t)
	for _, t := range t.Subtasks {
		walkTaskFn(t, fn)
	}
}
