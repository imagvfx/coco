package main

import (
	"fmt"
	"log"
	"sync"
	"time"

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
	*h = old[:n-1]
	return el
}

type jobManager struct {
	sync.Mutex

	jobs        map[int]*coco.Job
	taskWalkers map[*coco.Job]*taskWalker

	workers []*Worker
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.jobs = make(map[int]*coco.Job)
	m.taskWalkers = make(map[*coco.Job]*taskWalker)
	// TODO: accept workers
	m.workers = []*Worker{&Worker{addr: "localhost:8283", status: WorkerIdle}}
	return m
}

func (m *jobManager) Add(j *coco.Job) error {
	fmt.Println(j)
	if j == nil {
		return fmt.Errorf("nil job cannot be added")
	}
	if j.Root == nil {
		return fmt.Errorf("root task of job should not be nil")
	}
	m.Lock()
	defer m.Unlock()
	m.jobs[j.ID] = j
	return nil
}

func (m *jobManager) Delete(id int) {
	m.Lock()
	defer m.Unlock()
	delete(m.jobs, id)
}

func (m *jobManager) Jobs() map[int]*coco.Job {
	m.Lock()
	defer m.Unlock()
	return m.jobs
}

func (m *jobManager) idleWorkers() []*Worker {
	m.Lock()
	defer m.Unlock()
	workers := make([]*Worker, 0)
	for _, w := range m.workers {
		if w.status == WorkerIdle {
			workers = append(workers, w)
		}
	}
	return workers
}

func (m *jobManager) Start() {
	match := func() {
		workers := m.idleWorkers()
		if len(workers) == 0 {
			log.Print("no worker yet")
			return
		}

		h := &jobHeap{}
		jobs := m.Jobs()
		for _, j := range jobs {
			h.Push(j)
		}

		for i := 0; i < len(workers) && i < len(m.jobs); i++ {
			fmt.Println(i)
			w := workers[i]
			j := h.Pop().(*coco.Job)
			walk, ok := m.taskWalkers[j]
			if !ok {
				walk = newTaskWalker(j.Root)
				m.taskWalkers[j] = walk
			}
			var cmds []coco.Command
			for {
				t := walk.Next()
				if t == nil {
					break
				}
				if len(t.Commands) != 0 {
					cmds = t.Commands
					break
				}
			}
			if walk.Peek() == nil {
				m.Delete(j.ID)
			}
			err := sendCommands(w.addr, cmds)
			if err != nil {
				log.Print(err)
			}
		}
	}

	go func() {
		for {
			time.Sleep(time.Second)
			log.Print("matching...")
			match()
		}
	}()
}
