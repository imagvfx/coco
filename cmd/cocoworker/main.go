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

	// farmConn is a connection to the farm, that is underlying of
	// farmClient.
	farmConn *grpc.ClientConn

	// farmClient is a connection to the farm, which defines
	// communication interfaces for talking with farmServer.
	farmClient pb.FarmClient

	// maxTry is maximum number of trys when communication with the farm.
	maxTry int

	// retryDelaySec is retry delay in seconds when a communication with the farm failed.
	retryDelaySec int

	// sleepingFn is a function that is called after a waking up.
	// It is used to save the last message to the farm.
	sleepingFn func() error

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

// newWorkerServer creates a new server.
func newWorkerServer(addr, farm string) *server {
	s := &server{addr: addr, farm: farm, maxTry: 5, retryDelaySec: 5}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var err error
	s.farmConn, err = grpc.DialContext(ctx, farm, grpc.WithInsecure())
	if err != nil {
		// actually, I don't think we can see this message un.
		log.Print("failed to connect to farm: %v", err)
	}
	s.farmClient = pb.NewFarmClient(s.farmConn)
	return s
}

// Close closes the server.
func (s *server) Close() error {
	s.Lock()
	defer s.Unlock()
	if s.farmConn != nil {
		err := s.farmConn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// Sleep make the server stop sending a message to the farm.
// Unreached message is kept, and sended when the server wakes up.
func (s *server) Sleep(fn func() error) {
	s.Lock()
	defer s.Unlock()
	log.Print("worker fall asleep")
	s.sleepingFn = fn
}

// IsSleeping checks whether the server is in sleeping.
func (s *server) IsSleeping() bool {
	s.Lock()
	defer s.Unlock()
	return s.sleepingFn != nil
}

// WakeUp tries to wake up the server.
// It wakes when the unreached message reaches this time.
// It will not wake up, if the message is still not reached.
// For the case, farm should try again.
func (s *server) WakeUp() {
	s.Lock()
	defer s.Unlock()
	// Don't know why exactly, but if the farm has restarted, the
	// connection is reattached slowly. Recreate the connection for better speed.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var err error
	s.farmConn, err = grpc.DialContext(ctx, s.farm, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to farm: %v", err)
	}
	s.farmClient = pb.NewFarmClient(s.farmConn)

	var send func() error
	if s.sleepingFn != nil {
		send = s.sleepingFn
	} else if s.runningTaskID == "" {
		send = s.SendReady
	}
	if send == nil {
		return
	}

	err = send()
	if err != nil {
		log.Print("the worker didn't wake up as the message wasn't reached to the farm")
		return
	}
	s.sleepingFn = nil
}

// Ping got a ping from a server and try to wake up the server if it is sleeping.
func (s *server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	s.Lock()
	defer s.Unlock()
	go s.WakeUp()
	return &pb.PingResponse{TaskId: s.runningTaskID}, nil
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
	if s.IsSleeping() {
		go s.WakeUp()
		return &pb.RunResponse{}, fmt.Errorf("need to wake up first: %v", s.addr)
	}
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
				err := s.SendFailed(in.Id)
				if err != nil {
					s.Sleep(func() error { return s.SendFailed(in.Id) })
				}
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
		err := s.SendDone(tid)
		if err != nil {
			s.Sleep(func() error { return s.SendDone(tid) })
		}
	}()
	return &pb.RunResponse{}, nil
}

func (s *server) Cancel(ctx context.Context, in *pb.CancelRequest) (*pb.CancelResponse, error) {
	if s.IsSleeping() {
		go s.WakeUp()
		return &pb.CancelResponse{}, fmt.Errorf("need to wake up first: %v", s.addr)
	}
	log.Printf("cancel: %v", in.Id)
	err := s.Abort(in.Id)
	if err != nil {
		return &pb.CancelResponse{}, err
	}
	go func() {
		err := s.SendReady()
		if err != nil {
			s.Sleep(func() error { return s.SendReady() })
		}
	}()
	return &pb.CancelResponse{}, nil
}

// SendReady sends a ready message until the farm has recieved it.
// It tries until reached to maxTry, then return error.
func (s *server) SendReady() error {
	var err error
	for i := 0; i < s.maxTry; i++ {
		if i != 0 {
			log.Printf("failed to send that I'm ready. will try %v seconds later...", s.retryDelaySec)
			time.Sleep(time.Duration(s.retryDelaySec) * time.Second)
		}
		err = s.sendReady()
		if err == nil {
			return nil
		}
		log.Print(err)
	}
	return err
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

// SendDone sends a done message until the farm has recieved it.
// It blocks the next operations as syncronization with the farm is most important.
func (s *server) SendDone(taskID string) error {
	var err error
	for i := 0; i < s.maxTry; i++ {
		if i != 0 {
			log.Printf("failed to send the commands done. will try %v seconds later...", s.retryDelaySec)
			time.Sleep(time.Duration(s.retryDelaySec) * time.Second)
		}
		err = s.sendDone(taskID)
		if err == nil {
			return nil
		}
		log.Print(err)
	}
	return err
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

// SendFailed sends a failed message until the farm has recieved it.
// It blocks the next operations as syncronization with the farm is most important.
func (s *server) SendFailed(taskID string) error {
	var err error
	for i := 0; i < s.maxTry; i++ {
		if i != 0 {
			log.Printf("failed to send the commands failed. will try %v seconds later...", s.retryDelaySec)
			time.Sleep(time.Duration(s.retryDelaySec) * time.Second)
		}
		err = s.sendFailed(taskID)
		if err == nil {
			break
		}
		log.Print(err)
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

// SendBye let the farm knows the worker is quitting. Unlike other Send methods,
// it cannot wait, also it's relatively safe when the message has not reached.
// So it will just return an error if there is a communication problem.
func (s *server) SendBye() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.ByeRequest{Addr: s.addr}
	_, err := s.farmClient.Bye(ctx, req)
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

	srv := newWorkerServer(addr, farm)

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
		err := srv.SendBye()
		if err != nil {
			log.Printf("disconnection message was not reached to the farm: %v", err)
		}
		err = srv.Close()
		if err != nil {
			log.Printf("couldn't close the server gracefully: %v", err)
		}
		os.Exit(1)
	}()

	select {} // prevent exit
}
