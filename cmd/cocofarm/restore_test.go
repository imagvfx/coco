package main

import (
	"reflect"
	"testing"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/sqlite"
)

var job = (&coco.Job{
	Task: &coco.Task{
		Title:          "root",
		SerialSubtasks: true,
		Subtasks: []*coco.Task{
			&coco.Task{
				Title:          "sim",
				SerialSubtasks: true,
				Subtasks: []*coco.Task{
					&coco.Task{
						Title:          "ocean",
						SerialSubtasks: true,
					},
					&coco.Task{
						Title:          "foam",
						SerialSubtasks: true,
					},
				},
			},
			&coco.Task{
				Title:          "render",
				SerialSubtasks: true,
				Subtasks: []*coco.Task{
					&coco.Task{
						Title:          "diffuse",
						SerialSubtasks: false,
						Subtasks: []*coco.Task{
							&coco.Task{
								Title:          "1",
								SerialSubtasks: true,
							},
							&coco.Task{
								Title:          "2",
								SerialSubtasks: true,
							},
						},
					},
					&coco.Task{
						Title:          "reflection",
						SerialSubtasks: false,
						Subtasks: []*coco.Task{
							&coco.Task{
								Title:          "1",
								SerialSubtasks: true,
							},
							&coco.Task{
								Title:          "2",
								SerialSubtasks: true,
							},
						},
					},
				},
			},
		},
	},
}).Init()

func TestCocoRestoreJobManager(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("cannot open db: %v", err)
	}
	err = sqlite.Init(db)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	m := coco.NewJobManager()
	js := sqlite.NewJobService(db)
	m.JobService = js
	_, err = m.Add(job)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	// TODO: test with updated tasks
	r, err := coco.RestoreJobManager(js)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	for {
		want := m.PopTask([]string{"*"})
		got := r.PopTask([]string{"*"})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("restore job manager: got %+v, want %+v", got, want)
		}
		if want == nil {
			return
		}
	}
}
