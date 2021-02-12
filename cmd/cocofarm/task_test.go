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
		j    *Job
		want []string
	}{
		{
			j: initJob(&Job{
				Task: &Task{
					Title:          "simple serial",
					SerialSubtasks: true,
					Subtasks: []*Task{
						{Title: "1"},
						{Title: "2"},
						{Title: "3"},
					},
				},
			}),
			want: []string{"1", "", "2", "", "3", ""},
		},
		{
			j: initJob(&Job{
				Task: &Task{
					Title:          "simple parallel",
					SerialSubtasks: false,
					Subtasks: []*Task{
						{Title: "1"},
						{Title: "2"},
						{Title: "3"},
					},
				},
			}),
			want: []string{"1", "2", "3"},
		},
		{
			j: initJob(&Job{
				Task: &Task{
					Title:          "render",
					SerialSubtasks: true,
					Subtasks: []*Task{
						&Task{
							Title:          "layer",
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
			}),
			want: []string{"d1", "d2", "r1", "r2", "c1", "", "c2", ""},
		},
		{
			j: initJob(&Job{
				Task: &Task{
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
			}),
			want: []string{"d1", "f1", "p1", "", "d2", "f2", "p2", ""},
		},
	}

	for _, c := range cases {
		got := make([]string, 0)
		popTasks := make([]*Task, 0)
		for {
			popt, done := c.j.Pop()
			if popt == nil {
				if done {
					break
				}
				// pop blocked
				peek := c.j.Peek()
				if peek != nil {
					t.Fatalf("%v: peek should return nil when blocked", c.j.Title)
				}
				got = append(got, "") // make the blocking point visible
				for _, pt := range popTasks {
					pt.SetStatus(TaskDone)
				}
				popTasks = popTasks[:0]
			} else {
				got = append(got, popt.Title)
				popTasks = append(popTasks, popt)
			}

		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%v: got: %v, want: %v", c.j.Title, got, c.want)
		}
	}
}
