package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

var JobManager = newJobManager()

func handleRoot(w http.ResponseWriter, r *http.Request) {}

func main() {
	var addr string
	defaultAddr := os.Getenv("COCO_ADDR")
	if defaultAddr == "" {
		defaultAddr = "localhost:8282"
	}
	flag.StringVar(&addr, "addr", defaultAddr, "address to bind")
	flag.Parse()

	JobManager.Start()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/order", handleAPIOrder)

	log.Fatal(http.ListenAndServe(addr, mux))
}
