package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWorkerServer
	sync.Mutex

	// addr is the server's listen addr.
	addr string

	// farm is where the server is try to send grpc requests.
	farm string

	// farmClient is a connection to the farm, which is
	// needed for talking, not for listening.
	farmClient pb.FarmClient

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
	s.Lock()
	defer s.Unlock()
	if s.runningTaskID != "" {
		return &pb.RunResponse{}, fmt.Errorf("I'm busy: %v", s.addr)
	}
	s.runningTaskID = in.Id
	s.aborted = false
	// run commands are usually taking long time,
	// detach it with a goroutine.
	go func() {
		for _, cmd := range in.Cmds {
			s.Lock()
			if s.aborted {
				return
			}
			c := exec.Command(cmd.Args[0], cmd.Args[1:]...)
			s.cmd = c // we might have to cancel it.
			s.Unlock()
			out, err := c.CombinedOutput()
			if err != nil {
				log.Print(err)
				s.Lock()
				s.runningTaskID = ""
				s.aborted = false
				s.Unlock()
				s.sendFailed(in.Id)
				return
			}
			log.Print(string(out))
		}
		// finished running the commands. let the farm knows it.
		s.Lock()
		tid := s.runningTaskID
		s.runningTaskID = ""
		s.aborted = false
		s.Unlock()
		err := s.sendDone(tid)
		if err != nil {
			log.Print(err)
		}
	}()
	return &pb.RunResponse{}, nil
}

func (s *server) Cancel(ctx context.Context, in *pb.CancelRequest) (*pb.CancelResponse, error) {
	log.Printf("cancel: %v", in.Id)
	err := s.Abort(in.Id)
	if err != nil {
		return &pb.CancelResponse{}, err
	}
	go s.sendReady()
	return &pb.CancelResponse{}, nil
}

func (s *server) sendReady() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.ReadyRequest{Addr: s.addr}
	_, err := s.farmClient.Ready(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) sendDone(taskID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.DoneRequest{Addr: s.addr, TaskId: taskID}
	_, err := s.farmClient.Done(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) sendFailed(taskID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.FailedRequest{Addr: s.addr, TaskId: taskID}
	_, err := s.farmClient.Failed(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) handshakeWithFarm(n int) {
	for i := 0; i < n; i++ {
		err := s.sendReady()
		if err == nil {
			return
		}
		// failed for some reason. try again sometime later.
		log.Print(err)
		time.Sleep(30 * time.Second)
	}
	log.Fatalf("cannot find the farm: %v", s.farm)
}

func (s *server) bye() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.ByeRequest{Addr: s.addr}
	_, err := s.farmClient.Bye(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var (
		addr string
		farm string
	)
	flag.StringVar(&addr, "addr", "localhost:8283", "address to bind")
	defaultFarm := os.Getenv("COCO_FARM")
	if defaultFarm == "" {
		defaultFarm = "localhost:8284"
	}
	flag.StringVar(&farm, "farm", defaultFarm, "farm address")
	flag.Parse()

	conn, err := grpc.Dial(farm, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		log.Fatalf("failed to connect to farm: %v", err)
	}
	defer conn.Close()
	cli := pb.NewFarmClient(conn)
	srv := &server{addr: addr, farm: farm, farmClient: cli}

	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterWorkerServer(s, srv)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	srv.handshakeWithFarm(5)

	killed := make(chan os.Signal, 1)
	signal.Notify(killed, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-killed
		err := srv.bye()
		if err != nil {
			log.Fatalf("disconnection message was not reached to the farm")
		}
		os.Exit(1)
	}()

	select {} // prevent exit
}
