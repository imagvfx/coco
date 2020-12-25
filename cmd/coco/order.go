package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/imagvfx/coco"
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
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	// validate the job data before sending it to farm
	err = json.Unmarshal(data, &coco.Job{})
	if err != nil {
		log.Fatal(err)
	}

	// send the job to farm
	cli := &http.Client{}
	addr := os.Getenv("COCO_ADDR")
	if addr == "" {
		addr = "localhost:8282"
	}
	req, err := http.NewRequest("POST", "http://"+addr+"/api/order", bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	// check the response
	resp, err := cli.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}
