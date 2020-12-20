package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/imagvfx/coco"
)

func handleAPIOrder(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &coco.Job{}
	dec.Decode(j)

	JobManager.Add(j)
	io.WriteString(w, fmt.Sprintf("%v", JobManager.Jobs()))
	// TODO: return the job id
}
