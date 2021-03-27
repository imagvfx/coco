package coco

import (
	"fmt"
	"log"
	"time"

	"github.com/imagvfx/coco/service"
)

// Farm manages jobs and workers.
type Farm struct {
	FarmService service.FarmService
	jobman      *JobManager
	workerman   *WorkerManager
}

// NewFarm creates a new Farm.
// TODO: wgrps to generic config.
func NewFarm(services service.Services, wgrps []*WorkerGroup) (*Farm, error) {
	jobman, err := NewJobManager(services.JobService())
	if err != nil {
		return nil, err
	}
	workerman, err := NewWorkerManager(services.WorkerService(), wgrps)
	if err != nil {
		return nil, err
	}
	f := &Farm{
		FarmService: services.FarmService(),
		jobman:      jobman,
		workerman:   workerman,
	}
	return f, nil
}

func (f *Farm) Assign(worker string, task TaskID) error {
	err := f.updateAssign(service.AssignUpdater{
		Job:                task[0],
		Task:               task[1],
		UpdateTaskStatus:   true,
		TaskStatus:         int(TaskRunning),
		UpdateTaskAssignee: true,
		TaskAssignee:       worker,

		Worker:             worker,
		UpdateWorkerStatus: true,
		WorkerStatus:       int(WorkerRunning),
		UpdateWorkerTask:   true,
		WorkerJob:          &(task[0]),
		WorkerTask:         &(task[1]),
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *Farm) updateAssign(a service.AssignUpdater) error {
	t, err := f.jobman.GetTask(TaskID{a.Job, a.Task})
	if err != nil {
		return err
	}
	w := f.workerman.FindByAddr(a.Worker)
	if w == nil {
		return fmt.Errorf("worker not found: %v", a.Worker)
	}
	err = f.FarmService.UpdateAssign(a)
	if err != nil {
		return err
	}
	t.Assignee = a.Worker
	if a.UpdateTaskStatus {
		t.setStatus(TaskStatus(a.TaskStatus))
	}
	if a.UpdateTaskRetry {
		t.retry = a.TaskRetry
	}
	if a.UpdateWorkerStatus {
		w.status = WorkerStatus(a.WorkerStatus)
	}
	if a.UpdateWorkerTask {
		w.task = nil
		if a.WorkerJob != nil {
			w.task = &TaskID{*a.WorkerJob, *a.WorkerTask}
		}
	}
	return nil
}

// JobManager returns the job manager of the farm.
func (f *Farm) JobManager() *JobManager {
	return f.jobman
}

// JobManager returns the worker manager of the farm.
func (f *Farm) WorkerManager() *WorkerManager {
	return f.workerman
}

// RefreshWorker communicates with all remembered workers and refresh their status.
func (f *Farm) RefreshWorkers() {
	refresh := func(w *Worker) {
		f.workerman.Lock()
		defer f.workerman.Unlock()
		tid, err := f.workerman.SendPing(w)
		if err != nil {
			log.Print(err)
			var maybeJob *int
			var maybeTask *int
			if w.task != nil {
				maybeJob = &w.task[0]
				maybeTask = &w.task[1]
				t, err := f.jobman.GetTask(*w.task)
				if err != nil {
					log.Print(err)
					return
				}
				err = t.Update(service.TaskUpdater{
					UpdateStatus:   true,
					Status:         int(TaskFailed),
					UpdateAssignee: true,
					Assignee:       "",
				})
				if err != nil {
					log.Printf("couldn't update task: %v", w.task)
				}
			}
			err := w.Update(service.WorkerUpdater{
				UpdateStatus: true,
				Status:       int(WorkerNotFound),
				UpdateTask:   true,
				Job:          maybeJob,
				Task:         maybeTask,
			})
			if err != nil {
				log.Print(err)
			}
			return
		}
		if tid == w.task {
			return
		}
		if w.task != nil {
			log.Printf("worker is not running on expected task: %v", w.task)
			t, err := f.jobman.GetTask(*w.task)
			if err != nil {
				log.Print(err)
				return
			}
			err = t.Update(service.TaskUpdater{
				UpdateStatus:   true,
				Status:         int(TaskFailed),
				UpdateAssignee: true,
				Assignee:       "",
			})
			if err != nil {
				log.Printf("couldn't update task: %v", w.task)
			}
		}
		if tid != nil {
			log.Printf("worker is running on unexpected task: %v", tid)
			t, err := f.jobman.GetTask(*tid)
			if err != nil {
				log.Print(err)
				return
			}
			if t.Assignee != w.addr {
				f.jobman.CancelTaskCh <- t
			}
			// TODO: What should we do to the task?
		}
	}
	f.workerman.Lock()
	defer f.workerman.Unlock()
	for _, w := range f.workerman.worker {
		go refresh(w)
	}
}

// Ready indicates the worker is idle and waiting for commands to run.
func (f *Farm) Ready(addr string) error {
	// TODO: need to verify the worker
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		w = NewWorker(addr)
		err := f.workerman.Add(w)
		if err != nil {
			return err
		}
	}
	f.workerman.Ready(w)
	return nil
}

// Bye indicates the worker is in idle and waiting for commands to run.
func (f *Farm) Bye(addr string) error {
	// TODO: need to verify the worker
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		return fmt.Errorf("unknown worker: %v", addr)
	}
	if w.task != nil {
		f.jobman.Lock()
		defer f.jobman.Unlock()
		t, err := f.jobman.GetTask(*w.task)
		if err != nil {
			return err
		}
		j := t.Job
		j.Lock()
		defer j.Unlock()
		err = f.updateAssign(service.AssignUpdater{
			Job:                t.ID[0],
			Task:               t.ID[1],
			UpdateTaskStatus:   true,
			TaskStatus:         int(TaskFailed),
			UpdateTaskAssignee: true,
			TaskAssignee:       "",

			Worker:             addr,
			UpdateWorkerStatus: true,
			WorkerStatus:       int(WorkerNotFound),
			UpdateWorkerTask:   true,
			WorkerTask:         nil,
		})
		if err != nil {
			return err
		}
	}
	f.workerman.Bye(w.addr)
	return nil
}

// Done indicates the worker finished the requested task.
func (f *Farm) Done(addr string, task TaskID) error {
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	j := t.Job
	j.Lock()
	defer j.Unlock()
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		return fmt.Errorf("unknown worker: %v", addr)
	}
	if w.task == nil {
		return fmt.Errorf("worker has not assigned to any task: %v", addr)
	}
	if t.Assignee != addr || *w.task != task {
		// TODO: fail over
		return fmt.Errorf("unmatched task and worker information: %v.Assignee=%v, %v.task=%v", task, t.Assignee, addr, w.task)
	}
	err = f.updateAssign(service.AssignUpdater{
		Job:                t.ID[0],
		Task:               t.ID[1],
		UpdateTaskStatus:   true,
		TaskStatus:         int(TaskDone),
		UpdateTaskAssignee: true,
		TaskAssignee:       "",

		Worker:             addr,
		UpdateWorkerStatus: true,
		WorkerStatus:       int(WorkerReady),
		UpdateWorkerTask:   true,
		WorkerJob:          nil,
		WorkerTask:         nil,
	})
	if err != nil {
		return err
	}
	f.jobman.jobs.Push(j)
	f.workerman.Ready(w)
	return nil
}

// Failed indicates the worker failed to finish the requested task.
func (f *Farm) Failed(addr string, task TaskID) error {
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	j := t.Job
	j.Lock()
	defer j.Unlock()
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		return fmt.Errorf("unknown worker: %v", addr)
	}
	if w.task == nil {
		return fmt.Errorf("worker has not assigned to any task: %v", addr)
	}
	if t.Assignee != addr || *w.task != task {
		// TODO: fail over
		return fmt.Errorf("unmatched task and worker information: %v.Assignee=%v, %v.task=%v", task, t.Assignee, addr, *w.task)
	}
	ts := TaskFailed
	retry := t.CanRetry()
	if retry {
		t.retry++
		ts = TaskWaiting
	}
	err = f.updateAssign(service.AssignUpdater{
		Job:                t.ID[0],
		Task:               t.ID[1],
		UpdateTaskStatus:   true,
		TaskStatus:         int(ts),
		UpdateTaskRetry:    true,
		TaskRetry:          t.retry,
		UpdateTaskAssignee: true,
		TaskAssignee:       "",

		Worker:             addr,
		UpdateWorkerStatus: true,
		WorkerStatus:       int(WorkerReady),
		UpdateWorkerTask:   true,
		WorkerJob:          nil,
		WorkerTask:         nil,
	})
	if err != nil {
		return err
	}
	if retry {
		t.Push()
	}
	f.jobman.jobs.Push(j)
	f.workerman.Ready(w)
	return nil
}

func (f *Farm) match(worker string, task TaskID) error {
	w := f.workerman.FindByAddr(worker)
	if w == nil {
		return fmt.Errorf("worker not found: %v", worker)
	}
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	err = f.workerman.SendTask(w, t)
	if err != nil {
		return err
	}
	err = f.Assign(worker, task)
	if err != nil {
		return err
	}
	return nil
}

func (f *Farm) Matching() {
	match := func() {
		// ReadyCh gives faster matching loop when we are fortune.
		// There might have a chance that there was no job
		// when a worker sent a signal to ReadyCh.
		// Then the signal spent, no more signal will come
		// unless another worker sends a signal to the channel.
		// Below time.After case will helps us prevent deadlock
		// on matching loop in that case.
		select {
		case <-f.workerman.ReadyCh:
			break
		case <-time.After(time.Second):
			break
		}

		f.jobman.Lock()
		defer f.jobman.Unlock()
		f.workerman.Lock()
		defer f.workerman.Unlock()

		t := f.jobman.PopTask(f.workerman.ServableTargets())
		if t == nil {
			return
		}
		j := t.Job
		j.Lock()
		defer j.Unlock()
		cancel := len(t.Commands) == 0 || t.Status() == TaskFailed // eg. user canceled this task
		if cancel {
			return
		}
		w := f.workerman.Pop(t.Job.Target)
		if w == nil {
			panic("at least one worker should be able to serve this tag")
		}
		err := f.match(w.addr, t.ID)
		if err != nil {
			log.Print(err)
			f.jobman.PushTask(t)
			f.workerman.Push(w)
		}
	}

	for {
		match()
	}
}

func (f *Farm) cancel(worker string, task TaskID) error {
	w := f.workerman.FindByAddr(worker)
	if w == nil {
		return fmt.Errorf("worker not found: %v", worker)
	}
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	err = f.workerman.SendCancelTask(w, t)
	if err != nil {
		return err
	}
	err = f.updateAssign(service.AssignUpdater{
		Job:                t.ID[0],
		Task:               t.ID[1],
		UpdateTaskStatus:   true,
		TaskStatus:         int(TaskFailed),
		UpdateTaskAssignee: true,
		TaskAssignee:       "",

		Worker:             w.addr,
		UpdateWorkerStatus: true,
		WorkerStatus:       int(WorkerCooling),
		UpdateWorkerTask:   true,
		WorkerJob:          nil,
		WorkerTask:         nil,
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *Farm) Canceling() {
	cancel := func() {
		t := <-f.jobman.CancelTaskCh
		f.jobman.Lock()
		defer f.jobman.Unlock()
		t, err := f.jobman.GetTask(t.ID)
		if err != nil {
			log.Printf("failed to cancel task: %v", err)
			return
		}
		f.workerman.Lock()
		defer f.workerman.Unlock()
		err = f.cancel(t.Assignee, t.ID)
		if err != nil {
			log.Print(err)
		}
	}

	for {
		cancel()
	}
}
