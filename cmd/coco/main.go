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

	var err error

	subcmd := args[0]
	switch subcmd {
	case "order":
		err = order(args[1:])
	default:
		log.Fatalf("unknow subcommand: %s", subcmd)
	}

	if err != nil {
		log.Fatalf("%v: %v", subcmd, err)
	}
}
