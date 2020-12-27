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

	api := &apiHandler{
		jobManager: job,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", api.handleOrder)

	log.Fatal(http.ListenAndServe(addr, mux))
}

func matching(jobman *jobManager, workerman *workerManager) {
	match := func() {
		workers := workerman.idleWorkers()
		if len(workers) == 0 {
			log.Print("no worker yet")
			return
		}

		for i := 0; i < len(workers); i++ {
			w := workers[i]
			var cmds []Command
			for {
				t := jobman.NextTask()
				if t == nil {
					// no more task to do
					return
				}
				if len(t.Commands) == 0 {
					continue
				}
				cmds = t.Commands
				break
			}
			err := workerman.sendCommands(w, cmds)
			if err != nil {
				// TODO: currently, it skips the commands if failed to communicate with the worker.
				// is it right decision?
				log.Print(err)
			}
		}
	}

	go func() {
		for {
			time.Sleep(time.Second)
			log.Print("matching...")
			match()
		}
	}()
}
