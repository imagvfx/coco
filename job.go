package coco

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/rs/xid"
)

type JobID int

type JobFilter struct {
	Target string
}

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// NOTE: Private fields of this struct should be read-only after the initialization.
	// Otherwise, this program will get racy.

	sync.Mutex

	// id is the order number of the job.
	id JobID

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

	// Target defines which worker groups should work for the job.
	// A target can be served by multiple worker groups.
	// If the target is not defined, it can be served with worker groups
	// those can serve all targets.
	Target string

	// AutoRetry is number of maximum auto retry for tasks when they are failed.
	// When user retries the job, retry count for tasks will be reset to 0.
	AutoRetry int

	// blocked indicates whether the job has blocked due to a failed/unfinished task.
	blocked bool

	// tasks are all tasks of the job, which is different from Subtasks.
	// The order is walk order. So one can walk tasks by just iterate through it.
	tasks []*Task
}

// Validate validates a raw Job that is sended from user.
func (j *Job) Validate() error {
	if len(j.Subtasks) == 0 {
		return fmt.Errorf("a job should have at least one subtask")
	}
	if j.Title == "" {
		j.Title = "untitled"
	}
	return nil
}

// initJob inits a job's tasks.
// initJob returns unmodified pointer of the job, for in case
// when user wants to directly assign to a variable. (see test code)
func initJob(j *Job) *Job {
	_, j.tasks = initJobTasks(j.Task, j, nil, 0, 0, []*Task{})
	return j
}

// initJobTasks inits a job's tasks recursively before it is added to jobManager.
// No need to hold the lock.
func initJobTasks(t *Task, j *Job, parent *Task, nth, i int, tasks []*Task) (int, []*Task) {
	t.ID = TaskID(xid.New().String())
	t.num = len(tasks)
	tasks = append(tasks, t)
	if t.Title == "" {
		t.Title = "untitled"
	}
	t.Job = j
	t.parent = parent
	t.nthChild = nth
	t.isLeaf = len(t.Subtasks) == 0
	if t.isLeaf {
		i++
	}
	if t.Priority < 0 {
		// negative priority is invalid.
		t.Priority = 0
	}
	iOld := i
	for nth, subt := range t.Subtasks {
		i, tasks = initJobTasks(subt, j, t, nth, i, tasks)
	}
	t.Stat = newBranchStat(i - iOld)
	t.popIdx = 0
	return i, tasks
}

// MarshalJSON implements json.Marshaler interface.
func (j *Job) MarshalJSON() ([]byte, error) {
	m := struct {
		ID              JobID
		Status          string
		Title           string
		Priority        int
		CurrentPriority int
		Subtasks        []*Task
		SerialSubtasks  bool
	}{
		ID:              j.id,
		Status:          j.Status().String(),
		Title:           j.Title,
		Priority:        j.Priority,
		CurrentPriority: j.CurrentPriority,
		Subtasks:        j.Subtasks,
		SerialSubtasks:  j.SerialSubtasks,
	}
	return json.Marshal(m)
}

type jobHeap struct {
	heap []*Job
}

func newJobHeap() *jobHeap {
	return &jobHeap{
		heap: make([]*Job, 0),
	}
}

func (h jobHeap) Len() int {
	return len(h.heap)
}

func (h jobHeap) Less(i, j int) bool {
	if h.heap[i].CurrentPriority > h.heap[j].CurrentPriority {
		return true
	}
	if h.heap[i].CurrentPriority < h.heap[j].CurrentPriority {
		return false
	}
	return h.heap[i].id < h.heap[j].id
}

func (h jobHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

func (h *jobHeap) Push(el interface{}) {
	h.heap = append(h.heap, el.(*Job))
}

func (h *jobHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.heap = old[:n-1]
	return el
}

type JobManager struct {
	sync.Mutex
	nextJobID JobID

	// Job related informations.
	// When a job is deleted, those related info should be deleted all toghther,
	// except `jobs` heap. It is expensive to search an item from heap.
	// So, deleted job in `jobs` will be deleted when it is popped from PopTask.
	job  map[JobID]*Job
	task map[TaskID]*Task
	jobs *jobHeap

	CancelTaskCh chan *Task
}

func NewJobManager() *JobManager {
	m := &JobManager{}
	m.job = make(map[JobID]*Job)
	m.jobs = newJobHeap()
	m.task = make(map[TaskID]*Task)
	m.CancelTaskCh = make(chan *Task)
	return m
}

func (m *JobManager) Get(id JobID) *Job {
	return m.job[id]
}

func (m *JobManager) GetTask(id TaskID) *Task {
	return m.task[id]
}

func (m *JobManager) Add(j *Job) (JobID, error) {
	if j == nil {
		return -1, fmt.Errorf("nil job cannot be added")
	}
	err := j.Validate()
	if err != nil {
		return -1, err
	}
	initJob(j)

	j.id = m.nextJobID
	m.nextJobID++

	m.job[j.id] = j

	// didn't hold lock of the job as the job will not get published
	// until Add method returns.

	heap.Push(m.jobs, j)

	for _, t := range j.tasks {
		m.task[t.ID] = t
	}

	// set priority for the very first leaf task.
	peek := j.Peek()
	// peek can be nil, when the job doesn't have any leaf task.
	if peek != nil {
		j.CurrentPriority = peek.CalcPriority()
	}
	return j.id, nil
}

func (m *JobManager) Jobs(filter JobFilter) []*Job {
	jobs := make([]*Job, 0, len(m.job))
	for _, j := range m.job {
		if filter.Target == "" {
			jobs = append(jobs, j)
			continue
		}
		// list only jobs having the tag
		match := false
		if filter.Target == j.Target {
			match = true
		}
		if match {
			jobs = append(jobs, j)
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].id < jobs[j].id
	})
	return jobs
}

// Cancel cancels a job.
// Both running and waiting tasks of the job will be marked as failed,
// and commands executing from running tasks will be canceled right away.
func (m *JobManager) Cancel(id JobID) error {
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	if j.Status() == TaskDone {
		// TODO: the job's status doesn't get changed to done yet.
		return fmt.Errorf("job has already Done: %v", id)
	}
	for _, t := range j.tasks {
		if !t.isLeaf {
			continue
		}
		if t.status == TaskRunning {
			go func() {
				// Let worker knows that the task is canceled.
				m.CancelTaskCh <- t
			}()
		}
		if t.status != TaskDone {
			t.SetStatus(TaskFailed)
		}
	}
	return nil
}

// Retry resets all tasks of the job's retry count to 0,
// then retries all of the failed tasks,
func (m *JobManager) Retry(id JobID) error {
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	j.Lock()
	defer j.Unlock()
	for _, t := range j.tasks {
		if !t.isLeaf {
			continue
		}
		t.retry = 0
		if t.Status() == TaskFailed {
			t.SetStatus(TaskWaiting)
			t.Push()
		}
	}
	if j.Peek() == nil {
		// The job has blocked or popped all tasks.
		// The job isn't in m.jobs for both cases.
		heap.Push(m.jobs, j)
	}
	return nil
}

// Delete deletes a job irrecoverably.
func (m *JobManager) Delete(id JobID) error {
	j, ok := m.job[id]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	delete(m.job, id)
	for _, t := range j.tasks {
		delete(m.task, t.ID)
	}
	return nil
}

func (m *JobManager) PopTask(targets []string) *Task {
	if len(targets) == 0 {
		return nil
	}
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
		if j.Status() == TaskFailed {
			// one or more tasks of the job failed,
			// block the job until user retry the tasks.
			continue
		}

		servable := false
		for _, t := range targets {
			if t == "*" || t == j.Target {
				servable = true
				break
			}
		}
		if !servable {
			defer func() {
				// The job has remaing tasks. So, push back this job
				// after we've found a servable job.
				m.jobs.Push(j)
			}()
			continue
		}

		j.Lock()
		defer j.Unlock()
		t, done := j.Pop()
		// check there is any leaf task left.
		if !done {
			peek := j.Peek()
			// peek can be nil when next Job.Pop has blocked for some reason.
			if peek != nil {
				// the peeked task is also cared by this lock.
				p := peek.CalcPriority()
				j.CurrentPriority = p
				heap.Push(m.jobs, j)
			} else {
				j.blocked = true
			}
		}

		return t
	}
}

// pushTask pushes the task to it's job, so it can popped again.
// When `retry` argument is true, it will act as retry mode.
// It will return true when the push has succeed.
func (m *JobManager) pushTask(t *Task, retry bool) (ok bool) {
	j := t.Job
	peek := j.Peek()
	if peek == nil {
		// The job has blocked or popped all tasks.
		// The job isn't in m.jobs for both cases.
		// But the job's priority should be recalculated first.
		defer func() {
			if !ok {
				return
			}
			heap.Push(m.jobs, j)
			j.blocked = false
		}()
	}

	if retry {
		ok = t.Retry()
		if !ok {
			return ok
		}
	} else {
		t.Push()
		ok = true
	}
	// Peek again to reflect possible change of the job's priority.
	peek = j.Peek() // peek should not be nil
	p := peek.CalcPriority()
	j.CurrentPriority = p
	return ok
}

// PushTask pushes the task to it's job so it can popped again.
// It should be used when there is server/communication error.
// Use PushTaskForRetry for failed tasks.
func (m *JobManager) PushTask(t *Task) {
	m.pushTask(t, false)
}

// RetryTask pushes the task to it's job so it can be retried when it's failed.
// If the task already reaches to maxmium AutoRetry count, it will
// refuse to push and return false.
func (m *JobManager) PushTaskForRetry(t *Task) bool {
	return m.pushTask(t, true)
}
