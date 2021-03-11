package coco

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// TaskUpdater has information for updating a task.
type TaskUpdater struct {
	Order    int
	Num      int
	Status   *TaskStatus
	Assignee *string
}

// TaskStatus is a task status.
type TaskStatus int

const (
	TaskWaiting = TaskStatus(iota)
	TaskRunning
	TaskFailed
	TaskDone
)

// String represents TaskStatus as string.
func (s TaskStatus) String() string {
	return map[TaskStatus]string{
		TaskWaiting: "waiting",
		TaskRunning: "running",
		TaskFailed:  "failed",
		TaskDone:    "done",
	}[s]
}

// branchStat calculates the branch's status with Status.
type branchStat struct {
	// n indicates total number of leaves in this branch.
	// It should not be changed.
	n int

	nFailed  int
	nRunning int
	nWaiting int
	nDone    int
}

// newBranchStat creates a new branchStat, that is having n leafs.
func newBranchStat(n int) *branchStat {
	return &branchStat{n: n, nWaiting: n}
}

func (st *branchStat) N() int {
	return st.n
}

// Change changes a leaf child's TaskStatus to another.
func (st *branchStat) Change(from, to TaskStatus) {
	switch from {
	case TaskFailed:
		st.nFailed -= 1
	case TaskRunning:
		st.nRunning -= 1
	case TaskWaiting:
		st.nWaiting -= 1
	case TaskDone:
		st.nDone -= 1
	default:
		panic(fmt.Sprintf("unknown TaskStatus: to: %v", to))
	}

	// those are number of leaf status, make sure that isn't a nagative value.
	if st.nFailed < 0 {
		panic(fmt.Sprintf("nFailed shouldn't be a nagative value"))
	}
	if st.nRunning < 0 {
		panic(fmt.Sprintf("nRunning shouldn't be a nagative value"))
	}
	if st.nWaiting < 0 {
		panic(fmt.Sprintf("nWaiting shouldn't be a nagative value"))
	}
	if st.nDone < 0 {
		panic(fmt.Sprintf("nDone shouldn't be a nagative value"))
	}

	switch to {
	case TaskFailed:
		st.nFailed += 1
	case TaskRunning:
		st.nRunning += 1
	case TaskWaiting:
		st.nWaiting += 1
	case TaskDone:
		st.nDone += 1
	default:
		panic(fmt.Sprintf("unknown TaskStatus: from: %v", from))
	}
}

// Status calcuates the branch's status based on the leaf children's status.
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
	// Job is a job the task is belong to.
	Job *Job

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

	// Priority is a priority hint for the task.
	// Higher values are take precedence to lower values.
	// Priority set to 0 makes it inherit nearest parent that has non-zero priority.
	// If there isn't non-zero priority parent, the task has priority 0.
	// Negative values will be considered as a false value, and will be change to 0.
	//
	// NOTE: Use CalcPriority to get the real priority of the task.
	Priority int

	// Subtasks contains subtasks to be run.
	// Subtasks could be nil or empty.
	Subtasks []*Task

	// When true, a subtask will be launched after the prior task finished.
	// When false, a subtask will be launched right after the prior task started.
	SerialSubtasks bool

	// isLeaf indicates whether the task is a leaf task.
	isLeaf bool

	update func(TaskUpdater) error

	// Commands are guaranteed that they run serially from a same worker.
	Commands Commands

	//
	// NOTE:
	//
	// 	Fields below should be used/changed after hold the Job's lock.
	//
	//

	// status indicates task status using in the farm.
	// It should not be set from user.
	status TaskStatus

	// Stat aggrigates it's leafs status.
	// It is only meaningful when the task is a branch.
	Stat *branchStat

	// popIdx shows which child of the task should be popped.
	// It is -1, if the task is a leaf that has already run, or a branch popped all tasks.
	popIdx int

	// retry represents how many times the task retried automatically due to fail of the task.
	// It will be reset, when user retries the job of the task.
	// It is only meaningful to a leaf task.
	retry int

	// Assignee is a worker who is running the task's commands currently.
	// It is nil except, the task is running.
	// It is only meaningful to a leaf task.
	Assignee string
}

// ID is a Task identifier make it distinct from all other tasks.
func (t *Task) ID() string {
	return strconv.Itoa(t.Job.order) + "-" + strconv.Itoa(t.num)
}

// infoFromTaskID splits a task id string into job ID and task number.
// If the given string isn't valid, it will return an error as a third argument.
func infoFromTaskID(id string) (int, int, error) {
	toks := strings.Split(id, "-")
	if len(toks) != 2 {
		return -1, -1, fmt.Errorf("invalid task id: %v", toks)
	}
	jid, err := strconv.Atoi(toks[0])
	if err != nil {
		return -1, -1, fmt.Errorf("invalid task id: %v", toks)
	}
	tnum, err := strconv.Atoi(toks[1])
	if err != nil {
		return -1, -1, fmt.Errorf("invalid task id: %v", toks)
	}
	return jid, tnum, nil
}

// Commands are commands that are garuanteed to be run from a worker.
type Commands []Command

// Value implements driver.Valuer.
func (cs Commands) Value() (driver.Value, error) {
	v, err := json.Marshal(cs)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Scan implements sql.Scanner.
func (cs *Commands) Scan(v interface{}) error {
	if v == nil {
		return fmt.Errorf("scan commands: nil")
	}
	b := v.([]byte)
	err := json.Unmarshal(b, &cs)
	if err != nil {
		return err
	}
	return nil
}

// Command is a command to be run in a worker.
// First string is the executable and others are arguments.
// When a Command is nil or empty, the command will be skipped.
type Command []string

// MarshalJSON implements json.Marshaller.
func (t *Task) MarshalJSON() ([]byte, error) {
	m := struct {
		Title          string
		ID             string
		Status         string
		Priority       int
		Subtasks       []*Task
		SerialSubtasks bool
		Commands       Commands
	}{
		Title:          t.Title,
		ID:             t.ID(),
		Status:         t.Status().String(),
		Priority:       t.Priority,
		Subtasks:       t.Subtasks,
		SerialSubtasks: t.SerialSubtasks,
		Commands:       t.Commands,
	}
	return json.Marshal(m)
}

// SQLTask is a task information for sql database.
type SQLTask struct {
	Order          int
	Num            int
	ParentNum      int
	Title          string
	Status         TaskStatus
	SerialSubtasks bool
	Commands       Commands
	Assignee       string
}

// ToSQL converts a Task into a SQLTask.
func (t *Task) ToSQL() *SQLTask {
	s := &SQLTask{
		Order:          t.Job.order,
		Num:            t.num,
		Title:          t.Title,
		Status:         t.status,
		SerialSubtasks: t.SerialSubtasks,
		Commands:       t.Commands,
		Assignee:       t.Assignee,
	}
	s.ParentNum = -1
	if t.parent != nil {
		s.ParentNum = t.parent.num
	}
	return s
}

// FromSQL converts a SQLTask into a Task.
func (t *Task) FromSQL(st *SQLTask) {
	t.num = st.Num
	t.Title = st.Title
	t.status = st.Status
	t.SerialSubtasks = st.SerialSubtasks
	t.Commands = st.Commands
}

// Blocking returns a bool value that indicates whether the task is a blocking task.
// A serial task that didn't finished blocks the next task, and a failed task blocks the parent.
func (t *Task) Blocking() bool {
	if !t.isLeaf {
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

// Pop pops a first child task that isn't popped yet.
// The second return value indicates that there is no more task to be popped. (done)
// It will return (nil, true) if no more child task is left.
// It will return (nil, false) if there is remaining tasks, but
// a task cannot be popped due to one of the prior subtask is blocking the process.
func (t *Task) Pop() (*Task, bool) {
	if t.isLeaf {
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

// restorePopIdx restores the task and it's subtasks' popIdx
// from status of it's leaves.
// It is used for a job which just resurrected from db.
func (t *Task) restorePopIdx() {
	if t.isLeaf {
		if t.status == TaskWaiting {
			t.popIdx = 0
			return
		}
		t.popIdx = -1
		return
	}
	// branch
	for _, subt := range t.Subtasks {
		subt.restorePopIdx()
	}
	t.popIdx = -1 // in case all subtasks are popped
	for i, subt := range t.Subtasks {
		if subt.popIdx != -1 {
			t.popIdx = i
			break
		}
	}
}

// Peek peeks the next task that will be popped.
// It will be nil, if all the tasks are popped or
// a task cannot be popped due to one of the prior subtask is blocking the process.
func (t *Task) Peek() *Task {
	if t.popIdx == -1 {
		// t and it's subtasks has all done
		return nil
	}
	popt := t
	for !popt.isLeaf {
		// There should be no popIdx == -1, if t hasn't done yet.
		popt = popt.Subtasks[popt.popIdx]
	}
	if popt.Blocking() {
		return nil
	}
	return popt
}

// Push pushes the task to it's job, so it can popped again.
// Before pushing a task, change it's status to TaskWaiting,
// or it will be just skipped when popped.
func (t *Task) Push() {
	if !t.isLeaf {
		panic("cannot push a branch task")
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

// CanRetry checks the task reaches maximum retry count of the job.
func (t *Task) CanRetry() bool {
	if t.retry >= t.Job.AutoRetry {
		return false
	}
	return true
}

// Status returns the task's status.
// When it's a branch, it will be calculated from the childen's status.
func (t *Task) Status() TaskStatus {
	if t.isLeaf {
		return t.status
	}
	return t.Stat.Status()
}

// CalcPriority calculates the tasks prority.
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

func (t *Task) Update(u TaskUpdater) error {
	u.Order = t.Job.order
	u.Num = t.num
	err := t.update(u)
	if err != nil {
		return err
	}
	if u.Status != nil {
		t.setStatus(*u.Status)
	}
	if u.Assignee != nil {
		t.Assignee = *(u.Assignee)
	}
	return nil
}

// setStatus sets the task and it's parents status.
// Don't use this directly and use Task.Update unless there is need for
// updating task status without db update.
func (t *Task) setStatus(s TaskStatus) {
	if !t.isLeaf {
		panic("setStatus applied to a leaf task only")
	}
	old := t.status
	t.status = s
	parent := t.parent
	for parent != nil {
		parent.Stat.Change(old, s)
		parent = parent.parent
	}
}
