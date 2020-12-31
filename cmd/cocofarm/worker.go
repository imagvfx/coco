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
}

func (w *Worker) SetStatus(s WorkerStatus) {
	w.Lock()
	defer w.Unlock()
	w.status = s
}

type workerManager struct {
	sync.Mutex
	workers  []*Worker
	WorkerCh chan *Worker
	assignee map[string]*Worker
}

func newWorkerManager() *workerManager {
	m := &workerManager{}
	m.workers = make([]*Worker, 0)
	m.WorkerCh = make(chan *Worker)
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

func (m *workerManager) Waiting(w *Worker) {
	w.SetStatus(WorkerIdle)
	m.WorkerCh <- w
}

func (m *workerManager) idleWorkers() []*Worker {
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

	pbTask := &pb.Task{}
	pbTask.Id = t.id
	for _, c := range t.Commands {
		pbCmd := &pb.Command{
			Args: c,
		}
		pbTask.Cmds = append(pbTask.Cmds, pbCmd)
	}
	_, err = c.Run(ctx, pbTask)
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

	// only id is needed for cancelling
	pbTask := &pb.Task{}
	pbTask.Id = t.id

	_, err = c.Cancel(ctx, pbTask)
	if err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()
	delete(m.assignee, t.id)
	return nil
}
