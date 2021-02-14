package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
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

	job := newJobManager()

	wgrps, err := loadWorkerGroupsFromConfig()
	if err != nil {
		log.Fatal(err)
	}
	worker := newWorkerManager(wgrps)

	farm := newFarmServer("localhost:8284", job, worker)
	go farm.Listen()

	go matching(job, worker)
	go canceling(job, worker)

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

func matching(jobman *jobManager, workerman *workerManager) {
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

		t := jobman.PopTask(workerman.ServableTargets())
		if t == nil {
			return
		}
		// TODO: what if the job is deleted already?
		j := t.job
		j.Lock()
		defer j.Unlock()
		cancel := len(t.Commands) == 0 || t.Status() == TaskFailed // eg. user canceled this task
		if cancel {
			return
		}
		w := workerman.Pop(t.job.Target)
		if w == nil {
			panic("at least one worker should be able to serve this tag")
		}
		err := workerman.sendTask(w, t)
		if err != nil {
			// Failed to communicate with the worker.
			log.Print(err)
			jobman.PushTask(t)
			return
		}
		jobman.Assign(t.id, w)
		// worker got the task.
		t.SetStatus(TaskRunning)
	}

	go func() {
		for {
			match()
		}
	}()
}

func canceling(jobman *jobManager, workerman *workerManager) {
	cancel := func() {
		t := <-jobman.CancelTaskCh
		jobman.Lock()
		defer jobman.Unlock()
		w, ok := jobman.assignee[t.id]
		if !ok {
			return
		}
		err := workerman.sendCancelTask(w, t)
		if err != nil {
			log.Print(err)
			return
		}
		jobman.unassign(t.id, w)
	}

	go func() {
		for {
			cancel()
		}
	}()
}
