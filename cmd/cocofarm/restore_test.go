package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/sqlite"
)

func newJob() *coco.Job {
	return &coco.Job{
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
	}
}

func TestCocoRestoreJobManager(t *testing.T) {
	test := func(i int) {
		db, err := sqlite.Create(t.TempDir() + fmt.Sprintf("/test_%v.db", i))
		if err != nil {
			t.Fatalf("cannot create db: %v", err)
		}
		defer db.Close()
		js := sqlite.NewJobService(db)
		workerman, err := coco.NewWorkerManager(&coco.NopWorkerService{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		jobman, err := coco.NewJobManager(js)
		if err != nil {
			t.Fatal(err)
		}
		farm := coco.NewFarm(jobman, workerman)
		_, err = jobman.Add(newJob())
		if err != nil {
			t.Fatalf("add: %v", err)
		}
		// TODO: need a better way to test restored popIdx
		// This one only catches the situation that everything has done well.
		for {
			t := jobman.PopTask([]string{"*"})
			if t == nil {
				break
			}
			// <-- THE RANDOM PROCESS
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(5)
			if n < 4 {
				farm.Done("", t.ID())
			} else {
				farm.Failed("", t.ID())
			}
		}
		// TODO: test with updated tasks
		r, err := coco.NewJobManager(js)
		if err != nil {
			t.Fatalf("restore: %v", err)
		}
		for {
			want := jobman.PopTask([]string{"*"})
			got := r.PopTask([]string{"*"})
			err := coco.ShouldEqualTask(got, want)
			if err != nil {
				t.Fatalf("restore job manager: %v", err)
			}
			if want == nil {
				return
			}
		}
	}
	// The test could produce different results due to the random process inside of it.
	// Run reasonable times.
	for i := 0; i < 10; i++ {
		test(i)
	}
}
