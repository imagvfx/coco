package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
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
	go listenWorker(worker)

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

type server struct {
	pb.UnimplementedFarmServer
	workerman *workerManager
}

func listenWorker(workerman *workerManager) {
	lis, err := net.Listen("tcp", "localhost:8284")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFarmServer(s, &server{workerman: workerman})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failted to serve: %v", err)
	}
}

func (s *server) Waiting(ctx context.Context, in *pb.Here) (*pb.Empty, error) {
	log.Printf("received: %v", in.Addr)
	addr := in.Addr
	found := false
	for _, w := range s.workerman.workers {
		if addr == w.addr {
			s.workerman.SetStatus(w, WorkerIdle)
			found = true
			break
		}
	}
	if !found {
		err := s.workerman.Add(&Worker{addr: addr, status: WorkerIdle})
		if err != nil {
			log.Print(err)
		}
	}
	return &pb.Empty{}, nil
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
			var cmds []coco.Command
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
