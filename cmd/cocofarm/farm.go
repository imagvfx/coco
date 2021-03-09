package main

import (
	"context"
	"log"
	"net"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

// farmServer is a gRPC server that listens workers and manages their status.
type farmServer struct {
	pb.UnimplementedFarmServer
	addr string
	farm *coco.Farm
}

// newFarmServer creates a new farmServer.
func newFarmServer(addr string, farm *coco.Farm) *farmServer {
	return &farmServer{
		addr: addr,
		farm: farm,
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

// Ready will be called by workers through gRPC.
// It indicates the caller is idle and waiting for commands to run.
func (f *farmServer) Ready(ctx context.Context, in *pb.ReadyRequest) (*pb.ReadyResponse, error) {
	log.Printf("ready: %v", in.Addr)
	// TODO: need to verify the worker
	err := f.farm.Ready(in.Addr)
	if err != nil {
		// It's internal error. Don't send the error to the worker.
		log.Printf("worker message didn't handled well: %v ready: %v", in.Addr, err)
	}
	return &pb.ReadyResponse{}, nil
}

// Bye will be called by workers through gRPC.
// It indicates the caller is idle and waiting for commands to run.
func (f *farmServer) Bye(ctx context.Context, in *pb.ByeRequest) (*pb.ByeResponse, error) {
	log.Printf("bye: %v", in.Addr)
	err := f.farm.Bye(in.Addr)
	if err != nil {
		// It's internal error. Don't send the error to the worker.
		log.Printf("worker message didn't handled well: %v bye: %v", in.Addr, err)
	}
	return &pb.ByeResponse{}, nil
}

// Done will be called by workers through gRPC.
// It indicates the caller finished the requested task.
func (f *farmServer) Done(ctx context.Context, in *pb.DoneRequest) (*pb.DoneResponse, error) {
	log.Printf("done: %v %v", in.Addr, in.TaskId)
	err := f.farm.Done(in.Addr, in.TaskId)
	if err != nil {
		// It's internal error. Don't send the error to the worker.
		log.Printf("worker message didn't handled well: %v done: %v", in.Addr, err)
	}
	return &pb.DoneResponse{}, nil
}

// Done will be called by workers through gRPC.
// It indicates the caller finished the requested task.
func (f *farmServer) Failed(ctx context.Context, in *pb.FailedRequest) (*pb.FailedResponse, error) {
	log.Printf("failed: %v %v", in.Addr, in.TaskId)
	err := f.farm.Failed(in.Addr, in.TaskId)
	if err != nil {
		// It's internal error. Don't send the error to the worker.
		log.Printf("worker message didn't handled well: %v failed: %v", in.Addr, err)
	}
	return &pb.FailedResponse{}, nil
}
