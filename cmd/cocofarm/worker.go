package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type WorkerStatus int

const (
	WorkerNotFound = WorkerStatus(iota)
	WorkerReady
	WorkerRunning
)

// Worker is a worker who continuously takes a task and run it's commands.
type Worker struct {
	// addr is the worker's listen addr.
	addr string

	// status indicates the worker's current status.
	status WorkerStatus

	// task directs a task the worker is currently working.
	// The worker is in idle when it is empty string.
	task string
}

type uniqueWorkerQueue struct {
	has   map[*Worker]bool
	first *workerQueueItem
	last  *workerQueueItem
}

type workerQueueItem struct {
	w    *Worker
	next *workerQueueItem
}

func newUniqueWorkerQueue() *uniqueWorkerQueue {
	return &uniqueWorkerQueue{
		has: make(map[*Worker]bool),
	}
}

func (q *uniqueWorkerQueue) Push(w *Worker) {
	if q.has[w] {
		return
	}
	q.has[w] = true
	item := &workerQueueItem{w: w}
	if q.first == nil {
		q.first = item
	} else {
		q.last.next = item
	}
	q.last = item
}

func (q *uniqueWorkerQueue) Pop() *Worker {
	if q.first == nil {
		return nil
	}
	w := q.first.w
	delete(q.has, w)
	if q.first == q.last {
		q.first = nil
		q.last = nil
		return w
	}
	q.first = q.first.next
	return w
}

func (q *uniqueWorkerQueue) Remove(w *Worker) bool {
	if !q.has[w] {
		return false
	}
	delete(q.has, w)
	var prev *workerQueueItem
	for it := q.first; it != q.last; it = it.next {
		if it.w == w {
			prev.next = it.next
			break
		}
		prev = it
	}
	return true
}

type workerManager struct {
	sync.Mutex
	worker  map[string]*Worker
	workers *uniqueWorkerQueue
	// ReadyCh tries fast matching of a worker and a task.
	ReadyCh  chan struct{}
	assignee map[string]*Worker
}

func newWorkerManager() *workerManager {
	m := &workerManager{}
	m.worker = make(map[string]*Worker)
	m.workers = newUniqueWorkerQueue()
	m.ReadyCh = make(chan struct{})
	m.assignee = make(map[string]*Worker)
	return m
}

func (m *workerManager) Add(w *Worker) error {
	m.Lock()
	defer m.Unlock()
	_, ok := m.worker[w.addr]
	if ok {
		return fmt.Errorf("worker already exists: %v", w.addr)
	}
	m.worker[w.addr] = w
	return nil
}

func (m *workerManager) Bye(workerAddr string) error {
	m.Lock()
	defer m.Unlock()
	w, ok := m.worker[workerAddr]
	if !ok {
		return fmt.Errorf("worker not found: %v", workerAddr)
	}
	delete(m.worker, workerAddr)
	m.workers.Remove(w)
	return nil
}

func (m *workerManager) FindByAddr(addr string) *Worker {
	m.Lock()
	defer m.Unlock()
	return m.worker[addr]
}

func (m *workerManager) Assign(taskID string, w *Worker) error {
	m.Lock()
	defer m.Unlock()
	a, ok := m.assignee[taskID]
	if ok {
		return fmt.Errorf("task is assigned to a different worker: %v - %v", taskID, a.addr)
	}
	m.assign(taskID, w)
	return nil
}

func (m *workerManager) assign(taskID string, w *Worker) {
	w.status = WorkerRunning
	w.task = taskID
	m.assignee[taskID] = w
}

func (m *workerManager) Unassign(taskID string, w *Worker) error {
	m.Lock()
	defer m.Unlock()
	a, ok := m.assignee[taskID]
	if !ok {
		return fmt.Errorf("task isn't assigned to any worker: %v", taskID)
	}
	if w != a {
		return fmt.Errorf("task is assigned to a different worker: %v", taskID)
	}
	m.unassign(taskID, w)
	return nil
}

func (m *workerManager) unassign(taskID string, w *Worker) {
	w.task = ""
	delete(m.assignee, taskID)
	// Don't make the worker idle/ready, as it only can be set by the worker.
	// see comment on Waiting.
}

// Ready reports that a worker is ready for a new task.
// NOTE: It should be only called by the worker through workerFarm.
func (m *workerManager) Ready(w *Worker) {
	m.Lock()
	defer m.Unlock()
	w.status = WorkerReady
	m.workers.Push(w)
	go func() { m.ReadyCh <- struct{}{} }()
}

func (m *workerManager) Pop() *Worker {
	m.Lock()
	defer m.Unlock()
	return m.workers.Pop()
}

func (m *workerManager) Push(w *Worker) {
	m.Lock()
	defer m.Unlock()
	m.workers.Push(w)
}

func (m *workerManager) sendTask(w *Worker, t *Task) (err error) {
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.RunRequest{}
	req.Id = t.id
	for _, c := range t.Commands {
		reqCmd := &pb.Command{
			Args: c,
		}
		req.Cmds = append(req.Cmds, reqCmd)
	}

	// Lock before we send run message, in case the running is done by the worker
	// ever before the server assigning the worker, which makes the server messy.
	m.Lock()
	defer m.Unlock()
	_, err = c.Run(ctx, req)
	if err != nil {
		return err
	}
	m.assign(t.id, w)
	return nil
}

func (m *workerManager) sendCancelTask(w *Worker, t *Task) (err error) {
	log.Printf("cancel: %v %v", w.addr, t.id)
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CancelRequest{}
	req.Id = t.id

	// Lock before we send cancel message, in case the canceling is done by the worker
	// even before the server assigning the worker, which makes the server messy.
	m.Lock()
	defer m.Unlock()
	_, err = c.Cancel(ctx, req)
	if err != nil {
		return err
	}
	m.unassign(t.id, w)
	return nil
}
