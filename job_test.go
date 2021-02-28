package coco

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var job = (&Job{
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
}).Init()

func TestInitJob(t *testing.T) {
	n := job.tasks[4].Stat.N()
	if n != 4 {
		t.Fatalf("unexpected t.Stat.N: %v", n)
	}
}

func TestWalkTask(t *testing.T) {
	got := []string{}
	for _, t := range job.tasks {
		got = append(got, t.Title)
	}
	want := []string{
		"root",
		"sim",
		"ocean",
		"foam",
		"render",
		"diffuse",
		"1",
		"2",
		"reflection",
		"1",
		"2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%v, want\n%v", got, want)
	}
}

func TestWalkLeafTask(t *testing.T) {
	got := []string{}
	for _, t := range job.tasks {
		if t.isLeaf {
			got = append(got, t.Title)
		}
	}
	want := []string{
		"ocean",
		"foam",
		"1",
		"2",
		"1",
		"2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%v, want\n%v", got, want)
	}
}

func TestWalkFrom(t *testing.T) {
	got := []string{}
	n := job.Subtasks[0].Subtasks[1].num
	for _, t := range job.tasks[n:] {
		got = append(got, t.Title)
	}
	want := []string{
		"foam",
		"render",
		"diffuse",
		"1",
		"2",
		"reflection",
		"1",
		"2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%v, want\n%v", got, want)
	}
}

func newJobManagerForPop() *JobManager {
	j1 := &Job{
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
	}
	j2 := &Job{
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
	}
	m := NewJobManager()
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
	shouldEqual := func(got, want *JobManager) error {
		// Unfortunately, jobManager.job and jobManager.task is hard to compare.
		// Fortunately, those are hardly changed after initialized. Skip them for now.
		for i, g := range got.jobs.heap {
			w := want.jobs.heap[i]
			sg := sprintJob(g)
			sw := sprintJob(w)
			if sg != sw {
				return fmt.Errorf("\ngot jobs[%v]: %v\nwant jobs[%v]: %v", i, sg, i, sw)
			}
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

// sprintJob only print partial fields of the Job that is meaningful for comparing two jobs.
func sprintJob(j *Job) string {
	jstr := fmt.Sprintf("Job order=%v, CurrentPriority=%v\n", j.order, j.CurrentPriority)
	tstr := sprintTask(j.Task, 1)
	return jstr + tstr
}

func sprintTask(t *Task, depth int) string {
	s := fmt.Sprintf("%v%v: popIdx=%v, priority=%v, Assignee=%v\n", strings.Repeat("\t", depth), t.Title, t.popIdx, t.Assignee, t.CalcPriority())
	for _, t := range t.Subtasks {
		s += sprintTask(t, depth+1)
	}
	return s
}
