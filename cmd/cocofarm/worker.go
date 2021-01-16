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
	WorkerIdle
	WorkerRunning
)

type Worker struct {
	sync.Mutex

	addr   string
	status WorkerStatus

	// task directs a task the worker is currently working.
	// The worker is in idle when it is empty string.
	task string
}

func (w *Worker) Addr() string {
	w.Lock()
	defer w.Unlock()
	return w.addr
}

func (w *Worker) SetStatus(s WorkerStatus) {
	w.Lock()
	defer w.Unlock()
	w.status = s
}

type workerManager struct {
	sync.Mutex
	workers  []*Worker
	ReadyCh  chan struct{}
	assignee map[string]*Worker
}

func newWorkerManager() *workerManager {
	m := &workerManager{}
	m.workers = make([]*Worker, 0)
	m.ReadyCh = make(chan struct{})
	m.assignee = make(map[string]*Worker)
	return m
}

func (m *workerManager) Add(w *Worker) error {
	m.Lock()
	defer m.Unlock()
	found := false
	for _, v := range m.workers {
		if w.addr == v.addr {
			found = true
		}
	}
	if found {
		return fmt.Errorf("worker %s already added", w.addr)
	}
	m.workers = append(m.workers, w)
	return nil
}

func (m *workerManager) Bye(workerAddr string) error {
	m.Lock()
	defer m.Unlock()
	idx := -1
	for i, v := range m.workers {
		if workerAddr == v.addr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("worker not found: %v", workerAddr)
	}
	m.workers = append(m.workers[:idx], m.workers[idx+1:]...)
	return nil
}

func (m *workerManager) Assign(taskID string, w *Worker) error {
	m.Lock()
	defer m.Unlock()
	a, ok := m.assignee[taskID]
	if ok {
		return fmt.Errorf("task is assigned to a different worker: %v - %v", taskID, a.addr)
	}
	w.task = taskID
	m.assignee[taskID] = w
	return nil
}

func (m *workerManager) Unassign(taskID string, w *Worker) error {
	// TODO: do we need DoneBy and also Waiting? merge two?
	// this function is task centric, Waiting is worker centric.
	// also Waiting is a blocking function.
	m.Lock()
	defer m.Unlock()
	a, ok := m.assignee[taskID]
	if !ok {
		return fmt.Errorf("task isn't assigned to any worker: %v", taskID)
	}
	if w != a {
		return fmt.Errorf("task is assigned to a different worker: %v", taskID)
	}
	w.task = ""
	delete(m.assignee, taskID)
	return nil
}

func (m *workerManager) Waiting(w *Worker) {
	w.SetStatus(WorkerIdle)
	go func() { m.ReadyCh <- struct{}{} }()
}

func (m *workerManager) FindByAddr(addr string) *Worker {
	m.Lock()
	defer m.Unlock()
	for _, w := range m.workers {
		if addr == w.Addr() {
			return w
		}
	}
	return nil
}

func (m *workerManager) IdleWorkers() []*Worker {
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

func (m *workerManager) sendTask(w *Worker, t *Task) error {
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
	_, err = c.Run(ctx, req)
	if err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()
	m.assignee[t.id] = w
	w.status = WorkerRunning
	return nil
}

func (m *workerManager) sendCancelTask(w *Worker, t *Task) error {
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

	_, err = c.Cancel(ctx, req)
	if err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()
	delete(m.assignee, t.id)
	return nil
}
