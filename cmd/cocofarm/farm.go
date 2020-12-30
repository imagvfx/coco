package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

// farmServer is a gRPC server that listens workers and manages their status.
type farmServer struct {
	pb.UnimplementedFarmServer
	addr      string
	workerman *workerManager
}

// newFarmServer creates a new farmServer.
func newFarmServer(addr string, workerman *workerManager) *farmServer {
	return &farmServer{
		addr:      addr,
		workerman: workerman,
	}
}

// Listen listens to workers, to change their status.
func (f *farmServer) Listen() {
	lis, err := net.Listen("tcp", f.addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFarmServer(s, f)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failted to serve: %v", err)
	}
}

// Waiting will be called by workers through gRPC.
// Workers will call this to indicate they are idle and waiting for commands to run.
func (f *farmServer) Waiting(ctx context.Context, in *pb.Here) (*pb.Empty, error) {
	log.Printf("received: %v", in.Addr)
	found := false
	for _, w := range f.workerman.workers {
		if in.Addr == w.addr {
			f.workerman.SetStatus(w, WorkerIdle)
			found = true
			break
		}
	}
	w := &Worker{addr: in.Addr, status: WorkerIdle}
	if !found {
		err := f.workerman.Add(w)
		if err != nil {
			log.Print(err)
		}
	}
	go f.workerman.Waiting(w)
	return &pb.Empty{}, nil
}

// Done will be called by workers through gRPC.
// Workers will call this to indicate they are done with the requested task.
func (f *farmServer) Done(ctx context.Context, in *pb.DoneRequest) (*pb.Empty, error) {
	log.Printf("done: %v %v", in.Addr, in.TaskId)
	w, ok := f.workerman.assignee[in.TaskId]
	if !ok {
		found := false
		for _, w := range f.workerman.workers {
			if in.Addr == w.addr {
				f.workerman.SetStatus(w, WorkerIdle)
				found = true
				break
			}
		}
		if !found {
			return &pb.Empty{}, fmt.Errorf("worker not registered yet: %v", in.Addr)
		}
		return &pb.Empty{}, fmt.Errorf("task isn't assigned to any worker: %v", in.TaskId)
	}
	if w.addr != in.Addr {
		return &pb.Empty{}, fmt.Errorf("task is assigned from different worker: %v", in.TaskId)
	}

	delete(f.workerman.assignee, in.TaskId)
	go f.workerman.Waiting(w)
	return &pb.Empty{}, nil
}
