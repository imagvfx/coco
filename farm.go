package coco

import (
	"container/heap"
	"fmt"
	"log"
	"time"
)

// Farm manages jobs and workers.
type Farm struct {
	FarmService FarmService
	jobman      *JobManager
	workerman   *WorkerManager
}

// NewFarm creates a new Farm.
// TODO: wgrps to generic config.
func NewFarm(services Services, wgrps []*WorkerGroup) (*Farm, error) {
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
	tRunning := TaskRunning
	wRunning := WorkerRunning
	err := f.updateAssign(AssignUpdater{
		Task:         task,
		TaskStatus:   &tRunning,
		TaskAssignee: &worker,
		Worker:       worker,
		WorkerStatus: &wRunning,
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *Farm) updateAssign(a AssignUpdater) error {
	t, err := f.jobman.GetTask(a.Task)
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
	if a.TaskStatus != nil {
		t.setStatus(*a.TaskStatus)
	}
	if a.TaskRetry != nil {
		t.retry = *a.TaskRetry
	}
	w.task = a.Task
	if a.WorkerStatus != nil {
		w.status = *a.WorkerStatus
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
			if w.task.IsValid() {
				t, err := f.jobman.GetTask(w.task)
				if err != nil {
					log.Print(err)
					return
				}
				err = t.Update(TaskUpdater{
					Status:   ptrTaskStatus(TaskFailed),
					Assignee: ptrString(""),
				})
				if err != nil {
					log.Printf("couldn't update task: %v", w.task)
				}
			}
			err := w.Update(WorkerUpdater{
				Status: ptrWorkerStatus(WorkerNotFound),
				Task:   &w.task,
			})
			if err != nil {
				log.Print(err)
			}
			return
		}
		if !tid.IsValid() && !w.task.IsValid() {
			// invalid task id indicates that there isn't a task running.
			return
		}
		if tid == w.task {
			return
		}
		if w.task.IsValid() {
			log.Printf("worker is not running on expected task: %v", w.task)
			t, err := f.jobman.GetTask(w.task)
			if err != nil {
				log.Print(err)
				return
			}
			err = t.Update(TaskUpdater{
				Status:   ptrTaskStatus(TaskFailed),
				Assignee: ptrString(""),
			})
			if err != nil {
				log.Printf("couldn't update task: %v", w.task)
			}
		}
		if tid.IsValid() {
			log.Printf("worker is running on unexpected task: %v", tid)
			t, err := f.jobman.GetTask(tid)
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
	if w.task.IsValid() {
		f.jobman.Lock()
		defer f.jobman.Unlock()
		t, err := f.jobman.GetTask(w.task)
		if err != nil {
			return err
		}
		j := t.Job
		j.Lock()
		defer j.Unlock()
		err = f.updateAssign(AssignUpdater{
			Task:         t.ID,
			TaskStatus:   ptrTaskStatus(TaskFailed),
			TaskAssignee: ptrString(""),
			Worker:       addr,
			WorkerStatus: ptrWorkerStatus(WorkerNotFound),
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
	if t.Assignee != addr || w.task != task {
		// TODO: fail over
		return fmt.Errorf("unmatched task and worker information: %v.Assignee=%v, %v.task=%v", task, t.Assignee, addr, w.task)
	}
	err = f.updateAssign(AssignUpdater{
		Task:         t.ID,
		TaskStatus:   ptrTaskStatus(TaskDone),
		TaskAssignee: ptrString(""),
		Worker:       addr,
		WorkerStatus: ptrWorkerStatus(WorkerReady),
	})
	if err != nil {
		return err
	}
	heap.Push(f.jobman.jobs, j)
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
	if t.Assignee != addr || w.task != task {
		// TODO: fail over
		return fmt.Errorf("unmatched task and worker information: %v.Assignee=%v, %v.task=%v", task, t.Assignee, addr, w.task)
	}
	ts := TaskFailed
	retry := t.CanRetry()
	if retry {
		t.retry++
		ts = TaskWaiting
	}
	err = f.updateAssign(AssignUpdater{
		Task:         t.ID,
		TaskStatus:   &ts,
		TaskRetry:    &t.retry,
		TaskAssignee: ptrString(""),
		Worker:       addr,
		WorkerStatus: ptrWorkerStatus(WorkerReady),
	})
	if err != nil {
		return err
	}
	if retry {
		t.Push()
	}
	heap.Push(f.jobman.jobs, j)
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
	err = f.updateAssign(AssignUpdater{
		Task:         t.ID,
		TaskStatus:   ptrTaskStatus(TaskFailed),
		TaskAssignee: ptrString(""),
		Worker:       w.addr,
		WorkerStatus: ptrWorkerStatus(WorkerCooling),
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
