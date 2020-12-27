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

// taskWalker walks through the task and it's subtasks recursively.
type taskWalker struct {
	ch   chan *Task
	next *Task
}

func newTaskWalker(t *Task) *taskWalker {
	w := &taskWalker{}
	w.ch = make(chan *Task)
	go walk(t, w.ch)
	w.next = <-w.ch
	return w
}

func (w *taskWalker) Next() *Task {
	next := w.next
	w.next = <-w.ch
	return next
}

func (w *taskWalker) Peek() *Task {
	return w.next
}

func walk(t *Task, ch chan *Task) {
	walkR(t, ch)
	close(ch)
}

func walkR(t *Task, ch chan<- *Task) {
	for _, t := range t.Subtasks {
		walkR(t, ch)
	}
	ch <- t
}
