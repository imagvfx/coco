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

	io.WriteString(w, fmt.Sprintf("%v", j))

	// TODO: add job to job stack
	// TODO: return the job id
}
