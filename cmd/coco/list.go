package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type ListJob struct {
	ID     int
	Status string
	Title  string
}

func cutOrFill(s string, n int, fillLeft bool) string {
	if n < 0 {
		// invalid input
		return s
	}
	if len(s) > n {
		return s[:n]
	}
	spaces := strings.Repeat(" ", n-len(s))
	if fillLeft {
		return spaces + s
	}
	return s + spaces
}

func list(args []string) {
	fset := flag.NewFlagSet("list", flag.ExitOnError)
	var tag string
	fset.StringVar(&tag, "tag", "", "list jobs that having the tag.")
	fset.Parse(args)

	addr := os.Getenv("COCO_ADDR")
	if addr == "" {
		addr = "localhost:8282"
	}

	data := url.Values{}
	data.Add("tag", tag)

	// check the response
	resp, err := http.PostForm("http://"+addr+"/api/list", data)
	if err != nil {
		log.Fatal(err)
	}

	var list []ListJob
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	if err != nil {
		log.Fatal(err)
	}
	if len(list) == 0 {
		fmt.Println("no job to show")
	}
	for _, j := range list {
		fmt.Printf("[%v] %v - %v\n", j.ID, cutOrFill(j.Status, 7, false), j.Title)
	}
}
