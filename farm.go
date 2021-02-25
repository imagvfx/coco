package coco

import (
	"container/heap"
	"fmt"
)

// Farm manages jobs and workers.
type Farm struct {
	jobman    *JobManager
	workerman *WorkerManager
}

// NewFarm creates a new Farm.
func NewFarm(jobman *JobManager, workerman *WorkerManager) *Farm {
	return &Farm{
		jobman:    jobman,
		workerman: workerman,
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
	if w.task != "" {
		f.jobman.Lock()
		defer f.jobman.Unlock()
		t, err := f.jobman.GetTask(w.task)
		if err != nil {
			return err
		}
		j := t.Job
		j.Lock()
		defer j.Unlock()
		err = t.Unassign(w)
		if err != nil {
			return err
		}
		t.SetStatus(TaskFailed)
	}
	f.workerman.Bye(w.addr)
	return nil
}

// Done indicates the worker finished the requested task.
func (f *Farm) Done(addr, task string) error {
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	j := t.Job
	j.Lock()
	defer j.Unlock()
	t.SetStatus(TaskDone)
	if j.blocked {
		peek := j.Peek()
		if peek != nil {
			t.Job.blocked = false
			heap.Push(f.jobman.jobs, j)
		}
	}
	// TODO: need to verify the worker
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		return fmt.Errorf("unknown worker: %v", addr)
	}
	err = t.Unassign(w)
	if err != nil {
		return err
	}
	f.workerman.Ready(w)
	return nil
}

// Failed indicates the worker failed to finish the requested task.
func (f *Farm) Failed(addr, task string) error {
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t, err := f.jobman.GetTask(task)
	if err != nil {
		return err
	}
	j := t.Job
	j.Lock()
	defer j.Unlock()
	ok := f.jobman.PushTaskForRetry(t)
	if !ok {
		t.SetStatus(TaskFailed)
	}
	// TODO: need to verify the worker
	f.workerman.Lock()
	defer f.workerman.Unlock()
	w := f.workerman.FindByAddr(addr)
	if w == nil {
		return fmt.Errorf("unknown worker: %v", addr)
	}
	err = t.Unassign(w)
	if err != nil {
		return err
	}
	f.workerman.Ready(w)
	return nil
}
