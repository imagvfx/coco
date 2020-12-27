package main

import "testing"

func TestJobManagerNextTask(t *testing.T) {
	j1 := &Job{
		DefaultPriority: 1,
		Root: &Task{
			Priority: 0,
			Subtasks: []*Task{
				&Task{
					Priority: 3, // 2
				},
				&Task{
					Priority: 0, // 1
				},
			},
		},
	}
	initJob(j1)
	j2 := &Job{
		DefaultPriority: 2,
		Root: &Task{
			Priority: 0,
			Subtasks: []*Task{
				&Task{
					Priority: 0, // 2
				},
				&Task{
					Priority: 3, // 3
				},
			},
		},
	}
	initJob(j2)
	m := newJobManager()
	m.Add(j1)
	m.Add(j2)
	want := []*Task{
		j1.Root.Subtasks[0], // 2 with lower ID
		j2.Root.Subtasks[0], // 2 with higher ID
		j2.Root.Subtasks[1], // 1
		j1.Root.Subtasks[1], // 3
	}
	got := []*Task{}
	for {
		t := m.NextTask()
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
