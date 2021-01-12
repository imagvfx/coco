package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/rs/xid"
)

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// NOTE: Private fields of this struct should be read-only after the initialization.
	// Otherwise, this program will get racy.

	sync.Mutex

	// order is the number of the order
	order string

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
		Order           string
		Status          string
		Title           string
		Priority        int
		CurrentPriority int
		Subtasks        []*Task
		SerialSubtasks  bool
	}{
		Order:           j.order,
		Status:          j.Status().String(j.IsLeaf()),
		Title:           j.Title,
		Priority:        j.Priority,
		CurrentPriority: j.CurrentPriority,
		Subtasks:        j.Subtasks,
		SerialSubtasks:  j.SerialSubtasks,
	}
	return json.Marshal(m)
}

func (j *Job) WalkTaskFn(fn func(t *Task)) {
	for _, subt := range j.Subtasks {
		walkTaskFn(subt, fn)
	}
}

func walkTaskFn(t *Task, fn func(t *Task)) {
	fn(t)
	for _, subt := range t.Subtasks {
		walkTaskFn(subt, fn)
	}
}

func (j *Job) WalkLeafTaskFn(fn func(t *Task)) {
	for _, subt := range j.Subtasks {
		walkLeafTaskFn(subt, fn)
	}
}

func walkLeafTaskFn(t *Task, fn func(t *Task)) {
	if t.IsLeaf() {
		fn(t)
	}
	for _, subt := range t.Subtasks {
		walkLeafTaskFn(subt, fn)
	}
}

type jobHeap []*Job

func (h jobHeap) Len() int {
	return len(h)
}

func (h jobHeap) Less(i, j int) bool {
	h[i].Lock()
	h[j].Lock()
	defer h[i].Unlock()
	defer h[j].Unlock()
	if h[i].CurrentPriority > h[j].CurrentPriority {
		return true
	}
	if h[i].CurrentPriority < h[j].CurrentPriority {
		return false
	}
	return h[i].id < h[j].id
}

func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *jobHeap) Push(el interface{}) {
	*h = append(*h, el.(*Job))
}

func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[:n-1]
	return el
}

type taskHeap []*Task

func (h taskHeap) Len() int {
	return len(h)
}

func (h taskHeap) Less(i, j int) bool {
	h[i].job.Lock() // h[i].job == h[j].job
	defer h[i].job.Unlock()
	return h[i].num < h[j].num
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *taskHeap) Push(el interface{}) {
	*h = append(*h, el.(*Task))
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[:n-1]
	return el
}

type jobManager struct {
	sync.Mutex
	nextJobID int
	job       map[string]*Job
	// jobs may have cancelled jobs.
	// PopTask should handle this properly.
	jobs         *jobHeap
	task         map[string]*Task
	tasks        map[string]*taskHeap
	CancelTaskCh chan *Task
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.job = make(map[string]*Job)
	m.jobs = &jobHeap{}
	m.task = make(map[string]*Task)
	m.tasks = make(map[string]*taskHeap)
	m.CancelTaskCh = make(chan *Task)
	return m
}

func (m *jobManager) Get(id string) *Job {
	m.Lock()
	defer m.Unlock()
	return m.job[id]
}

func (m *jobManager) GetTask(id string) *Task {
	m.Lock()
	defer m.Unlock()
	return m.task[id]
}

func (m *jobManager) Add(j *Job) (string, error) {
	if j == nil {
		return "", fmt.Errorf("nil job cannot be added")
	}
	if len(j.Subtasks) == 0 {
		return "", fmt.Errorf("a job should have at least one subtask")
	}
	initJobTasks(j.Task, j, nil, 0)

	j.order = strconv.Itoa(m.nextJobID)
	m.nextJobID++

	m.Lock()
	defer m.Unlock()
	m.job[j.order] = j

	// didn't hold lock of the job as the job will not get published
	// until Add method returns.
	// NOTE: but anyway, the heap operations following will hold lock on jobs.

	heap.Push(m.jobs, j)

	tasks, ok := m.tasks[j.order]
	if !ok {
		tasks = &taskHeap{}
		m.tasks[j.order] = tasks
	}
	j.WalkLeafTaskFn(func(t *Task) {
		heap.Push(tasks, t)
	})
	j.WalkTaskFn(func(t *Task) {
		m.task[t.id] = t
	})

	nLeafs := 0
	j.WalkLeafTaskFn(func(t *Task) {
		nLeafs++
	})
	j.Stat.nWaiting = nLeafs

	// set priority for the very first leaf task.
	peek := (*tasks)[0]
	j.CurrentPriority = peek.CalcPriority()
	return j.order, nil
}

// initJob inits a job's tasks.
func initJob(j *Job) {
	initJobTasks(j.Task, j, nil, 0)
}

// initJobTasks inits a job's tasks recursively before it is added to jobManager.
// No need to hold the lock.
func initJobTasks(t *Task, j *Job, parent *Task, i int) int {
	t.id = xid.New().String()
	t.job = j
	t.parent = parent
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
	for _, subt := range t.Subtasks {
		i = initJobTasks(subt, j, t, i)
	}
	t.Stat.nWaiting = i - iOld
	return i
}

func (m *jobManager) Cancel(id string) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	if j.Status() == TaskCancelled {
		return fmt.Errorf("job has already cancelled: %v", id)
	}
	if j.Status() == TaskDone {
		// TODO: the job's status doesn't get changed to done yet.
		return fmt.Errorf("job has already Done: %v", id)
	}
	// indicate the job and it's tasks are cancelled, first.
	j.WalkLeafTaskFn(func(t *Task) {
		t.SetStatus(TaskCancelled)
	})
	// Delete the job from m.jobs (heap) will be expensive.
	// Let PopTask do the job.
	delete(m.tasks, id)
	go func() {
		j.WalkLeafTaskFn(func(t *Task) {
			m.CancelTaskCh <- t
		})
	}()
	return nil
}

func (m *jobManager) Delete(id string) error {
	m.Lock()
	defer m.Unlock()
	_, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	delete(m.job, id)
	// Delete the job from m.jobs (heap) will be expensive.
	// Let PopTask do the job.
	delete(m.tasks, id)
	return nil
}

func (m *jobManager) NextTask() *Task {
	m.Lock()
	defer m.Unlock()
	if m.jobs.Len() == 0 {
		return nil
	}
	j := (*m.jobs)[0]
	return (*m.tasks[j.order])[0]
}

func (m *jobManager) PopTask() *Task {
	m.Lock()
	defer m.Unlock()
	for {
		if m.jobs.Len() == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*Job)
		_, ok := m.job[j.order]
		if !ok {
			// the job deleted
			continue
		}
		if j.Status() == TaskCancelled {
			// the job cancelled
			continue
		}

		tasks := m.tasks[j.order]
		t := heap.Pop(tasks).(*Task)

		// check there is any leaf task left.
		if tasks.Len() != 0 {
			peek := (*tasks)[0]
			j.Lock()
			// the peeked task is also cared by this lock.
			j.CurrentPriority = peek.CalcPriority()
			j.Unlock()
			heap.Push(m.jobs, j)
		}

		return t
	}
}
