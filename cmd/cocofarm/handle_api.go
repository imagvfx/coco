package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/service"
)

type apiHandler struct {
	jobman *coco.JobManager
}

func (h *apiHandler) handleOrder(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	j := &coco.Job{}
	err := dec.Decode(j)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	id, err := h.jobman.Add(j)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	io.WriteString(w, fmt.Sprintf("%v", id))
}

func (h *apiHandler) handleCancel(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	jid, err := coco.JobIDFromString(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.jobman.Cancel(jid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *apiHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	jid, err := coco.JobIDFromString(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.jobman.Retry(jid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *apiHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	jid, err := coco.JobIDFromString(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.jobman.Delete(jid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *apiHandler) handleJob(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.Form.Get("id")
	jid, err := coco.JobIDFromString(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	j, err := h.jobman.Get(jid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	j.Lock()
	defer j.Unlock()
	enc := json.NewEncoder(w)
	err = enc.Encode(j)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *apiHandler) handleList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	target := r.Form.Get("target")
	filter := service.JobFilter{
		Target: target,
	}
	jobs := h.jobman.Jobs(filter)
	for _, j := range jobs {
		j.Lock()
		defer j.Unlock()
	}
	enc := json.NewEncoder(w)
	err := enc.Encode(jobs)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
