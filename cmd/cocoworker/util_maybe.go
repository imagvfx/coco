// +build

package main

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		log.Fatalf("maybe needs a argument for define probability [0-1]")
	}
	prob, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		log.Fatalf("probability should be float, got %v", args[0])
	}
	if prob < 0 {
		prob = 0
	}
	if prob > 1 {
		prob = 1
	}
	rand.Seed(time.Now().UnixNano())
	v := rand.Float64()
	if v > prob {
		os.Exit(1)
	}
	os.Exit(0)
}
