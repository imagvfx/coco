package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func order(args []string) {
	fset := flag.NewFlagSet("order", flag.ExitOnError)
	fset.Parse(args)
	fargs := fset.Args()
	if len(fargs) == 0 {
		log.Fatal("need a json file to order")
	}

	orderfile := fargs[0]
	f, err := os.Open(orderfile)
	if err != nil {
		log.Fatal(err)
	}

	// send the job to farm
	cli := &http.Client{}
	addr := os.Getenv("COCO_ADDR")
	if addr == "" {
		addr = "localhost:8282"
	}
	req, err := http.NewRequest("POST", "http://"+addr+"/api/order", f)
	if err != nil {
		log.Fatal(err)
	}

	// check the response
	resp, err := cli.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("%s", b)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}
