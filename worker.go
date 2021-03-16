package coco

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
	task string

	group *WorkerGroup

	update func(WorkerUpdater) error
}

func NewWorker(addr string) *Worker {
	return &Worker{addr: addr, status: WorkerReady}
}

// ToSQL converts a Worker into a SQLWorker.
func (w *Worker) ToSQL() *SQLWorker {
	sw := &SQLWorker{
		Addr:   w.addr,
		Status: w.status,
		Task:   w.task,
	}
	return sw
}

// FromSQL converts a SQLWorker into a Worker.
func (w *Worker) FromSQL(sw *SQLWorker) {
	w.addr = sw.Addr
	w.status = sw.Status
	w.task = sw.Task
}

// Update updates a worker in both the program and the db.
func (w *Worker) Update(u WorkerUpdater) error {
	u.Addr = w.addr
	err := w.update(u)
	if err != nil {
		return err
	}
	if u.Status != nil {
		w.status = *u.Status
	}
	if u.Task != nil {
		w.task = *u.Task
	}
	return nil
}

type WorkerGroup struct {
	Name         string
	Matchers     []AddressMatcher
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

type WorkerManager struct {
	sync.Mutex
	WorkerService WorkerService
	worker        map[string]*Worker
	workers       *uniqueQueue
	workerGroups  []*WorkerGroup
	nForTag       map[string]int
	// ReadyCh tries fast matching of a worker and a task.
	ReadyCh chan struct{}
}

func NewWorkerManager(ws WorkerService, wgrps []*WorkerGroup) *WorkerManager {
	m := &WorkerManager{}
	m.WorkerService = ws
	m.worker = make(map[string]*Worker)
	m.workers = newUniqueQueue()
	m.workerGroups = wgrps
	m.nForTag = make(map[string]int)
	m.ReadyCh = make(chan struct{})
	return m
}

// RestoreWorkerManager restores a WorkerManager from a db.
func RestoreWorkerManager(ws WorkerService, wgrps []*WorkerGroup) (*WorkerManager, error) {
	m := NewWorkerManager(ws, wgrps)
	sqlWorkers, err := m.WorkerService.FindWorkers(WorkerFilter{})
	if err != nil {
		return nil, err
	}
	for _, sw := range sqlWorkers {
		w := &Worker{}
		w.FromSQL(sw)
		m.add(w)
	}
	return m, nil
}

func (m *WorkerManager) ServableTargets() []string {
	servable := make([]string, 0)
	for t, n := range m.nForTag {
		if n != 0 {
			servable = append(servable, t)
		}
	}
	return servable
}

func (m *WorkerManager) Add(w *Worker) error {
	_, ok := m.worker[w.addr]
	if ok {
		return fmt.Errorf("worker already exists: %v", w.addr)
	}
	err := m.WorkerService.AddWorker(w.ToSQL())
	if err != nil {
		return err
	}
	m.add(w)
	return nil
}

// add adds a worker to the WorkerManager without update a db.
// Use Add instead of this, except when it is really needed.
func (m *WorkerManager) add(w *Worker) {
	addr := strings.Split(w.addr, ":")[0] // remove port
	for _, g := range m.workerGroups {
		if g.Match(addr) {
			w.group = g
			break
		}
	}
	w.update = m.WorkerService.UpdateWorker
	m.worker[w.addr] = w
}

func (m *WorkerManager) Bye(workerAddr string) error {
	w, ok := m.worker[workerAddr]
	if !ok {
		return fmt.Errorf("worker not found: %v", workerAddr)
	}
	if w.task != "" {
		// TODO: how to cancel the task?
	}
	olds := w.status
	if olds == WorkerNotFound {
		return fmt.Errorf("worker already logged off: %v", workerAddr)
	}
	s := WorkerNotFound
	t := ""
	err := w.Update(WorkerUpdater{
		Status: &s,
		Task:   &t,
	})
	if err != nil {
		return err
	}
	m.workers.Remove(w)
	for _, t := range w.group.ServeTargets {
		m.nForTag[t] -= 1
		if m.nForTag[t] < 0 {
			panic("shouldn't happen")
		}
	}
	return nil
}

func (m *WorkerManager) FindByAddr(addr string) *Worker {
	return m.worker[addr]
}

// Ready reports that a worker is ready for a new task.
// NOTE: It should be only called by the worker through workerFarm.
func (m *WorkerManager) Ready(w *Worker) error {
	s := WorkerReady
	t := ""
	err := w.Update(WorkerUpdater{
		Status: &s,
		Task:   &t,
	})
	if err != nil {
		return err
	}
	m.workers.Push(w)
	for _, t := range w.group.ServeTargets {
		m.nForTag[t] += 1
	}
	go func() { m.ReadyCh <- struct{}{} }()
	return nil
}

func (m *WorkerManager) Pop(target string) *Worker {
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

func (m *WorkerManager) Push(w *Worker) {
	m.workers.Push(w)
}

// SendPing sends a ping to a worker.
// The worker will let us know the task the worker is currently working on.
// It will return an error if the communication is failed.
func (m *WorkerManager) SendPing(w *Worker) (string, error) {
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := c.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return "", err
	}
	return resp.TaskId, err
}

func (m *WorkerManager) SendTask(w *Worker, t *Task) error {
	s := WorkerRunning
	tid := t.ID()
	err := w.Update(WorkerUpdater{
		Status: &s,
		Task:   &tid,
	})
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.RunRequest{}
	req.Id = t.ID()
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
	return nil
}

func (m *WorkerManager) SendCancelTask(w *Worker, t *Task) (err error) {
	log.Printf("cancel: %v %v", w.addr, t.ID())
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CancelRequest{}
	req.Id = t.ID()

	_, err = c.Cancel(ctx, req)
	if err != nil {
		return err
	}
	w.status = WorkerCooling
	w.task = ""
	return nil
}
