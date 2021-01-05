package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type apiHandler struct {
	jobman *jobManager
}

func (h *apiHandler) handleOrder(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &Job{}
	err := dec.Decode(j)
	if err != nil {
		log.Print(err)
	}

	id, err := h.jobman.Add(j)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
	io.WriteString(w, id)
}

func (h *apiHandler) handleCancel(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	err := h.jobman.Cancel(id)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
}

func (h *apiHandler) handleTree(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	j := h.jobman.Get(id)
	if j == nil {
		http.Error(w, fmt.Sprintf("job not found by id: %v", id), http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	j.Lock()
	defer j.Unlock()
	err := enc.Encode(j)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
}
