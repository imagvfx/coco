package main

import (
	"context"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWorkerServer
}

func (s *server) Run(ctx context.Context, in *pb.Commands) (*pb.Empty, error) {
	log.Printf("received: %v", in.Cmds)
	go func() {
		cmds := in.Cmds
		for _, cmd := range cmds {
			c := exec.Command(cmd.Args[0], cmd.Args[1:]...)
			out, err := c.CombinedOutput()
			if err != nil {
				log.Print(err)
			}
			log.Print(string(out))
		}
		// finished running the commands. let the farm knows it.
		err := sendWaiting("localhost:8284", "localhost:8283")
		if err != nil {
			log.Print(err)
		}
	}()
	return &pb.Empty{}, nil
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

	pbHere := &pb.Here{Addr: addr}
	_, err = c.Waiting(ctx, pbHere)
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
