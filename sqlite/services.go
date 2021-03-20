package sqlite

import (
	"database/sql"

	"github.com/imagvfx/coco"
)

type Services struct {
	fs *FarmService
	js *JobService
	ws *WorkerService
}

func NewServices(db *sql.DB) *Services {
	return &Services{
		fs: NewFarmService(db),
		js: NewJobService(db),
		ws: NewWorkerService(db),
	}
}

func (s *Services) FarmService() coco.FarmService {
	return s.fs
}

func (s *Services) JobService() coco.JobService {
	return s.js
}

func (s *Services) WorkerService() coco.WorkerService {
	return s.ws
}
