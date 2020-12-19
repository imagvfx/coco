package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func handleAPIUserJobAdd(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &Job{}
	dec.Decode(j)

	io.WriteString(w, fmt.Sprintf("%v", j))

	// TODO: add job to job stack
	// TODO: return the job id
}
