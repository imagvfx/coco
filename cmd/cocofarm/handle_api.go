package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/imagvfx/coco"
)

func handleAPIOrder(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &coco.Job{}
	err := dec.Decode(j)
	if err != nil {
		log.Print(err)
	}

	err = JobManager.Add(j)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
	// TODO: return the job id
}
