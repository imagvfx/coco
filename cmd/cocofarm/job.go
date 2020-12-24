package main

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/imagvfx/coco"
)

type taskWalker struct {
	ch   chan *coco.Task
	next *coco.Task
}

func newTaskWalker(t *coco.Task) *taskWalker {
	w := &taskWalker{}
	w.ch = make(chan *coco.Task)
	go walk(t, w.ch)
	w.next = <-w.ch
	return w
}

func (w *taskWalker) Next() *coco.Task {
	next := w.next
	w.next = <-w.ch
	return next
}

func (w *taskWalker) Peek() *coco.Task {
	return w.next
}

func walk(t *coco.Task, ch chan *coco.Task) {
	walkR(t, ch)
	close(ch)
}

func walkR(t *coco.Task, ch chan<- *coco.Task) {
	for _, t := range t.Subtasks {
		walkR(t, ch)
	}
	ch <- t
}

type jobHeap []*coco.Job

func (h jobHeap) Len() int {
	return len(h)
}

func (h jobHeap) Less(i, j int) bool {
	return h[i].Priority > h[j].Priority
}

func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *jobHeap) Push(el interface{}) {
	*h = append(*h, el.(*coco.Job))
}

func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[:n-1]
	return el
}

type jobManager struct {
	sync.Mutex

	jobs        *jobHeap
	taskWalkers map[*coco.Job]*taskWalker
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.jobs = &jobHeap{}
	m.taskWalkers = make(map[*coco.Job]*taskWalker)
	return m
}

func (m *jobManager) Add(j *coco.Job) error {
	if j == nil {
		return fmt.Errorf("nil job cannot be added")
	}
	if j.Root == nil {
		return fmt.Errorf("root task of job should not be nil")
	}
	m.Lock()
	defer m.Unlock()
	heap.Push(m.jobs, j)
	return nil
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

func (m *jobManager) Jobs() *jobHeap {
	m.Lock()
	defer m.Unlock()
	return m.jobs
}

func (m *jobManager) NextTask() *coco.Task {
	for {
		if len(*m.jobs) == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*coco.Job)
		m.Lock()
		walk, ok := m.taskWalkers[j]
		if !ok {
			walk = newTaskWalker(j.Root)
			m.taskWalkers[j] = walk
		}
		m.Unlock()
		next := walk.Next()

		// check there is any task left.
		peek := walk.Peek()
		if peek != nil {
			// TODO: calculate real priority of the task.
			j.Priority = peek.Priority
			heap.Push(m.jobs, j)
		}

		return next
	}
}
