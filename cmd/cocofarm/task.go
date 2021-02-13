package main

import (
	"encoding/json"
	"fmt"
)

type TaskStatus int

const (
	TaskWaiting = TaskStatus(iota)
	TaskRunning
	TaskFailed
	TaskDone
)

func (s TaskStatus) String() string {
	return map[TaskStatus]string{
		TaskWaiting: "waiting",
		TaskRunning: "running",
		TaskFailed:  "failed",
		TaskDone:    "done",
	}[s]
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

type TaskID string

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
	id TaskID

	// job is a job the task is belong to.
	job *Job

	// parent is a parent task of the task.
	// It will be nil, if the task is a root task.
	parent *Task

	// nthChild indicates this task is nth child of the parent task.
	nthChild int

	// next is the next task for walking a job's tasks.
	// When it is nil, the task is last task of the job.
	next *Task

	// num is internal order of the leaf task in a job.
	// Only leaf task has value in num.
	num int

	// Title is human readable title for task.
	// Empty Title is allowed.
	Title string

	// status indicates task status using in the farm.
	// It should not be set from user.
	status TaskStatus

	// Stat aggrigates it's leafs status.
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

	popIdx int
}

func (t *Task) MarshalJSON() ([]byte, error) {
	m := struct {
		Title          string
		ID             TaskID
		Status         string
		Priority       int
		Subtasks       []*Task
		SerialSubtasks bool
		Commands       []Command
	}{
		Title:          t.Title,
		ID:             t.id,
		Status:         t.Status().String(),
		Priority:       t.Priority,
		Subtasks:       t.Subtasks,
		SerialSubtasks: t.SerialSubtasks,
		Commands:       t.Commands,
	}
	return json.Marshal(m)
}

func (t *Task) Blocking() bool {
	if !t.IsLeaf() {
		panic("shouldn't call Task.Blocking on non-leaf task")
	}
	// A parallel leaf task will always return done == true.
	// On the other hand a serial leaf task will return done == true,
	// only when the task has really finished.
	block := false
	if t.parent.SerialSubtasks && t.status != TaskDone {
		block = true
	}
	if t.status == TaskFailed {
		// Make the task block any leaf task on the next stages.
		// For a temporary error, user can restart the task to make it done.
		block = true
	}
	return block
}

func (t *Task) Pop() (*Task, bool) {
	if t.IsLeaf() {
		block := t.Blocking()
		if t.popIdx != -1 {
			t.popIdx = -1
			return t, !block
		}
		return nil, !block
	}
	// branch
	if t.popIdx < 0 {
		return nil, true
	}
	i := t.popIdx
	var popt *Task
	alldone := true // all done until the subtask
	for i < len(t.Subtasks) {
		subt := t.Subtasks[i]
		p, done := subt.Pop()
		popt = p
		if done {
			if alldone {
				// caching the result for next pop
				t.popIdx = i + 1
			}
			// this subtask has done, but one of the prior task hasn't done yet.
			// cannot jump to this subtask.
		} else {
			alldone = false
			if t.SerialSubtasks {
				// should wait the subtask has done
				break
			}
		}
		if p != nil {
			break
		}
		i++
	}
	if t.popIdx == len(t.Subtasks) {
		t.popIdx = -1
	}
	return popt, t.popIdx == -1
}

func (t *Task) Peek() *Task {
	if t.popIdx == -1 {
		// t and it's subtasks has all done
		return nil
	}
	popt := t
	for !popt.IsLeaf() {
		// There should be no popIdx == -1, if t hasn't done yet.
		popt = popt.Subtasks[popt.popIdx]
	}
	if popt.Blocking() {
		return nil
	}
	return popt
}

// Push pushes the popped task to the job again.
// This can be used to retry of the task.
// Set the task's status to TaskWaiting, first. Otherwise, it will panic.
func (t *Task) Push() {
	if !t.IsLeaf() {
		panic("cannot push a branch task")
	}
	if t.status != TaskWaiting {
		// TODO: is the panic best?
		panic("the task's status should be TaskWaiting when pushed")
	}
	t.popIdx = 0
	parent := t.parent
	child := t
	for parent != nil {
		n := child.nthChild
		if parent.popIdx == -1 || n < parent.popIdx {
			parent.popIdx = n
		}
		child = parent
		parent = parent.parent
	}
}

func (t *Task) Status() TaskStatus {
	if t.IsLeaf() {
		return t.status
	}
	return t.Stat.Status()
}

func (t *Task) SetStatus(s TaskStatus) {
	if !t.IsLeaf() {
		panic("cannot set status to a branch task")
	}
	old := t.status
	t.status = s
	parent := t.parent
	for parent != nil {
		parent.Stat.Sub(old)
		parent.Stat.Add(s)
		parent = parent.parent
	}
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
	// the job's Priority was 0.
	return 0
}

func (t *Task) IsLeaf() bool {
	return len(t.Subtasks) == 0
}
