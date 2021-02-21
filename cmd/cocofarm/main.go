package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/imagvfx/coco"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {}

func main() {
	var addr string
	defaultAddr := os.Getenv("COCO_ADDR")
	if defaultAddr == "" {
		defaultAddr = "localhost:8282"
	}
	flag.StringVar(&addr, "addr", defaultAddr, "address to bind")
	flag.Parse()

	job := coco.NewJobManager()

	wgrps, err := loadWorkerGroupsFromConfig()
	if err != nil {
		log.Fatal(err)
	}
	worker := coco.NewWorkerManager(wgrps)

	farm := newFarmServer("localhost:8284", coco.NewFarm(job, worker))
	go farm.Listen()

	go matching(job, worker)
	go canceling(job, worker)
	go checking(job, "jobman")
	go checking(worker, "workerman")

	api := &apiHandler{
		jobman: job,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", api.handleOrder)
	mux.HandleFunc("/api/cancel", api.handleCancel)
	mux.HandleFunc("/api/retry", api.handleRetry)
	mux.HandleFunc("/api/delete", api.handleDelete)
	mux.HandleFunc("/api/job", api.handleJob)
	mux.HandleFunc("/api/list", api.handleList)

	log.Fatal(http.ListenAndServe(addr, mux))
}

func matching(jobman *coco.JobManager, workerman *coco.WorkerManager) {
	match := func() {
		// ReadyCh gives faster matching loop when we are fortune.
		// There might have a chance that there was no job
		// when a worker sent a signal to ReadyCh.
		// Then the signal spent, no more signal will come
		// unless another worker sends a signal to the channel.
		// Below time.After case will helps us prevent deadlock
		// on matching loop in that case.
		select {
		case <-workerman.ReadyCh:
			break
		case <-time.After(time.Second):
			break
		}

		jobman.Lock()
		defer jobman.Unlock()
		workerman.Lock()
		defer workerman.Unlock()

		t := jobman.PopTask(workerman.ServableTargets())
		if t == nil {
			return
		}
		// TODO: what if the job is deleted already?
		j := t.Job
		j.Lock()
		defer j.Unlock()
		cancel := len(t.Commands) == 0 || t.Status() == coco.TaskFailed // eg. user canceled this task
		if cancel {
			return
		}
		w := workerman.Pop(t.Job.Target)
		if w == nil {
			panic("at least one worker should be able to serve this tag")
		}
		err := workerman.SendTask(w, t)
		if err != nil {
			// Failed to communicate with the worker.
			log.Printf("failed to send task to a worker: %v", err)
			jobman.PushTask(t)
			return
		}
		jobman.Assign(t.ID, w)
		// worker got the task.
		t.SetStatus(coco.TaskRunning)
	}

	go func() {
		for {
			match()
		}
	}()
}

func canceling(jobman *coco.JobManager, workerman *coco.WorkerManager) {
	cancel := func() {
		t := <-jobman.CancelTaskCh
		jobman.Lock()
		defer jobman.Unlock()
		w, ok := jobman.Assignee[t.ID]
		if !ok {
			return
		}
		workerman.Lock()
		defer workerman.Unlock()
		err := workerman.SendCancelTask(w, t)
		if err != nil {
			log.Print(err)
			return
		}
		jobman.Unassign(t.ID, w)
	}

	go func() {
		for {
			cancel()
		}
	}()
}

func checking(l Locker, label string) {
	done := make(chan bool)
	for {
		time.Sleep(time.Second)
		go func() {
			l.Lock()
			defer l.Unlock()
			done <- true
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			// Couldn't obtain the lock while significant time passed.
			log.Fatalf("deadlock: %v", label)
		}
	}

}
