package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWorkerServer
	sync.Mutex

	// runningTaskID is Task's id that is currently processed.
	// when the worker is in idle, this will be empty string
	runningTaskID string

	// cmd is a command that is currently running.
	// when the worker is in idle, this will be a nil.
	cmd *exec.Cmd

	// aborted indicates that the latest run is aborted,
	// so subsequent commands should not be launched.
	aborted bool
}

func (s *server) Start(tid string, cmds [][]string) {
	s.Lock()
	s.runningTaskID = tid
	s.aborted = false
	s.Unlock()
	// run commands are usually taking long time,
	// detach it with a goroutine.
	go func() {
		for _, cmd := range cmds {
			s.Lock()
			aborted := s.aborted
			s.Unlock()
			if aborted {
				return
			}
			c := exec.Command(cmd[0], cmd[1:]...)
			s.Lock()
			s.cmd = c // we might have to cancel it.
			s.Unlock()
			out, err := c.CombinedOutput()
			if err != nil {
				sendFailed("localhost:8284", "localhost:8283", tid)
				return
			}
			log.Print(string(out))
		}
		// finished running the commands. let the farm knows it.
		s.Lock()
		tid := s.runningTaskID
		s.Unlock()
		err := sendDone("localhost:8284", "localhost:8283", tid)
		if err != nil {
			log.Print(err)
		}
	}()
}

func (s *server) Abort(id string) error {
	s.Lock()
	defer s.Unlock()
	if id != s.runningTaskID {
		return fmt.Errorf("%v is not a running task", id)
	}
	s.runningTaskID = ""
	s.aborted = true
	err := s.cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}
	s.cmd = nil
	return nil
}

func (s *server) Run(ctx context.Context, in *pb.RunRequest) (*pb.RunResponse, error) {
	log.Printf("run: %v %v", in.Id, in.Cmds)
	cmds := make([][]string, len(in.Cmds))
	for i, cmd := range in.Cmds {
		cmds[i] = cmd.Args
	}
	s.Start(in.Id, cmds)
	return &pb.RunResponse{}, nil
}

func (s *server) Cancel(ctx context.Context, in *pb.CancelRequest) (*pb.CancelResponse, error) {
	log.Printf("cancel: %v", in.Id)
	err := s.Abort(in.Id)
	if err != nil {
		return &pb.CancelResponse{}, err
	}
	go sendWaiting("localhost:8284", "localhost:8283")
	return &pb.CancelResponse{}, nil
}

func sendWaiting(farm, addr string) error {
	conn, err := grpc.Dial(farm, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewFarmClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.WaitingRequest{Addr: addr}
	_, err = c.Waiting(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func sendDone(farm, addr string, taskID string) error {
	conn, err := grpc.Dial(farm, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewFarmClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.DoneRequest{Addr: addr, TaskId: taskID}
	_, err = c.Done(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func sendFailed(farm, addr string, taskID string) error {
	conn, err := grpc.Dial(farm, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewFarmClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.FailedRequest{Addr: addr, TaskId: taskID}
	_, err = c.Failed(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func handshakeWithFarm(n int) {
	for i := 0; i < n; i++ {
		err := sendWaiting("localhost:8284", "localhost:8283")
		if err == nil {
			return
		}
		// failed for some reason. try again sometime later.
		log.Print(err)
		time.Sleep(30 * time.Second)
	}
	log.Fatalf("cannot find the farm")
}

func main() {
	go func() {
		lis, err := net.Listen("tcp", "localhost:8283")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterWorkerServer(s, &server{})
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failted to serve: %v", err)
		}
	}()

	handshakeWithFarm(5)

	select {} // prevent exit
}
