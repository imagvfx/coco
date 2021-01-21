package main

import (
	"fmt"
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

func ExampleTaskPop() {
	c := &Task{
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
							{Title: "d3"},
							{Title: "d4"},
							{Title: "d5"},
						},
					},
					{
						Title:          "reflection",
						SerialSubtasks: false,
						Subtasks: []*Task{
							{Title: "r1"},
							{Title: "r2"},
							{Title: "r3"},
							{Title: "r4"},
							{Title: "r5"},
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
	}
	for {
		popt, done := c.Pop()
		if done {
			break
		}
		if popt == nil {
			break
		}
		fmt.Println(popt.Title)
	}
	// Output:
	// d1
	// d2
	// d3
	// d4
	// d5
	// r1
	// r2
	// r3
	// r4
	// r5
}
