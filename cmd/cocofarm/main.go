package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

var JobManager = &jobManager{}

func handleRoot(w http.ResponseWriter, r *http.Request) {}

func main() {
	var addr string
	defaultAddr := os.Getenv("COCO_ADDR")
	if defaultAddr == "" {
		defaultAddr = "localhost:8282"
	}
	flag.StringVar(&addr, "addr", defaultAddr, "address to bind")
	flag.Parse()

	// grpc test
	conn, err := grpc.Dial("localhost:8283", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.Run(ctx, &pb.Commands{
		Cmds: []*pb.Commands_C{
			{Args: []string{"ls"}},
			{Args: []string{"ls", "-al"}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", handleAPIOrder)

	log.Fatal(http.ListenAndServe(addr, mux))
}
