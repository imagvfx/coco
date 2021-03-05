package coco

import (
	"container/heap"
	"fmt"
	"log"
	"time"
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
		// TODO: what if the job is deleted already?
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
		err := f.workerman.SendTask(w, t)
		if err != nil {
			// Failed to communicate with the worker.
			log.Printf("failed to send task to a worker: %v", err)
			f.jobman.PushTask(t)
			return
		}
		t.Assign(w)
		// worker got the task.
		t.SetStatus(TaskRunning)
	}

	for {
		match()
	}
}

func (f *Farm) Canceling() {
	cancel := func() {
		t := <-f.jobman.CancelTaskCh
		f.jobman.Lock()
		defer f.jobman.Unlock()
		t, err := f.jobman.GetTask(t.ID())
		if err != nil {
			log.Printf("failed to cancel task: %v", err)
			return
		}
		f.workerman.Lock()
		defer f.workerman.Unlock()
		w := t.Assignee
		if w == nil {
			return
		}
		err = f.workerman.SendCancelTask(w, t)
		if err != nil {
			log.Print(err)
			return
		}
		t.Unassign(w)
	}

	for {
		cancel()
	}
}
