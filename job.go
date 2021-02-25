package coco

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

// JobFilter is a job filter for searching jobs.
type JobFilter struct {
	Target string
}

// Job is a job, user sended to server to run them in a farm.
type Job struct {
	// Job is a Task. Please also check Task.
	*Task

	// order is the order number of the job.
	// It has a same value with ID but of int type.
	// Think ID is external type, order is internal type of Job.
	order int

	// Target defines which worker groups should work for the job.
	// A target can be served by multiple worker groups.
	// If the target is not defined, it can be served with worker groups
	// those can serve all targets.
	Target string

	// AutoRetry is number of maximum auto retry for tasks when they are failed.
	// When user retries the job, retry count for tasks will be reset to 0.
	AutoRetry int

	// tasks are all tasks of the job, which is different from Subtasks.
	// The order is walk order. So one can walk tasks by just iterate through it.
	tasks []*Task

	// Job is a mutex.
	sync.Mutex

	//
	// NOTE:
	//
	// 	Fields below should be used/changed after hold the Job's lock.
	//
	//

	// CurrentPriority is the job's task priority waiting at the time.
	// Jobs are compete each other with their current priority.
	// Higher values take precedence to lower values.
	// Negative values will corrected to 0, the lowest priority value.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	CurrentPriority int

	// blocked indicates whether the job has blocked due to a failed/unfinished task.
	blocked bool
}

// ID returns the job's order number as a string.
func (j *Job) ID() string {
	return strconv.Itoa(j.order)
}

// infoFromJobID returns the job's order number which is basically an id but of type int.
// It will return an error as a second argument if the id string cannot be converted to int.
func infoFromJobID(id string) (int, error) {
	ord, err := strconv.Atoi(id)
	if err != nil {
		return -1, fmt.Errorf("invalid job id: %v", id)
	}
	return ord, nil
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
		ID              string
		Status          string
		Title           string
		Priority        int
		CurrentPriority int
		Subtasks        []*Task
		SerialSubtasks  bool
	}{
		ID:              j.ID(),
		Status:          j.Status().String(),
		Title:           j.Title,
		Priority:        j.Priority,
		CurrentPriority: j.CurrentPriority,
		Subtasks:        j.Subtasks,
		SerialSubtasks:  j.SerialSubtasks,
	}
	return json.Marshal(m)
}

// JobManager manages jobs and pops tasks to run the commands by workers.
type JobManager struct {
	sync.Mutex

	// nextOrder indicates next job's order.
	// It will be incremented when a job has added.
	nextOrder int

	// job is a map of an order to the job.
	job map[int]*Job

	// jobs is a job heap for PopTask.
	// Delete a job doesn't delete the job in this heap,
	// Because it is expensive to search an item from heap.
	// It will be deleted when it is popped from PopTask.
	jobs *jobHeap

	// CancelTaskCh pass tasks to make them canceled and
	// stop the processes running by workers.
	CancelTaskCh chan *Task
}

// NewJobManager creates a new JobManager.
func NewJobManager() *JobManager {
	m := &JobManager{}
	m.job = make(map[int]*Job)
	m.jobs = newJobHeap()
	m.CancelTaskCh = make(chan *Task)
	return m
}

// Get gets a job with a job id.
func (m *JobManager) Get(id string) (*Job, error) {
	ord, err := infoFromJobID(id)
	if err != nil {
		return nil, err
	}
	return m.job[ord], nil
}

// GetTask gets a task with a task id.
func (m *JobManager) GetTask(id string) (*Task, error) {
	ord, n, err := infoFromTaskID(id)
	if err != nil {
		return nil, err
	}
	return m.job[ord].tasks[n], nil
}

// Add adds a job to the manager.
func (m *JobManager) Add(j *Job) (int, error) {
	if j == nil {
		return -1, fmt.Errorf("nil job cannot be added")
	}
	err := j.Validate()
	if err != nil {
		return -1, err
	}
	initJob(j)

	j.order = m.nextOrder
	m.nextOrder++

	m.job[j.order] = j

	// didn't hold lock of the job as the job will not get published
	// until Add method returns.

	heap.Push(m.jobs, j)

	// set priority for the very first leaf task.
	peek := j.Peek()
	// peek can be nil, when the job doesn't have any leaf task.
	if peek != nil {
		j.CurrentPriority = peek.CalcPriority()
	}
	return j.order, nil
}

// Jobs searches jobs with a job filter.
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
		return jobs[i].order < jobs[j].order
	})
	return jobs
}

// Cancel cancels a job.
// Both running and waiting tasks of the job will be marked as failed,
// and commands executing from running tasks will be canceled right away.
func (m *JobManager) Cancel(id int) error {
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
func (m *JobManager) Retry(id string) error {
	ord, err := infoFromJobID(id)
	if err != nil {
		return err
	}
	j, ok := m.job[ord]
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
func (m *JobManager) Delete(id string) error {
	ord, err := infoFromJobID(id)
	if err != nil {
		return err
	}
	_, ok := m.job[ord]
	if !ok {
		return fmt.Errorf("cannot find the job: %v", id)
	}
	delete(m.job, ord)
	return nil
}

// PopTask returns a task that has highest priority for given targets.
// When every job has done or blocked, it will return nil.
func (m *JobManager) PopTask(targets []string) *Task {
	if len(targets) == 0 {
		return nil
	}
	for {
		if m.jobs.Len() == 0 {
			return nil
		}
		j := heap.Pop(m.jobs).(*Job)
		_, ok := m.job[j.order]
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
