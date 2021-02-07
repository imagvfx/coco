package main

import (
	"fmt"
	"reflect"
	"testing"
)

var job = initJob(&Job{
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
})

func TestJobManagerPopTask(t *testing.T) {
	j1 := initJob(&Job{
		Target: "2d",
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
	})
	j2 := initJob(&Job{
		Target: "3d",
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
	})

	cases := []struct {
		targets []string
		want    []*Task
	}{
		{
			targets: []string{"*"},
			want: []*Task{
				j1.Subtasks[0].Subtasks[0], // 2 with lower ID
				j2.Subtasks[0].Subtasks[0], // 2 with higher ID
				j2.Subtasks[0].Subtasks[1], // 1
				j1.Subtasks[0].Subtasks[1], // 3
			},
		},
		{
			targets: []string{"2d", "3d"},
			want: []*Task{
				j1.Subtasks[0].Subtasks[0],
				j2.Subtasks[0].Subtasks[0],
				j2.Subtasks[0].Subtasks[1],
				j1.Subtasks[0].Subtasks[1],
			},
		},
		{
			targets: []string{"2d"},
			want: []*Task{
				j1.Subtasks[0].Subtasks[0],
				j1.Subtasks[0].Subtasks[1],
			},
		},
		{
			targets: []string{"3d"},
			want: []*Task{
				j2.Subtasks[0].Subtasks[0], // 2 with higher ID
				j2.Subtasks[0].Subtasks[1], // 1
			},
		},
		{
			targets: []string{},
			want:    []*Task{},
		},
	}
	for _, c := range cases {
		m := newJobManager()
		m.Add(j1)
		m.Add(j2)

		got := []*Task{}
		for {
			t := m.PopTask(c.targets)
			if t == nil {
				break
			}
			got = append(got, t)
		}
		if len(got) != len(c.want) {
			t.Fatalf("targets %v: length: got %v, want %v", c.targets, got, c.want)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("targets %v: got %v, want %v", c.targets, got, c.want)
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
