package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("need a subcommand: [order]")
	}

	subcmd := args[0]
	switch subcmd {
	case "order":
		order(args[1:])
	default:
		log.Fatalf("unknown subcommand: %s", subcmd)
	}
}
