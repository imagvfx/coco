package main

import (
	"fmt"
	"testing"

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
		services := sqlite.NewServices(db)
		farm, err := coco.NewFarm(services, nil)
		if err != nil {
			t.Fatal(err)
		}
		jobman := farm.JobManager()
		jid, err := jobman.Add(newJob())
		if err != nil {
			t.Fatalf("add: %v", err)
		}
		workerman := farm.WorkerManager()
		worker := "localhost:0000"
		w := coco.NewWorker(worker)
		err = workerman.Add(w)
		n := 0
		for n <= i {
			popt := jobman.PopTask([]string{"*"})
			if popt == nil {
				t.Fatal("should get a task")
			}
			err := farm.Assign(worker, popt.ID())
			if err != nil {
				t.Fatal(err)
			}
			if n < i {
				err := farm.Done(worker, popt.ID())
				if err != nil {
					t.Fatal(err)
				}
			} else {
				// n == i
				err := farm.Failed(worker, popt.ID())
				if err != nil {
					t.Fatal(err)
				}
			}
			n++
		}
		f, err := coco.NewFarm(services, nil)
		if err != nil {
			t.Fatalf("restore: %v", err)
		}
		r := f.JobManager()
		want, err := jobman.Get(jid)
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.Get(jid)
		if err != nil {
			t.Fatal(err)
		}
		err = coco.ShouldEqualJob(got, want)
		if err != nil {
			t.Fatal(err)
		}

	}
	// test all possible results
	for i := 0; i < 6; i++ {
		test(i)
	}
}
