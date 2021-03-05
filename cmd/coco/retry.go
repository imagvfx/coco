package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func retry(args []string) {
	fset := flag.NewFlagSet("retry", flag.ExitOnError)
	fset.Parse(args)
	fargs := fset.Args()
	if len(fargs) == 0 {
		log.Fatal("need a job id to retry")
	}

	id := fargs[0]
	// request cancel the job to farm
	addr := os.Getenv("COCO_ADDR")
	if addr == "" {
		addr = "localhost:8282"
	}

	data := url.Values{}
	data.Add("id", id)

	// check the response
	resp, err := http.PostForm("http://"+addr+"/api/retry", data)
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
