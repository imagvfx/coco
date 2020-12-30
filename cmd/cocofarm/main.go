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

	worker := newWorkerManager()

	farm := newFarmServer("localhost:8284", worker)
	go farm.Listen()

	job := newJobManager()
	go matching(job, worker)
	go cancelling(job, worker)

	api := &apiHandler{
		jobman: job,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", api.handleOrder)
	mux.HandleFunc("/api/cancel", api.handleCancel)

	log.Fatal(http.ListenAndServe(addr, mux))
}

func matching(jobman *jobManager, workerman *workerManager) {
	match := func() {
		t := jobman.NextTask()
		if t == nil {
			time.Sleep(time.Second)
			return
		}
		if len(t.Commands) == 0 {
			// noting to do
			return
		}
		w := <-workerman.WorkerCh
		jobman.Lock()
		defer jobman.Unlock()
		if t.status == TaskCancelled {
			// the task is cancelled while it is waiting a worker.
			go workerman.Waiting(w)
			return
		}
		err := workerman.sendTask(w, t)
		if err != nil {
			// TODO: currently, it skips the commands if failed to communicate with the worker.
			// is it right decision?
			log.Print(err)
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
