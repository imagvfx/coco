package main

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// ID lets a Job distinguishes from others.
	ID int

	// Title is human readable title for job.
	Title string

	// Priority sets the job's default priority.
	// Jobs are compete each other with priority.
	// Job's priority could be temporarily updated by a task that waits at the time.
	// Higher values take precedence to lower values.
	// Negative values will corrected to 0, the lowest priority value.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	Priority int

	// Root task contains commands or subtasks to be run.
	Root *Task
}

// Task has a command and/or subtasks that will be run by workers.
type Task struct {
	// Title is human readable title for task.
	// Empty Title is allowed.
	Title string

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

// Command is a command to be run in a worker.
// First string is the executable and others are arguments.
// When a Command is nil or empty, the command will be skipped.
type Command []string

func main() {
}
