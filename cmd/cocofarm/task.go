package main

import (
	"encoding/json"
	"fmt"
)

type TaskStatus int

const (
	TaskWaiting = TaskStatus(iota)
	TaskRunning
	TaskCancelled
	TaskFailed
	TaskDone
)

var leafStr = map[TaskStatus]string{
	TaskWaiting:   "waiting",
	TaskRunning:   "running",
	TaskCancelled: "cancelled",
	TaskFailed:    "failed",
	TaskDone:      "done",
}

var branchStr = map[TaskStatus]string{
	TaskWaiting:   "waiting",
	TaskRunning:   "processing",
	TaskCancelled: "cancelled",
	TaskFailed:    "blocked",
	TaskDone:      "done",
}

func (s TaskStatus) String(isLeaf bool) string {
	if isLeaf {
		return leafStr[s]
	}
	return branchStr[s]
}

type branchStat struct {
	nFailed  int
	nRunning int
	nWaiting int
	nDone    int
}

func (st *branchStat) Add(s TaskStatus) {
	switch s {
	case TaskFailed:
		st.nFailed += 1
	case TaskRunning:
		st.nRunning += 1
	case TaskWaiting:
		st.nWaiting += 1
	case TaskDone:
		st.nDone += 1
	default:
		panic(fmt.Sprintf("unknown TaskStatus: %v", s))
	}
}

func (st *branchStat) Sub(s TaskStatus) {
	switch s {
	case TaskFailed:
		st.nFailed -= 1
	case TaskRunning:
		st.nRunning -= 1
	case TaskWaiting:
		st.nWaiting -= 1
	case TaskDone:
		st.nDone -= 1
	default:
		panic(fmt.Sprintf("unknown TaskStatus: %v", s))
	}
}

func (st *branchStat) Status() TaskStatus {
	if st.nFailed > 0 {
		return TaskFailed
	}
	if st.nRunning > 0 {
		return TaskRunning
	}
	if st.nWaiting > 0 {
		return TaskWaiting
	}
	return TaskDone
}

// Task has a command and/or subtasks that will be run by workers.
//
// Task having subtasks are called Branch.
// Others are called Leaf, which can have commands.
// Leaf not having commands are valid, but barely useful.
// A Task is either a Branch or a Leaf. It cannot be both at same time.
type Task struct {
	// NOTE: Private fields of this struct should be read-only after the initialization.
	// Otherwise, this program will get racy.

	// id is a Task identifier make it distinct from all other tasks.
	id string

	// job is a job the task is belong to.
	job *Job

	// parent is a parent task of the task.
	// It will be nil, if the task is a root task.
	parent *Task

	// num is internal order of the task in a job.
	num int

	// Title is human readable title for task.
	// Empty Title is allowed.
	Title string

	// status indicates task status using in the farm.
	// It should not be set from user.
	Status TaskStatus

	Stat *branchStat

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

	// Commands are guaranteed that they run serially from a same worker.
	Commands []Command
}

func (t *Task) MarshalJSON() ([]byte, error) {
	m := struct {
		Title          string
		ID             string
		Status         string
		Priority       int
		Subtasks       []*Task
		SerialSubtasks bool
		Commands       []Command
	}{
		Title:          t.Title,
		ID:             t.id,
		Status:         t.Status.String(t.IsLeaf()),
		Priority:       t.Priority,
		Subtasks:       t.Subtasks,
		SerialSubtasks: t.SerialSubtasks,
		Commands:       t.Commands,
	}
	return json.Marshal(m)
}

func (t *Task) SetStatus(s TaskStatus) {
	old := t.Status
	t.Status = s
	t.job.Stat.Sub(old)
	t.job.Stat.Add(s)
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
	return t.job.Priority
}

func (t *Task) IsLeaf() bool {
	return len(t.Subtasks) == 0
}
