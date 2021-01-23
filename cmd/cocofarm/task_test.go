package main

import (
	"reflect"
	"testing"
)

func TestTaskCalcPriority(t *testing.T) {
	job := &Job{
		Task: &Task{
			Priority: 1,
			Subtasks: []*Task{
				&Task{
					Priority: 0,
					Subtasks: []*Task{
						{
							Priority: 5,
						},
					},
				},
				&Task{
					Priority: 3,
					Subtasks: []*Task{
						&Task{
							Priority: 0,
						},
					},
				},
			},
		},
	}
	initJob(job)
	cases := []struct {
		t    *Task
		want int
	}{
		{
			t:    job.Subtasks[0],
			want: 1,
		},
		{
			t:    job.Subtasks[0].Subtasks[0],
			want: 5,
		},
		{
			t:    job.Subtasks[1],
			want: 3,
		},
		{
			t:    job.Subtasks[1].Subtasks[0],
			want: 3,
		},
	}
	for i, c := range cases {
		got := c.t.CalcPriority()
		if got != c.want {
			t.Fatalf("%d: got %v, want %v", i, got, c.want)
		}
	}
}

func TestTaskPop(t *testing.T) {
	cases := []struct {
		t    *Task
		want []string
	}{
		{
			t: &Task{
				SerialSubtasks: true,
				Subtasks: []*Task{
					&Task{
						Title:          "render",
						SerialSubtasks: false,
						Subtasks: []*Task{
							{
								Title:          "diffuse",
								SerialSubtasks: false,
								Subtasks: []*Task{
									{Title: "d1"},
									{Title: "d2"},
								},
							},
							{
								Title:          "reflection",
								SerialSubtasks: false,
								Subtasks: []*Task{
									{Title: "r1"},
									{Title: "r2"},
								},
							},
						},
					},
					&Task{
						Title:          "cleanup",
						SerialSubtasks: true,
						Subtasks: []*Task{
							{Title: "c1"},
							{Title: "c2"},
						},
					},
				},
			},
			want: []string{"d1", "d2", "r1", "r2"},
		},
		{
			t: &Task{
				SerialSubtasks: true,
				Subtasks: []*Task{
					&Task{
						Title:          "sim",
						SerialSubtasks: false,
						Subtasks: []*Task{
							{
								Title:          "destruction",
								SerialSubtasks: true,
								Subtasks: []*Task{
									{Title: "d1"},
									{Title: "d2"},
								},
							},
							{
								Title:          "fire",
								SerialSubtasks: true,
								Subtasks: []*Task{
									{Title: "f1"},
									{Title: "f2"},
								},
							},
							{
								Title:          "particle",
								SerialSubtasks: true,
								Subtasks: []*Task{
									{Title: "p1"},
									{Title: "p2"},
								},
							},
						},
					},
				},
			},
			want: []string{"d1", "f1", "p1"},
		},
	}

	titles := func(tasks []*Task) []string {
		sl := make([]string, len(tasks))
		for i, t := range tasks {
			sl[i] = t.Title
		}
		return sl
	}

	for _, c := range cases {
		ts := make([]*Task, 0)
		for {
			t, _ := c.t.Pop()
			if t == nil {
				break
			}
			ts = append(ts, t)
		}
		got := titles(ts)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got: %v, want: %v", got, c.want)
		}
	}
}
