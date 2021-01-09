package main

import "testing"

func TestTaskCalcPriority(t *testing.T) {
	job := &Job{
		DefaultPriority: 1,
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
