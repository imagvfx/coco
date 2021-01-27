package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/xid"
)

type JobID int

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// NOTE: Private fields of this struct should be read-only after the initialization.
	// Otherwise, this program will get racy.

	sync.Mutex

	// id is the order number of the job.
	id JobID

	// Job is a Task.
	// Some of the Task's field should be explained in Job's context.
	//
	// Task.Title is human readable title for job.
	//
	// Task.Priority sets the job's default priority.
	// Jobs are compete each other with priority.
	// Job's priority could be temporarily updated by a task that waits at the time.
	// Higher values take precedence to lower values.
	// Negative values will corrected to 0, the lowest priority value.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	*Task

	// CurrentPriority is the job's task priority waiting at the time.
	CurrentPriority int
}

func (j *Job) MarshalJSON() ([]byte, error) {
	m := struct {
		ID              JobID
		Status          string
		Title           string
		Priority        int
		CurrentPriority int
		Subtasks        []*Task
		SerialSubtasks  bool
	}{
		ID:              j.id,
		Status:          j.Status().String(),
		Title:           j.Title,
		Priority:        j.Priority,
		CurrentPriority: j.CurrentPriority,
		Subtasks:        j.Subtasks,
		SerialSubtasks:  j.SerialSubtasks,
	}
	return json.Marshal(m)
}

func (j *Job) WalkTaskFn(fn func(t *Task)) {
	walkFromFn(j.Task, fn)
}

func (j *Job) WalkLeafTaskFn(fn func(t *Task)) {
	leafFn := func(t *Task) {
		if t.IsLeaf() {
			fn(t)
		}
	}
	walkFromFn(j.Task, leafFn)
}

func walkFromFn(t *Task, fn func(t *Task)) {
	down := true
	for {
		if down {
			fn(t)
			if len(t.Subtasks) != 0 {
				t = t.Subtasks[0]
				continue
			}
		}
		if t.next != nil {
			down = true
			t = t.next
			continue
		}
		if t.parent != nil {
			down = false
			t = t.parent
			continue
		}
		// back to root
		return
	}
}

type jobHeap struct {
	heap     []*Job
	priority map[JobID]int
}

func newJobHeap(priority map[JobID]int) *jobHeap {
	return &jobHeap{
		heap:     make([]*Job, 0),
		priority: priority,
	}
}

func (h jobHeap) Len() int {
	return len(h.heap)
}

func (h jobHeap) Less(i, j int) bool {
	if h.priority[h.heap[i].id] > h.priority[h.heap[j].id] {
		return true
	}
	if h.priority[h.heap[i].id] < h.priority[h.heap[j].id] {
		return false
	}
	return h.heap[i].id < h.heap[j].id
}

func (h jobHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

func (h *jobHeap) Push(el interface{}) {
	h.heap = append(h.heap, el.(*Job))
}

func (h *jobHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.heap = old[:n-1]
	return el
}

type jobManager struct {
	sync.Mutex
	nextJobID JobID

	// Job related informations.
	// When a job is deleted, those related info should be deleted all toghther,
	// except `jobs` heap. It is expensive to search an item from heap.
	// So, deleted job in `jobs` will be deleted when it is popped from PopTask.
	job         map[JobID]*Job
	jobPriority map[JobID]int
	jobBlocked  map[JobID]bool
	task        map[TaskID]*Task
	jobs        *jobHeap

	assignee     map[TaskID]*Worker
	CancelTaskCh chan *Task
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.job = make(map[JobID]*Job)
	m.jobBlocked = make(map[JobID]bool)
	m.jobPriority = make(map[JobID]int)
	m.jobs = newJobHeap(m.jobPriority)
	m.task = make(map[TaskID]*Task)
	m.assignee = make(map[TaskID]*Worker)
	m.CancelTaskCh = make(chan *Task)
	return m
}

func (m *jobManager) Get(id JobID) *Job {
	m.Lock()
	defer m.Unlock()
	return m.job[id]
}

func (m *jobManager) GetTask(id TaskID) *Task {
	m.Lock()
	defer m.Unlock()
	return m.task[id]
}

func (m *jobManager) Add(j *Job) (JobID, error) {
	if j == nil {
		return -1, fmt.Errorf("nil job cannot be added")
	}
	if len(j.Subtasks) == 0 {
		return -1, fmt.Errorf("a job should have at least one subtask")
	}
	initJob(j)

	j.id = m.nextJobID
	m.nextJobID++

	m.Lock()
	defer m.Unlock()
	m.job[j.id] = j

	// didn't hold lock of the job as the job will not get published
	// until Add method returns.

	heap.Push(m.jobs, j)

	j.WalkTaskFn(func(t *Task) {
		m.task[t.id] = t
	})

	// set priority for the very first leaf task.
	peek := j.Peek()
	// peek can be nil, when the job doesn't have any leaf task.
	if peek != nil {
		m.jobPriority[j.id] = peek.CalcPriority()
	}
	return j.id, nil
}

// initJob inits a job's tasks.
// initJob returns unmodified pointer of the job, for in case
// when user wants to directly assign to a variable. (see test code)
func initJob(j *Job) *Job {
	initJobTasks(j.Task, j, nil, 0, 0)
	return j
}

// initJobTasks inits a job's tasks recursively before it is added to jobManager.
// No need to hold the lock.
func initJobTasks(t *Task, j *Job, parent *Task, nth, i int) int {
	t.id = TaskID(xid.New().String())
	t.job = j
	t.parent = parent
	t.nthChild = nth
	if t.IsLeaf() {
		t.num = i
		i++
	}
	if t.Priority < 0 {
		// nagative priority is invalid.
		t.Priority = 0
	}
	t.Stat = &branchStat{}
	iOld := i
	var prev *Task
	for nth, subt := range t.Subtasks {
		if prev != nil {
			prev.next = subt
		}
		i = initJobTasks(subt, j, t, nth, i)
		prev = subt
	}
	t.Stat.nWaiting = i - iOld
	return i
}

// Cancel cancels a job.
// Both running and waiting tasks of the job will be marked as failed,
// and commands executing from running tasks will be canceled right away.
func (m *jobManager) Cancel(id JobID) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	if j.Status() == TaskDone {
		// TODO: the job's status doesn't get changed to done yet.
		return fmt.Errorf("job has already Done: %v", id)
	}
	j.WalkLeafTaskFn(func(t *Task) {
		if t.status == TaskRunning {
			go func() {
				// Let worker knows that the task is canceled.
				m.CancelTaskCh <- t
			}()
		}
		if t.status != TaskDone {
			t.SetStatus(TaskFailed)
		}
	})
	return nil
}

func (m *jobManager) Retry(id JobID) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	j.WalkLeafTaskFn(func(t *Task) {
		if t.Status() == TaskFailed {
			t.SetStatus(TaskWaiting)
		}
	})
	heap.Push(m.jobs, j)
	return nil
}

// Delete deletes a job irrecoverably.
func (m *jobManager) Delete(id JobID) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	delete(m.job, id)
	delete(m.jobPriority, id)
	delete(m.jobBlocked, id)
	j.WalkTaskFn(func(t *Task) {
		delete(m.task, t.id)
	})
	return nil
}

func (m *jobManager) PopTask() *Task {
	m.Lock()
	defer m.Unlock()
	for {
		if m.jobs.Len() == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*Job)
		_, ok := m.job[j.id]
		if !ok {
			// the job deleted
			continue
		}
		if j.Status() == TaskFailed {
			// one or more tasks of the job failed,
			// block the job until user retry the tasks.
			continue
		}

		j.Lock()
		defer j.Unlock()
		t, done := j.Pop()
		// check there is any leaf task left.
		if !done {
			peek := j.Peek()
			// peek can be nil when next Job.Pop has blocked for some reason.
			if peek != nil {
				// the peeked task is also cared by this lock.
				p := peek.CalcPriority()
				m.jobPriority[j.id] = p
				// also keep the info in the Job.
				j.CurrentPriority = p
				heap.Push(m.jobs, j)
			} else {
				m.jobBlocked[j.id] = true
			}
		}

		return t
	}
}

func (m *jobManager) Assign(id TaskID, w *Worker) error {
	m.Lock()
	defer m.Unlock()
	return m.assign(id, w)
}

func (m *jobManager) assign(id TaskID, w *Worker) error {
	a, ok := m.assignee[id]
	if ok {
		return fmt.Errorf("task is assigned to a different worker: %v - %v", id, a.addr)
	}
	m.assignee[id] = w
	return nil
}

func (m *jobManager) Unassign(id TaskID, w *Worker) error {
	m.Lock()
	defer m.Unlock()
	return m.unassign(id, w)
}

func (m *jobManager) unassign(id TaskID, w *Worker) error {
	a, ok := m.assignee[id]
	if !ok {
		return fmt.Errorf("task isn't assigned to any worker: %v", id)
	}
	if w != a {
		return fmt.Errorf("task is assigned to a different worker: %v", id)
	}
	delete(m.assignee, id)
	return nil
}
