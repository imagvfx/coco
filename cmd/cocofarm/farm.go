package main

import (
	"container/heap"
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

// Ready will be called by workers through gRPC.
// It indicates the caller is idle and waiting for commands to run.
func (f *farmServer) Ready(ctx context.Context, in *pb.ReadyRequest) (*pb.ReadyResponse, error) {
	log.Printf("ready: %v", in.Addr)
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		w = &Worker{addr: in.Addr, status: WorkerReady}
		err := f.workerman.Add(w)
		if err != nil {
			return &pb.ReadyResponse{}, err
		}
	}
	f.workerman.Ready(w)
	return &pb.ReadyResponse{}, nil
}

// Bye will be called by workers through gRPC.
// It indicates the caller is idle and waiting for commands to run.
func (f *farmServer) Bye(ctx context.Context, in *pb.ByeRequest) (*pb.ByeResponse, error) {
	log.Printf("bye: %v", in.Addr)
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		return &pb.ByeResponse{}, fmt.Errorf("unknown worker: %v", in.Addr)
	}
	if w.task != "" {
		f.jobman.Lock()
		defer f.jobman.Unlock()
		err := f.jobman.unassign(w.task, w)
		if err != nil {
			// It's farm's error, no need to send back the error to worker.
			log.Print(err)
		} else {
			t := f.jobman.task[w.task]
			t.job.Lock()
			defer t.job.Unlock()
			t.SetStatus(TaskFailed)
		}
	}
	f.workerman.Bye(w.addr)
	return &pb.ByeResponse{}, nil
}

// Done will be called by workers through gRPC.
// It indicates the caller finished the requested task.
func (f *farmServer) Done(ctx context.Context, in *pb.DoneRequest) (*pb.DoneResponse, error) {
	log.Printf("done: %v %v", in.Addr, in.TaskId)
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t := f.jobman.task[TaskID(in.TaskId)]
	t.job.Lock()
	defer t.job.Unlock()
	t.SetStatus(TaskDone)
	if f.jobman.jobBlocked[t.job.id] {
		peek := f.jobman.job[t.job.id].Peek()
		if peek != nil {
			delete(f.jobman.jobBlocked, t.job.id)
			heap.Push(f.jobman.jobs, t.job)
		}
	}
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		return &pb.DoneResponse{}, fmt.Errorf("unknown worker: %v", in.Addr)
	}
	err := f.jobman.unassign(TaskID(in.TaskId), w)
	if err != nil {
		return &pb.DoneResponse{}, err
	}
	f.workerman.Ready(w)
	return &pb.DoneResponse{}, nil
}

// Done will be called by workers through gRPC.
// It indicates the caller finished the requested task.
func (f *farmServer) Failed(ctx context.Context, in *pb.FailedRequest) (*pb.FailedResponse, error) {
	log.Printf("failed: %v %v", in.Addr, in.TaskId)
	f.jobman.Lock()
	defer f.jobman.Unlock()
	t := f.jobman.task[TaskID(in.TaskId)]
	j := t.job
	j.Lock()
	defer j.Unlock()
	ok := f.jobman.PushTaskForRetry(t)
	if !ok {
		t.SetStatus(TaskFailed)
	}
	// TODO: need to verify the worker
	w := f.workerman.FindByAddr(in.Addr)
	if w == nil {
		return &pb.FailedResponse{}, fmt.Errorf("unknown worker: %v", in.Addr)
	}
	err := f.jobman.unassign(TaskID(in.TaskId), w)
	if err != nil {
		return &pb.FailedResponse{}, err
	}
	f.workerman.Ready(w)
	return &pb.FailedResponse{}, nil
}
