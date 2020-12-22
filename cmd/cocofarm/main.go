package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/imagvfx/coco"
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
	err := sendCommands("localhost:8283", []coco.Command{
		{"ls"},
		{"ls", "-al"},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", handleAPIOrder)

	log.Fatal(http.ListenAndServe(addr, mux))
}
