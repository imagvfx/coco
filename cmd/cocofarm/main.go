package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/sqlite"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {}

func main() {
	var (
		addr   string
		dbpath string
	)
	defaultAddr := os.Getenv("COCO_ADDR")
	if defaultAddr == "" {
		defaultAddr = "localhost:8282"
	}
	flag.StringVar(&addr, "addr", defaultAddr, "address to bind")
	flag.StringVar(&dbpath, "db", "coco.db", "database path to be created/opened")
	flag.Parse()

	wgrps, err := loadWorkerGroupsFromConfig()
	if err != nil {
		log.Fatal(err)
	}

	dbCreated := false
	db, err := sqlite.Open(dbpath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatal(err)
		}
		db, err = sqlite.Create(dbpath)
		if err != nil {
			log.Fatal(err)
		}
		dbCreated = true
	}
	defer db.Close()

	js := sqlite.NewJobService(db)
	ws := sqlite.NewWorkerService(db)

	var job *coco.JobManager
	var worker *coco.WorkerManager
	if dbCreated {
		job = coco.NewJobManager(js)
		worker = coco.NewWorkerManager(ws, wgrps)
	} else {
		job, err = coco.RestoreJobManager(js)
		if err != nil {
			log.Fatal(err)
		}
		worker, err = coco.RestoreWorkerManager(ws, wgrps)
		if err != nil {
			log.Fatal(err)
		}
	}
	farm := coco.NewFarm(job, worker)

	go newFarmServer("localhost:8284", farm).Listen()

	if !dbCreated {
		farm.RefreshWorkers()
	}
	go farm.Matching()
	go farm.Canceling()
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
