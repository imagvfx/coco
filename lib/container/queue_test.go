package container

import (
	"reflect"
	"testing"
)

func TestUniqueQueue(t *testing.T) {
	workerA := "a.imagvfx.com:8283"
	workerB := "b.imagvfx.com:8283"
	workers := []string{workerA, workerB}
	q := NewUniqueQueue()
	for _, w := range workers {
		q.Push(w)
	}
	got := make([]string, 0)
	for {
		v := q.Pop()
		if v == nil {
			break
		}
		w := v.(string)
		got = append(got, w)
	}
	if !reflect.DeepEqual(got, workers) {
		t.Fatalf("got: %v, want: %v", got, workers)
	}
}

func TestUniqueQueueRemove(t *testing.T) {
	workerA := "a.imagvfx.com:8283"
	workerB := "b.imagvfx.com:8283"
	workers := []string{workerA, workerB}
	q := NewUniqueQueue()
	for _, w := range workers {
		q.Push(w)
	}
	for _, w := range workers {
		removed := q.Remove(w)
		if !removed {
			t.Fatalf("%v wasn't removed", w)
		}
	}
	v := q.Pop()
	if v != nil {
		t.Fatalf("queue should be empty, got %v", v)
	}
}
