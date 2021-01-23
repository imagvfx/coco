package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/rs/xid"
)

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// NOTE: Private fields of this struct should be read-only after the initialization.
	// Otherwise, this program will get racy.

	sync.Mutex

	// id is the order number of the job.
	id string

	// Job is a Task.
	// Some of the Task's field should be explained in Job's context.
	//
	// Task.Title is human readable title for job.
	//
	// Task.Priority sets the job's default priority.
	// Jobs are compete each other with priority.
	// Job's priority could be temporarily updated by a task that waits at the time.
	// Higher values take precedence to lower values.
	// Negative values will corrected to 0, the lowest priority value.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	*Task

	// CurrentPriority is the job's task priority waiting at the time.
	CurrentPriority int
}

func (j *Job) MarshalJSON() ([]byte, error) {
	m := struct {
		ID              string
		Status          string
		Title           string
		Priority        int
		CurrentPriority int
		Subtasks        []*Task
		SerialSubtasks  bool
	}{
		ID:              j.id,
		Status:          j.Status().String(j.IsLeaf()),
		Title:           j.Title,
		Priority:        j.Priority,
		CurrentPriority: j.CurrentPriority,
		Subtasks:        j.Subtasks,
		SerialSubtasks:  j.SerialSubtasks,
	}
	return json.Marshal(m)
}

func (j *Job) WalkTaskFn(fn func(t *Task)) {
	walkFromFn(j.Task, fn)
}

func (j *Job) WalkLeafTaskFn(fn func(t *Task)) {
	leafFn := func(t *Task) {
		if t.IsLeaf() {
			fn(t)
		}
	}
	walkFromFn(j.Task, leafFn)
}

func walkFromFn(t *Task, fn func(t *Task)) {
	down := true
	for {
		if down {
			fn(t)
			if len(t.Subtasks) != 0 {
				t = t.Subtasks[0]
				continue
			}
		}
		if t.next != nil {
			down = true
			t = t.next
			continue
		}
		if t.parent != nil {
			down = false
			t = t.parent
			continue
		}
		// back to root
		return
	}
}

type jobHeap []*Job

func (h jobHeap) Len() int {
	return len(h)
}

func (h jobHeap) Less(i, j int) bool {
	h[i].Lock()
	h[j].Lock()
	defer h[i].Unlock()
	defer h[j].Unlock()
	if h[i].CurrentPriority > h[j].CurrentPriority {
		return true
	}
	if h[i].CurrentPriority < h[j].CurrentPriority {
		return false
	}
	return h[i].id < h[j].id
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

type jobTaskPopper struct {
	job         *Job
	pTaskGroups [][]*Task
	// index to the next task in pTaskGroups to pop
	i, j int
	done bool
}

func newJobTaskPopper(j *Job) *jobTaskPopper {
	return &jobTaskPopper{
		job:         j,
		pTaskGroups: parallelTaskGroups(j),
	}
}

func (p *jobTaskPopper) Pop() *Task {
	for {
		t := p.pop()
		if t == nil {
			return nil
		}
		if t.Status() != TaskDone {
			return t
		}
	}
}

func (p *jobTaskPopper) pop() *Task {
	if len(p.pTaskGroups) == 0 {
		// not possible, but let's not panic.
		return nil
	}
	if len(p.pTaskGroups) == 1 && len(p.pTaskGroups[0]) == 0 {
		// when the job doesn't have leaf tasks.
		return nil
	}
	if p.done {
		return nil
	}
	if p.j == len(p.pTaskGroups[p.i]) {
		// all the task of the group popped, we can move to the next group
		// only if all the tasks in current group has been done
		p.job.Lock()
		defer p.job.Unlock()
		for _, t := range p.pTaskGroups[p.i] {
			if t.Status() != TaskDone {
				return nil
			}
		}
		p.i++
		p.j = 0
	}
	if p.i == len(p.pTaskGroups) {
		p.done = true
		return nil
	}
	defer func() {
		if p.j < len(p.pTaskGroups[p.i]) {
			p.j++
		}
	}()
	return p.pTaskGroups[p.i][p.j]
}

func (p *jobTaskPopper) Peek() *Task {
	i := p.i
	j := p.j
	done := p.done
	defer func() {
		p.i = i
		p.j = j
		p.done = done
	}()
	return p.Pop()
}

func (p *jobTaskPopper) Reset() {
	p.i = 0
	p.j = 0
	p.done = false
}

func (p *jobTaskPopper) Done() bool {
	return p.done
}

// parallelTaskGroups returns task groups, those tasks in a group are parallelly executable.
func parallelTaskGroups(j *Job) [][]*Task {
	j.Lock()
	defer j.Unlock()
	grps := make([][]*Task, 0)
	grps = append(grps, make([]*Task, 0))
	for _, subt := range j.Subtasks {
		grps = addParallelTaskGroups(subt, grps)
	}
	return grps
}

// addParallelTaskGroups modifies the last or adds a new task group to the input groups.
func addParallelTaskGroups(t *Task, grps [][]*Task) [][]*Task {
	for _, subt := range t.Subtasks {
		if t.SerialSubtasks {
			last := grps[len(grps)-1]
			if len(last) != 0 {
				grps = append(grps, make([]*Task, 0))
			}
		}
		if subt.IsLeaf() {
			last := grps[len(grps)-1]
			last = append(last, subt)
			// pointer to the last may changed by the append.
			grps[len(grps)-1] = last
			continue
		}
		grps = addParallelTaskGroups(subt, grps)
	}
	return grps
}

type jobManager struct {
	sync.Mutex
	nextJobID  int
	job        map[string]*Job
	jobBlocked map[string]bool
	// jobs may have cancelled jobs.
	// PopTask should handle this properly.
	jobs         *jobHeap
	task         map[string]*Task
	tasks        map[string]*jobTaskPopper
	CancelTaskCh chan *Task
}

func newJobManager() *jobManager {
	m := &jobManager{}
	m.job = make(map[string]*Job)
	m.jobBlocked = make(map[string]bool)
	m.jobs = &jobHeap{}
	m.task = make(map[string]*Task)
	m.tasks = make(map[string]*jobTaskPopper)
	m.CancelTaskCh = make(chan *Task)
	return m
}

func (m *jobManager) Get(id string) *Job {
	m.Lock()
	defer m.Unlock()
	return m.job[id]
}

func (m *jobManager) GetTask(id string) *Task {
	m.Lock()
	defer m.Unlock()
	return m.task[id]
}

func (m *jobManager) Add(j *Job) (string, error) {
	if j == nil {
		return "", fmt.Errorf("nil job cannot be added")
	}
	if len(j.Subtasks) == 0 {
		return "", fmt.Errorf("a job should have at least one subtask")
	}
	initJob(j)

	j.id = strconv.Itoa(m.nextJobID)
	m.nextJobID++

	m.Lock()
	defer m.Unlock()
	m.job[j.id] = j

	// didn't hold lock of the job as the job will not get published
	// until Add method returns.
	// NOTE: but anyway, the heap operations following will hold lock on jobs.

	heap.Push(m.jobs, j)

	tasks, ok := m.tasks[j.id]
	if !ok {
		tasks = newJobTaskPopper(j)
		m.tasks[j.id] = tasks
	}
	j.WalkTaskFn(func(t *Task) {
		m.task[t.id] = t
	})

	// set priority for the very first leaf task.
	peek := tasks.Peek()
	// peek can be nil, when the job doesn't have any leaf task.
	if peek != nil {
		j.CurrentPriority = peek.CalcPriority()
	}
	return j.id, nil
}

// initJob inits a job's tasks.
// initJob returns unmodified pointer of the job, for in case
// when user wants to directly assign to a variable. (see test code)
func initJob(j *Job) *Job {
	initJobTasks(j.Task, j, nil, 0, 0)
	return j
}

// initJobTasks inits a job's tasks recursively before it is added to jobManager.
// No need to hold the lock.
func initJobTasks(t *Task, j *Job, parent *Task, nth, i int) int {
	t.id = xid.New().String()
	t.job = j
	t.parent = parent
	t.nthChild = nth
	if t.IsLeaf() {
		t.num = i
		i++
	}
	if t.Priority < 0 {
		// nagative priority is invalid.
		t.Priority = 0
	}
	t.Stat = &branchStat{}
	iOld := i
	var prev *Task
	for nth, subt := range t.Subtasks {
		if prev != nil {
			prev.next = subt
		}
		i = initJobTasks(subt, j, t, nth, i)
		prev = subt
	}
	t.Stat.nWaiting = i - iOld
	return i
}

func (m *jobManager) Cancel(id string) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	if j.Status() == TaskCancelled {
		return fmt.Errorf("job has already cancelled: %v", id)
	}
	if j.Status() == TaskDone {
		// TODO: the job's status doesn't get changed to done yet.
		return fmt.Errorf("job has already Done: %v", id)
	}
	// indicate the job and it's tasks are cancelled, first.
	j.WalkLeafTaskFn(func(t *Task) {
		t.SetStatus(TaskCancelled)
	})
	// Delete the job from m.jobs (heap) will be expensive.
	// Let PopTask do the job.
	delete(m.tasks, id)
	go func() {
		j.WalkLeafTaskFn(func(t *Task) {
			m.CancelTaskCh <- t
		})
	}()
	return nil
}

func (m *jobManager) Retry(id string) error {
	m.Lock()
	defer m.Unlock()
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	j.WalkLeafTaskFn(func(t *Task) {
		if t.Status() == TaskFailed {
			t.SetStatus(TaskWaiting)
		}
	})
	m.tasks[j.id].Reset()
	j.Unlock()
	heap.Push(m.jobs, j)
	return nil
}

func (m *jobManager) Delete(id string) error {
	m.Lock()
	defer m.Unlock()
	_, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	delete(m.job, id)
	// Delete the job from m.jobs (heap) will be expensive.
	// Let PopTask do the job.
	delete(m.tasks, id)
	return nil
}

func (m *jobManager) PopTask() *Task {
	m.Lock()
	defer m.Unlock()
	for {
		if m.jobs.Len() == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*Job)
		_, ok := m.job[j.id]
		if !ok {
			// the job deleted
			continue
		}
		if j.Status() == TaskCancelled {
			// the job cancelled
			continue
		}
		if j.Status() == TaskFailed {
			// one or more tasks of the job failed,
			// block the job until user retry the tasks.
			continue
		}

		tasks := m.tasks[j.id]
		t := tasks.Pop()

		// check there is any leaf task left.
		if !tasks.Done() {
			peek := tasks.Peek()
			// peek can be nil when next tasks.Pop has blocked for some reason.
			if peek != nil {
				j.Lock()
				// the peeked task is also cared by this lock.
				j.CurrentPriority = peek.CalcPriority()
				j.Unlock()
				heap.Push(m.jobs, j)
			} else {
				m.jobBlocked[j.id] = true
			}
		}

		return t
	}
}
