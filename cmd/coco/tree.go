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

type TreeJob struct {
	ID    string
	Title string
	Root  *TreeTask

	line string
}

type TreeTask struct {
	Title    string
	Status   string
	Subtasks []*TreeTask

	line string
}

func fixJob(j *TreeJob) {
	title := j.Title
	if title == "" {
		title = "untitled"
	}
	if j.Root == nil {
		// it shouldn't, but not much to do here.
		return
	}
	n := fixTask(j.Root)
	j.line = fmt.Sprintf("[%v] %v (%v/%v)", j.ID, title, n, 1)
}

func fixTask(t *TreeTask) int {
	return fixTaskR(t, 0)
}

func fixTaskR(t *TreeTask, nthChild int) int {
	nDone := 0
	for i, subt := range t.Subtasks {
		done := fixTaskR(subt, i)
		nDone += done
	}
	if len(t.Subtasks) == 0 {
		t.line = fmt.Sprintf("- %v", t.Status)
	} else {
		t.line = fmt.Sprintf("+ %v (%v/%v)", t.Status, nDone, len(t.Subtasks))
	}
	if t.Title != "" {
		t.line += ": " + t.Title
	}

	if t.Status == "done" {
		return 1
	}
	return 0
}

func printJob(j *TreeJob) {
	fmt.Println(j.line)
	printTask(j.Root, 1)
}

func printTask(t *TreeTask, depth int) {
	if t == nil {
		fmt.Println("nil task")
		return
	}
	pre := strings.Repeat("\t", depth)
	fmt.Printf("%v%v\n", pre, t.line)
	for _, t := range t.Subtasks {
		printTask(t, depth+1)
	}
}

func tree(args []string) {
	fset := flag.NewFlagSet("tree", flag.ExitOnError)
	fset.Parse(args)
	fargs := fset.Args()
	if len(fargs) == 0 {
		log.Fatal("need a job id to get tree")
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
	resp, err := http.PostForm("http://"+addr+"/api/job", data)
	if err != nil {
		log.Fatal(err)
	}

	j := &TreeJob{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(j)
	if err != nil {
		log.Fatal(err)
	}

	fixJob(j)
	printJob(j)
}
