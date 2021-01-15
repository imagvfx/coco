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
	worker := newWorkerManager()

	farm := newFarmServer("localhost:8284", job, worker)
	go farm.Listen()

	go matching(job, worker)
	go cancelling(job, worker)

	api := &apiHandler{
		jobman: job,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", api.handleOrder)
	mux.HandleFunc("/api/cancel", api.handleCancel)
	mux.HandleFunc("/api/retry", api.handleRetry)
	mux.HandleFunc("/api/job", api.handleJob)

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

		workers := workerman.IdleWorkers()
		if len(workers) == 0 {
			return
		}
		for _, w := range workers {
			var t *Task
			for {
				// find next task.
				t = jobman.PopTask()
				if t == nil {
					return
				}
				// TODO: what if the job is deleted already?
				t.job.Lock()
				cancel := len(t.Commands) == 0 || t.Status() == TaskCancelled
				t.job.Unlock()
				if cancel {
					continue
				}
				break
			}
			err := workerman.sendTask(w, t)
			if err != nil {
				// TODO: currently, it skips the commands if failed to communicate with the worker.
				// is it right decision?
				log.Print(err)
			}
			// worker got the task.
			t.job.Lock()
			t.SetStatus(TaskRunning)
			t.job.Unlock()
		}
	}

	go func() {
		for {
			match()
		}
	}()
}

func cancelling(jobman *jobManager, workerman *workerManager) {
	cancel := func() {
		t := <-jobman.CancelTaskCh
		w, ok := workerman.assignee[t.id]
		if !ok {
			return
		}
		err := workerman.sendCancelTask(w, t)
		if err != nil {
			log.Print(err)
		}
	}

	go func() {
		for {
			cancel()
		}
	}()
}
