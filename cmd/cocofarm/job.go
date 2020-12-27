package main

import (
	"container/heap"
	"fmt"
	"sync"
)

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// ID lets a Job distinguishes from others.
	ID int

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
	return h[i].priority > h[j].priority
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

	jobs  *jobHeap
	tasks map[*Job]*taskHeap
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.jobs = &jobHeap{}
	m.tasks = make(map[*Job]*taskHeap)
	return m
}

func (m *jobManager) Add(j *Job) error {
	if j == nil {
		return fmt.Errorf("nil job cannot be added")
	}
	if j.Root == nil {
		return fmt.Errorf("root task of job should not be nil")
	}
	initJob(j)

	m.Lock()
	defer m.Unlock()
	heap.Push(m.jobs, j)

	tasks, ok := m.tasks[j]
	if !ok {
		tasks = &taskHeap{}
		m.tasks[j] = tasks
	}
	walkTaskFn(j.Root, func(t *Task) {
		heap.Push(tasks, t)
	})
	return nil
}

// initJob inits a job before it is added to jobManager.
func initJob(j *Job) {
	j.priority = j.DefaultPriority
	initJobTasks(j.Root, j, nil, 0)
}

// initJobTasks inits a job tasks recursively.
func initJobTasks(t *Task, j *Job, parent *Task, i int) int {
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

func (m *jobManager) Delete(id int) error {
	m.Lock()
	defer m.Unlock()
	idx := -1
	for i := 0; i < len(*m.jobs); i++ {
		j := (*m.jobs)[i]
		if id == j.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("cannot find the job with id")
	}
	heap.Remove(m.jobs, idx)
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

		// check there is any task left.
		if tasks.Len() != 0 {
			peek := (*tasks)[0]
			j.priority = peek.CalcPriority()
			heap.Push(m.jobs, j)
		}

		return t
	}
}
