package main

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	WorkerCooling
)

// Worker is a worker who continuously takes a task and run it's commands.
type Worker struct {
	sync.Mutex

	// addr is the worker's listen addr.
	addr string

	// status indicates the worker's current status.
	status WorkerStatus

	// task directs a task the worker is currently working.
	// The worker is in idle when it is empty string.
	task TaskID

	group *WorkerGroup
}

type WorkerGroup struct {
	Name         string
	Matchers     []IPMatcher
	ServeTargets []string
}

func (g WorkerGroup) Match(addr string) bool {
	for _, m := range g.Matchers {
		if m.Match(addr) {
			return true
		}
	}
	return false
}

type workerManager struct {
	sync.Mutex
	worker       map[string]*Worker
	workers      *uniqueQueue
	workerGroups []*WorkerGroup
	nForTag      map[string]int
	// ReadyCh tries fast matching of a worker and a task.
	ReadyCh chan struct{}
}

func newWorkerManager(wgrps []*WorkerGroup) *workerManager {
	m := &workerManager{}
	m.worker = make(map[string]*Worker)
	m.workers = newUniqueQueue()
	m.workerGroups = wgrps
	m.nForTag = make(map[string]int)
	m.ReadyCh = make(chan struct{})
	return m
}

func (m *workerManager) ServableTags() []string {
	servable := make([]string, 0)
	for t, n := range m.nForTag {
		if n != 0 {
			servable = append(servable, t)
		}
	}
	return servable
}

func (m *workerManager) Add(w *Worker) error {
	m.Lock()
	defer m.Unlock()
	_, ok := m.worker[w.addr]
	if ok {
		return fmt.Errorf("worker already exists: %v", w.addr)
	}
	addr := strings.Split(w.addr, ":")[0] // remove port
	find := false
	for _, g := range m.workerGroups {
		if g.Match(addr) {
			find = true
			w.group = g
			break
		}
	}
	if !find {
		return fmt.Errorf("worker address not defined any groups")
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
	for _, t := range w.group.ServeTargets {
		m.nForTag[t] -= 1
		if m.nForTag[t] < 0 {
			panic("shouldn't happen")
		}
	}
	return nil
}

func (m *workerManager) FindByAddr(addr string) *Worker {
	m.Lock()
	defer m.Unlock()
	return m.worker[addr]
}

// Ready reports that a worker is ready for a new task.
// NOTE: It should be only called by the worker through workerFarm.
func (m *workerManager) Ready(w *Worker) {
	m.Lock()
	defer m.Unlock()
	w.status = WorkerReady
	w.task = ""
	m.workers.Push(w)
	for _, t := range w.group.ServeTargets {
		m.nForTag[t] += 1
	}
	go func() { m.ReadyCh <- struct{}{} }()
}

func (m *workerManager) Pop(target string) *Worker {
	m.Lock()
	defer m.Unlock()
	var w *Worker
	for {
		v := m.workers.Pop()
		if v == nil {
			return nil
		}
		w = v.(*Worker)
		servable := false
		for _, t := range w.group.ServeTargets {
			if t == "*" || t == target {
				servable = true
				break
			}
		}
		if servable {
			break
		}
		defer func() { m.workers.Push(w) }()
	}
	for _, t := range w.group.ServeTargets {
		m.nForTag[t] -= 1
		if m.nForTag[t] < 0 {
			panic("shouldn't happen")
		}
	}
	return w
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
	req.Id = string(t.id)
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
	w.status = WorkerRunning
	w.task = t.id
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
	req.Id = string(t.id)

	// Lock before we send cancel message, in case the canceling is done by the worker
	// even before the server assigning the worker, which makes the server messy.
	m.Lock()
	defer m.Unlock()
	_, err = c.Cancel(ctx, req)
	if err != nil {
		return err
	}
	w.status = WorkerCooling
	w.task = ""
	return nil
}
