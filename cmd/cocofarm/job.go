package main

import (
	"sync"

	"github.com/imagvfx/coco"
)

type jobManager struct {
	sync.Mutex
	jobs []*coco.Job
}

func (m *jobManager) Add(j *coco.Job) {
	m.Lock()
	defer m.Unlock()
	m.jobs = append(m.jobs, j)
}

func (m *jobManager) Jobs() []*coco.Job {
	m.Lock()
	defer m.Unlock()
	return m.jobs
}
