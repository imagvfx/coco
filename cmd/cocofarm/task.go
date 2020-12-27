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

	// parent is a parent task of the task.
	// it will be nil, if the task is a root task.
	parent *Task

	// num is internal order of the task in a job.
	num int

	// status indicates task status using in the farm.
	// It should not be set from user.
	status TaskStatus

	// Priority will change it's job priority temporarily for this task.
	// Priority set to zero makes it inherit the parent's priority.
	// Negative values will be considered as false value, and treated as zero.
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

func walkTaskFn(t *Task, fn func(t *Task)) {
	fn(t)
	for _, t := range t.Subtasks {
		walkTaskFn(t, fn)
	}
}
