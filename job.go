package coco

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

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
	// Jobs with higher values take precedence to jobs with lower values.
	// If multiple jobs are having same priority, server will take a job with rotation rule.
	CurrentPriority int
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

// initJob inits a job and it's tasks.
// initJob returns unmodified pointer of the job, for in case
// when user wants to directly assign to a variable. (see test code)
func (j *Job) Init(js JobService) *Job {
	_, j.tasks = initJobTasks(j.Task, j, nil, 0, 0, []*Task{})
	j.CurrentPriority = j.Peek().CalcPriority()
	for _, t := range j.tasks {
		t.update = js.UpdateTask
	}
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

// ToSQL converts a Job into a SQLJob.
func (j *Job) ToSQL() *SQLJob {
	s := &SQLJob{
		Order:     j.order,
		Target:    j.Target,
		AutoRetry: j.AutoRetry,
		Tasks:     make([]*SQLTask, len(j.tasks)),
	}
	for i, t := range j.tasks {
		s.Tasks[i] = t.ToSQL()
	}
	return s
}

// FromSQL converts a SQLJob into a Job.
func (j *Job) FromSQL(sj *SQLJob) {
	j.order = sj.Order
	j.Target = sj.Target
	j.AutoRetry = sj.AutoRetry
	j.tasks = make([]*Task, len(sj.Tasks))
	for i, st := range sj.Tasks {
		t := &Task{
			Job: j,
		}
		t.FromSQL(st)
		if i != 0 {
			// Add a task to the parent's Subtasks.
			// The parent should have already been created when the child is created.
			parent := j.tasks[st.ParentNum]
			parent.Subtasks = append(parent.Subtasks, t)
		}
		j.tasks[i] = t
	}
	j.Task = j.tasks[0]
}

// JobManager manages jobs and pops tasks to run the commands by workers.
type JobManager struct {
	sync.Mutex

	// JobService allows us to communicate with db for Jobs.
	JobService JobService

	// job is a map of an order to the job.
	job map[int]*Job

	// jobs is a job heap for PopTask.
	// Delete a job doesn't delete the job in this heap,
	// Because it is expensive to search an item from heap.
	// It will be deleted when it is popped from PopTask.
	jobs *uniqueHeap

	// CancelTaskCh pass tasks to make them canceled and
	// stop the processes running by workers.
	CancelTaskCh chan *Task
}

func popCompare(i, j interface{}) bool {
	iv := i.(*Job)
	jv := j.(*Job)
	if iv.CurrentPriority > jv.CurrentPriority {
		return true
	}
	if iv.CurrentPriority < jv.CurrentPriority {
		return false
	}
	return iv.order < jv.order

}

// NewJobManager creates a new JobManager.
func NewJobManager(js JobService) *JobManager {
	m := &JobManager{}
	m.JobService = js
	m.job = make(map[int]*Job)
	m.jobs = newUniqueHeap(popCompare)
	m.CancelTaskCh = make(chan *Task)
	return m
}

// RestoreJobManager restores a JobManager from given JobService.
func RestoreJobManager(js JobService) (*JobManager, error) {
	m := NewJobManager(js)
	sqlJobs, err := m.JobService.FindJobs(JobFilter{})
	if err != nil {
		return nil, err
	}
	for _, sj := range sqlJobs {
		j := &Job{}
		j.FromSQL(sj)
		j.Init(m.JobService)
		// restore branch tasks status
		for _, t := range j.tasks {
			if !t.isLeaf {
				continue
			}
			parent := t.parent
			for parent != nil {
				parent.Stat.Change(TaskWaiting, t.status)
				parent = parent.parent
			}
		}
		j.restorePopIdx()
		m.job[j.order] = j
		heap.Push(m.jobs, j)
	}
	return m, nil
}

// Get gets a job with a job id.
func (m *JobManager) Get(id string) (*Job, error) {
	ord, err := infoFromJobID(id)
	if err != nil {
		return nil, err
	}
	j, ok := m.job[ord]
	if !ok {
		return nil, fmt.Errorf("job not exists: %v", ord)
	}
	return j, nil
}

// GetTask gets a task with a task id.
func (m *JobManager) GetTask(id string) (*Task, error) {
	ord, n, err := infoFromTaskID(id)
	if err != nil {
		return nil, err
	}
	j, ok := m.job[ord]
	if !ok {
		return nil, fmt.Errorf("job not exists: %v", id)
	}
	if n < 0 || n >= len(j.tasks) {
		return nil, fmt.Errorf("task not exists: %v", id)
	}
	return j.tasks[n], nil
}

// Add adds a job to the manager and return it's ID.
func (m *JobManager) Add(j *Job) (string, error) {
	if j == nil {
		return "", fmt.Errorf("nil job cannot be added")
	}
	err := j.Validate()
	if err != nil {
		return "", err
	}
	j.Init(m.JobService)

	j.order, err = m.JobService.AddJob(j.ToSQL())
	if err != nil {
		return "", err
	}

	m.job[j.order] = j
	heap.Push(m.jobs, j)

	return j.ID(), nil
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
func (m *JobManager) Cancel(id string) error {
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
			err := t.Update(TaskUpdater{
				Status: ptrTaskStatus(TaskFailed),
			})
			if err != nil {
				return err
			}

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
			err := t.Update(TaskUpdater{
				Status: ptrTaskStatus(TaskWaiting),
			})
			if err != nil {
				return err
			}
			t.Push()
		}
	}
	heap.Push(m.jobs, j)
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
			}
		}

		return t
	}
}

// PushTask pushes the task to it's job, so it can popped again.
func (m *JobManager) PushTask(t *Task) {
	j := t.Job
	t.Push()
	// Peek again to reflect possible change of the job's priority.
	peek := j.Peek() // peek should not be nil
	p := peek.CalcPriority()
	j.CurrentPriority = p
	heap.Push(m.jobs, j)
}
