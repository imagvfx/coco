package main

import (
	"context"
	"fmt"
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
	addr   string
	status WorkerStatus
}

type workerManager struct {
	sync.Mutex
	workers []*Worker
}

func newWorkerManager() *workerManager {
	m := &workerManager{}
	m.workers = make([]*Worker, 0)
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

func (m *workerManager) SetStatus(w *Worker, s WorkerStatus) {
	m.Lock()
	defer m.Unlock()
	w.status = s
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

func (m *workerManager) sendCommands(w *Worker, cmds []Command) error {
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pbCmds := &pb.Commands{}
	for _, c := range cmds {
		pbCmd := &pb.Command{
			Args: c,
		}
		pbCmds.Cmds = append(pbCmds.Cmds, pbCmd)
	}
	_, err = c.Run(ctx, pbCmds)
	if err != nil {
		return err
	}

	m.Lock()
	defer m.Unlock()
	w.status = WorkerRunning
	return nil
}
