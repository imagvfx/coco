package coco

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/imagvfx/coco/lib/container"
	"github.com/imagvfx/coco/pb"
	"github.com/imagvfx/coco/service"
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
	// It is nil if the worker is in idle.
	task *TaskID

	group *WorkerGroup

	update func(service.WorkerUpdater) error
}

func NewWorker(addr string) *Worker {
	return &Worker{addr: addr, status: WorkerReady}
}

// ToSQL converts a Worker into a SQLWorker.
func (w *Worker) ToSQL() *service.Worker {
	var maybeJob *int
	var maybeTask *int
	if w.task != nil {
		maybeJob = &w.task[0]
		maybeTask = &w.task[1]
	}
	sw := &service.Worker{
		Addr:   w.addr,
		Status: int(w.status),
		Job:    maybeJob,
		Task:   maybeTask,
	}
	return sw
}

// FromSQL converts a SQLWorker into a Worker.
func (w *Worker) FromSQL(sw *service.Worker) {
	w.addr = sw.Addr
	w.status = WorkerStatus(sw.Status)
	w.task = nil
	if sw.Job != nil {
		w.task = &TaskID{*sw.Job, *sw.Task}
	}
}

// Update updates a worker in both the program and the db.
func (w *Worker) Update(u service.WorkerUpdater) error {
	u.Addr = w.addr
	err := w.update(u)
	if err != nil {
		return err
	}
	if u.UpdateStatus {
		w.status = WorkerStatus(u.Status)
	}
	if u.UpdateTask {
		w.task = nil
		if u.Job != nil {
			w.task = &TaskID{*u.Job, *u.Task}
		}
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
	WorkerService service.WorkerService
	worker        map[string]*Worker
	workers       *container.UniqueQueue
	workerGroups  []*WorkerGroup
	nForTag       map[string]int
	// ReadyCh tries fast matching of a worker and a task.
	ReadyCh chan struct{}
}

// NewWorkerManager create a new WorkerManager.
// It restores previous data with WorkerService.
func NewWorkerManager(ws service.WorkerService, wgrps []*WorkerGroup) (*WorkerManager, error) {
	m := &WorkerManager{}
	m.WorkerService = ws
	m.worker = make(map[string]*Worker)
	m.workers = container.NewUniqueQueue()
	m.workerGroups = wgrps
	m.nForTag = make(map[string]int)
	m.ReadyCh = make(chan struct{})
	err := m.restore()
	if err != nil {
		return nil, err
	}
	return m, nil
}

// restore restores a WorkerManager from a db with WorkerService.
func (m *WorkerManager) restore() error {
	sqlWorkers, err := m.WorkerService.FindWorkers(service.WorkerFilter{})
	if err != nil {
		return err
	}
	for _, sw := range sqlWorkers {
		w := &Worker{}
		w.FromSQL(sw)
		m.add(w)
	}
	return nil
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
	if w.task != nil {
		// TODO: how to cancel the task?
	}
	olds := w.status
	if olds == WorkerNotFound {
		return fmt.Errorf("worker already logged off: %v", workerAddr)
	}
	var maybeJob *int
	var maybeTask *int
	if w.task != nil {
		maybeJob = &w.task[0]
		maybeTask = &w.task[1]
	}
	err := w.Update(service.WorkerUpdater{
		UpdateStatus: true,
		Status:       int(WorkerNotFound),
		UpdateTask:   true,
		Job:          maybeJob,
		Task:         maybeTask,
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
	if w == nil {
		return fmt.Errorf("nil worker")
	}
	err := w.Update(service.WorkerUpdater{
		UpdateStatus: true,
		Status:       int(WorkerReady),
		UpdateTask:   true,
		Job:          nil,
		Task:         nil,
	})
	if err != nil {
		return err
	}
	m.workers.Push(w)
	if w.group != nil {
		for _, t := range w.group.ServeTargets {
			m.nForTag[t] += 1
		}
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
// The worker will let us know the task id the worker is currently working on.
// It returns a pointer of the task id. Or returns nil if the worker is currently in idle.
// It will return an error if the communication is failed.
func (m *WorkerManager) SendPing(w *Worker) (*TaskID, error) {
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := c.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return nil, err
	}
	if resp.TaskId == "" {
		return nil, nil
	}
	tid, err := TaskIDFromString(resp.TaskId)
	if err != nil {
		return nil, err
	}
	return &tid, nil
}

func (m *WorkerManager) SendTask(w *Worker, t *Task) error {
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.RunRequest{}
	req.Id = t.ID.String()
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
	log.Printf("cancel: %v %v", w.addr, t.ID)
	conn, err := grpc.Dial(w.addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.CancelRequest{}
	req.Id = t.ID.String()

	_, err = c.Cancel(ctx, req)
	if err != nil {
		return err
	}
	return nil
}
