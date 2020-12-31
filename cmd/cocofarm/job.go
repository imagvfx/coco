package main

import (
	"container/heap"
	"fmt"
	"strconv"
	"sync"

	"github.com/rs/xid"
)

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// ID lets a Job distinguishes from others.
	ID string

	// Title is human readable title for job.
	Title string

	// DefaultPriority sets the job's default priority.
	// Jobs are compete each other with priority.
	// Job's priority could be temporarily updated by a task that waits at the time.
	// Higher values take precedence to lower values.
	// Negative values will corrected to 0, the lowest priority value.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	DefaultPriority int

	// priority is the job's task priority waiting at the time.
	priority int

	// Root task contains commands or subtasks to be run.
	Root *Task
}

type jobHeap []*Job

func (h jobHeap) Len() int {
	return len(h)
}

func (h jobHeap) Less(i, j int) bool {
	if h[i].priority > h[j].priority {
		return true
	}
	if h[i].priority < h[j].priority {
		return false
	}
	return h[i].ID < h[j].ID
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
	nextJobID    int
	job          map[string]*Job
	jobs         *jobHeap
	tasks        map[*Job]*taskHeap
	CancelTaskCh chan *Task
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.job = make(map[string]*Job)
	m.jobs = &jobHeap{}
	m.tasks = make(map[*Job]*taskHeap)
	m.CancelTaskCh = make(chan *Task)
	return m
}

func (m *jobManager) Add(j *Job) (string, error) {
	if j == nil {
		return "", fmt.Errorf("nil job cannot be added")
	}
	if j.Root == nil {
		return "", fmt.Errorf("root task of job should not be nil")
	}
	initJob(j)

	j.ID = strconv.Itoa(m.nextJobID)
	m.nextJobID++

	m.Lock()
	defer m.Unlock()
	m.job[j.ID] = j
	heap.Push(m.jobs, j)

	tasks, ok := m.tasks[j]
	if !ok {
		tasks = &taskHeap{}
		m.tasks[j] = tasks
	}
	walkLeafTaskFn(j.Root, func(t *Task) {
		heap.Push(tasks, t)
	})

	// set priority for the very first leaf task.
	peek := (*tasks)[0]
	j.priority = peek.CalcPriority()
	return j.ID, nil
}

// initJob inits a job before it is added to jobManager.
func initJob(j *Job) {
	initJobTasks(j.Root, j, nil, 0)
}

// initJobTasks inits a job tasks recursively.
func initJobTasks(t *Task, j *Job, parent *Task, i int) int {
	t.id = xid.New().String()
	t.job = j
	t.parent = parent
	t.num = i
	i++
	if t.Priority < 0 {
		// nagative priority is invalid.
		t.Priority = 0
	}
	for _, subt := range t.Subtasks {
		i = initJobTasks(subt, j, t, i)
	}
	return i
}

func (m *jobManager) Cancel(id string) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	// also remove the job from heap.
	idx := -1
	for i, j := range *m.jobs {
		if id == j.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("job already done: %v", id)
	}
	heap.Remove(m.jobs, idx)
	go func() {
		walkLeafTaskFn(j.Root, func(t *Task) {
			// indicate the task is cancelled, first.
			t.status = TaskCancelled
		})
		walkLeafTaskFn(j.Root, func(t *Task) {
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
	idx := -1
	for i, j := range *m.jobs {
		if id == j.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		heap.Remove(m.jobs, idx)
	}
	delete(m.job, id)
	return nil
}

func (m *jobManager) NextTask() *Task {
	m.Lock()
	defer m.Unlock()
	for {
		if len(*m.jobs) == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*Job)
		tasks := m.tasks[j]
		t := heap.Pop(tasks).(*Task)

		// check there is any leaf task left.
		if tasks.Len() != 0 {
			peek := (*tasks)[0]
			j.priority = peek.CalcPriority()
			heap.Push(m.jobs, j)
		}

		return t
	}
}
