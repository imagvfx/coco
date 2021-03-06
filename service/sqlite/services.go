package sqlite

import (
	"database/sql"

	"github.com/imagvfx/coco/service"
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

func (s *Services) FarmService() service.FarmService {
	return s.fs
}

func (s *Services) JobService() service.JobService {
	return s.js
}

func (s *Services) WorkerService() service.WorkerService {
	return s.ws
}
