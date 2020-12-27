package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type apiHandler struct {
	jobManager *jobManager
}

func (h *apiHandler) handleOrder(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &Job{}
	err := dec.Decode(j)
	if err != nil {
		log.Print(err)
	}

	err = h.jobManager.Add(j)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
	// TODO: return the job id
}
