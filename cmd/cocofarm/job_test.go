package main

import (
	"fmt"
	"reflect"
	"testing"
)

var job = &Job{
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

func init() {
	initJob(job)
}

func TestJobManagerPopTask(t *testing.T) {
	j1 := &Job{
		Task: &Task{
			Priority: 1,
			Subtasks: []*Task{
				&Task{
					Priority: 0, // for checking priority inheritance
					Subtasks: []*Task{
						&Task{
							Priority: 3, // 2
						},
						&Task{
							Priority: 0, // 1
						},
					},
				},
			},
		},
	}
	initJob(j1)
	j2 := &Job{
		Task: &Task{
			Priority: 2,
			Subtasks: []*Task{
				&Task{
					Priority: 0, // for checking priority inheritance
					Subtasks: []*Task{
						&Task{
							Priority: 0, // 2
						},
						&Task{
							Priority: 3, // 3
						},
					},
				},
			},
		},
	}
	initJob(j2)
	m := newJobManager()
	m.Add(j1)
	m.Add(j2)
	want := []*Task{
		j1.Subtasks[0].Subtasks[0], // 2 with lower ID
		j2.Subtasks[0].Subtasks[0], // 2 with higher ID
		j2.Subtasks[0].Subtasks[1], // 1
		j1.Subtasks[0].Subtasks[1], // 3
	}
	got := []*Task{}
	for {
		t := m.PopTask()
		if t == nil {
			break
		}
		got = append(got, t)
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestParallelTaskGroups(t *testing.T) {
	cases := []struct {
		j    *Job
		want [][]*Task
	}{
		{
			j: job,
			want: [][]*Task{
				[]*Task{
					job.Subtasks[0].Subtasks[0], // sim/ocean
				},
				[]*Task{
					job.Subtasks[0].Subtasks[1], // sim/foam
				},
				[]*Task{
					job.Subtasks[1].Subtasks[0].Subtasks[0], // render/diffuse/0
					job.Subtasks[1].Subtasks[0].Subtasks[1], // render/diffuse/1
				},
				[]*Task{
					job.Subtasks[1].Subtasks[1].Subtasks[0], // render/reflection/0
					job.Subtasks[1].Subtasks[1].Subtasks[1], // render/reflection/1
				},
			},
		},
	}
	for _, c := range cases {
		got := parallelTaskGroups(job)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got: %v, want: %v", got, c.want)
		}
	}
}

func ExampleWalkTaskFn() {
	fn := func(t *Task) {
		fmt.Println(t.Title)
	}
	job.WalkTaskFn(fn)
	// Output:
	// root
	// sim
	// ocean
	// foam
	// render
	// diffuse
	// 1
	// 2
	// reflection
	// 1
	// 2
}

func ExampleWalkLeafTaskFn() {
	fn := func(t *Task) {
		fmt.Println(t.Title)
	}
	job.WalkLeafTaskFn(fn)
	// Output:
	// ocean
	// foam
	// 1
	// 2
	// 1
	// 2
}

func ExampleWalkFromFn() {
	fn := func(t *Task) {
		fmt.Println(t.Title)
	}
	walkFromFn(job.Subtasks[0].Subtasks[1], fn)
	// Output:
	// foam
	// render
	// diffuse
	// 1
	// 2
	// reflection
	// 1
	// 2
}
