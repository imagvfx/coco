package main

import (
	"context"
	"log"
	"net"
	"os/exec"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWorkerServer
}

func (s *server) Run(ctx context.Context, in *pb.Commands) (*pb.Empty, error) {
	cmds := in.Cmds
	log.Printf("received: %v", in.Cmds)
	if len(cmds) == 0 {
		return &pb.Empty{}, nil
	}
	for _, cmd := range cmds {
		c := exec.Command(cmd.Args[0], cmd.Args[1:]...)
		out, err := c.CombinedOutput()
		if err != nil {
			log.Print(err)
		}
		log.Print(string(out))
	}
	return &pb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", "localhost:8283")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWorkerServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failted to serve: %v", err)
	}
}
