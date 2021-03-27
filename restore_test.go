package coco

import (
	"fmt"
	"testing"

	"github.com/imagvfx/coco/service/sqlite"
)

func newJob() *Job {
	return &Job{
		Task: &Task{
			Title:          "root",
			SerialSubtasks: true,
			Subtasks: []*Task{
				&Task{
					Title:          "sim",
					SerialSubtasks: true,
					Subtasks: []*Task{
						&Task{
							Title:          "ocean",
							SerialSubtasks: true,
						},
						&Task{
							Title:          "foam",
							SerialSubtasks: true,
						},
					},
				},
				&Task{
					Title:          "render",
					SerialSubtasks: true,
					Subtasks: []*Task{
						&Task{
							Title:          "diffuse",
							SerialSubtasks: false,
							Subtasks: []*Task{
								&Task{
									Title:          "1",
									SerialSubtasks: true,
								},
								&Task{
									Title:          "2",
									SerialSubtasks: true,
								},
							},
						},
						&Task{
							Title:          "reflection",
							SerialSubtasks: false,
							Subtasks: []*Task{
								&Task{
									Title:          "1",
									SerialSubtasks: true,
								},
								&Task{
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

func TestRestoreJobManager(t *testing.T) {
	test := func(i int) {
		dbpath := t.TempDir() + fmt.Sprintf("/test_%v.db", i)
		// fmt.Println(dbpath)
		db, err := sqlite.Create(dbpath)
		if err != nil {
			t.Fatalf("cannot create db: %v", err)
		}
		defer db.Close()
		services := sqlite.NewServices(db)
		farm, err := NewFarm(services, nil)
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
		w := NewWorker(worker)
		err = workerman.Add(w)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for n <= i {
			popt := jobman.PopTask([]string{"*"})
			if popt == nil {
				t.Fatal("should get a task")
			}
			err := farm.Assign(worker, popt.ID)
			if err != nil {
				t.Fatal(err)
			}
			if n < i {
				err := farm.Done(worker, popt.ID)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				// n == i
				err := farm.Failed(worker, popt.ID)
				if err != nil {
					t.Fatal(err)
				}
			}
			n++
		}
		// time.Sleep(time.Hour)
		f, err := NewFarm(services, nil)
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
		err = ShouldEqualJob(got, want)
		if err != nil {
			t.Fatal(err)
		}

	}
	// test all possible results
	for i := 0; i < 6; i++ {
		test(i)
	}
}
