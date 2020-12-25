package main

import (
	"context"
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
	addr := in.Addr
	found := false
	for _, w := range f.workerman.workers {
		if addr == w.addr {
			f.workerman.SetStatus(w, WorkerIdle)
			found = true
			break
		}
	}
	if !found {
		err := f.workerman.Add(&Worker{addr: addr, status: WorkerIdle})
		if err != nil {
			log.Print(err)
		}
	}
	return &pb.Empty{}, nil
}
