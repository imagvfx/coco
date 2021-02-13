package main

import (
	"fmt"
	"reflect"
	"strings"
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

func newJobManagerForPop() *jobManager {
	j1 := initJob(&Job{
		Target: "2d",
		Task: &Task{
			Title:    "0",
			Priority: 1,
			Subtasks: []*Task{
				&Task{
					Title:    "untitled",
					Priority: 0, // for checking priority inheritance
					Subtasks: []*Task{
						&Task{
							Title:    "0-0",
							Priority: 2, // 2
						},
						&Task{
							Title:    "0-1",
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
			Title:    "1",
			Priority: 2,
			Subtasks: []*Task{
				&Task{
					Title:    "untitled",
					Priority: 0, // for checking priority inheritance
					Subtasks: []*Task{
						&Task{
							Title:    "1-0",
							Priority: 0, // 2
						},
						&Task{
							Title:    "1-1",
							Priority: 3, // 3
						},
					},
				},
			},
		},
	})
	m := newJobManager()
	m.Add(j1)
	m.Add(j2)
	return m
}

func TestJobManagerPopTask(t *testing.T) {
	cases := []struct {
		targets []string
		want    []string
	}{
		{
			targets: []string{"*"},
			want: []string{
				"0-0", // 2 with lower ID
				"1-0", // 2 with higher ID
				"1-1", // 3
				"0-1", // 1
			},
		},
		{
			targets: []string{"2d", "3d"},
			want: []string{
				"0-0", // 2 with lower ID
				"1-0", // 2 with higher ID
				"1-1", // 3
				"0-1", // 1
			},
		},
		{
			targets: []string{"2d"},
			want: []string{
				"0-0",
				"0-1",
			},
		},
		{
			targets: []string{"3d"},
			want: []string{
				"1-0",
				"1-1",
			},
		},
		{
			targets: []string{},
			want:    []string{},
		},
	}
	for _, c := range cases {
		m := newJobManagerForPop()
		got := []string{}
		for {
			t := m.PopTask(c.targets)
			if t == nil {
				break
			}
			got = append(got, t.Title)
		}
		if len(got) != len(c.want) {
			t.Fatalf("targets %v: length: got %v, want %v", c.targets, got, c.want)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("targets %v: got %v, want %v", c.targets, got, c.want)
		}
	}
}

func TestJobManagerPopTaskThenPushTask(t *testing.T) {
	shouldEqual := func(got, want *jobManager) error {
		// Unfortunately, jobManager.job and jobManager.task is hard to compare.
		// Fortunately, those are hardly changed after initialized. Skip them for now.
		if !reflect.DeepEqual(got.jobBlocked, want.jobBlocked) {
			return fmt.Errorf("jobBlocked: got %v, want %v", got.jobBlocked, want.jobBlocked)
		}
		if !reflect.DeepEqual(got.jobPriority, want.jobPriority) {
			return fmt.Errorf("jobPriority: got %v, want %v", got.jobPriority, want.jobPriority)
		}
		sprint := func(d int, t *Task) string {
			return fmt.Sprintf("%v%v: popIdx=%v, priority=%v\n", strings.Repeat("\t", d), t.Title, t.popIdx, t.CalcPriority())
		}
		for i, g := range got.jobs.heap {
			w := want.jobs.heap[i]
			sg := sprintJob(g, sprint)
			sw := sprintJob(w, sprint)
			if sg != sw {
				return fmt.Errorf("\ngot jobs[%v]: %v\nwant jobs[%v]: %v", i, sg, i, sw)
			}
		}
		if !reflect.DeepEqual(got.assignee, want.assignee) {
			return fmt.Errorf("assignee: got %v, want %v", got.assignee, want.assignee)
		}
		return nil
	}

	m := newJobManagerForPop()
	tasks := make([]*Task, 0)
	for {
		t := m.PopTask([]string{"*"})
		if t == nil {
			break
		}
		tasks = append(tasks, t)
	}
	for _, t := range tasks {
		m.PushTask(t)
	}
	want := newJobManagerForPop()

	err := shouldEqual(m, want)
	if err != nil {
		t.Fatalf("pop all then push all should get the same jobManager: %v", err)
	}
}

func sprintJob(j *Job, sprintTaskFn func(d int, t *Task) string) string {
	return sprintTask(j.Task, 0, sprintTaskFn)
}

func sprintTask(t *Task, depth int, sprintTaskFn func(d int, t *Task) string) string {
	s := sprintTaskFn(depth, t)
	for _, t := range t.Subtasks {
		s += sprintTask(t, depth+1, sprintTaskFn)
	}
	return s
}
