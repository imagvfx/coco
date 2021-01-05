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
	jobman    *jobManager
	workerman *workerManager
}

// newFarmServer creates a new farmServer.
func newFarmServer(addr string, jobman *jobManager, workerman *workerManager) *farmServer {
	return &farmServer{
		addr:      addr,
		jobman:    jobman,
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
// It indicates the caller is idle and waiting for commands to run.
func (f *farmServer) Waiting(ctx context.Context, in *pb.WaitingRequest) (*pb.WaitingResponse, error) {
	log.Printf("received: %v", in.Addr)
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		w = &Worker{addr: in.Addr, status: WorkerIdle}
		err := f.workerman.Add(w)
		if err != nil {
			log.Print(err)
		}
	}
	go f.workerman.Waiting(w)
	return &pb.WaitingResponse{}, nil
}

// Done will be called by workers through gRPC.
// It indicates the caller finished the requested task.
func (f *farmServer) Done(ctx context.Context, in *pb.DoneRequest) (*pb.DoneResponse, error) {
	log.Printf("done: %v %v", in.Addr, in.TaskId)
	t := f.jobman.GetTask(in.TaskId)
	t.SetStatus(TaskDone)
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		return &pb.DoneResponse{}, fmt.Errorf("unknown worker: %v", in.Addr)
	}
	err := f.workerman.DoneBy(in.TaskId, w)
	if err != nil {
		return &pb.DoneResponse{}, err
	}
	go f.workerman.Waiting(w)
	return &pb.DoneResponse{}, nil
}
