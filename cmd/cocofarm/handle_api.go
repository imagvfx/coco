package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	io.WriteString(w, fmt.Sprintf("%v", id))
}

func (h *apiHandler) handleCancel(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("job id is not valid: %v", err), http.StatusBadRequest)
		return
	}
	err = h.jobman.Cancel(JobID(id))
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
}

func (h *apiHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("job id is not valid: %v", err), http.StatusBadRequest)
		return
	}
	err = h.jobman.Retry(JobID(id))
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
}

func (h *apiHandler) handleJob(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("job id is not valid: %v", err), http.StatusBadRequest)
		return
	}
	j := h.jobman.Get(JobID(id))
	if j == nil {
		http.Error(w, fmt.Sprintf("job not found by id: %v", id), http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	j.Lock()
	defer j.Unlock()
	err = enc.Encode(j)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%v", err))
	}
}
