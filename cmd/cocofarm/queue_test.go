package main

import (
	"reflect"
	"testing"
)

func TestUniqueQueue(t *testing.T) {
	workerA := &Worker{addr: "a.imagvfx.com:8283"}
	workerB := &Worker{addr: "b.imagvfx.com:8283"}
	workers := []*Worker{workerA, workerB}
	q := newUniqueQueue()
	for _, w := range workers {
		q.Push(w)
	}
	got := make([]*Worker, 0)
	for {
		v := q.Pop()
		if v == nil {
			break
		}
		w := v.(*Worker)
		got = append(got, w)
	}
	if !reflect.DeepEqual(got, workers) {
		t.Fatalf("got: %v, want: %v", got, workers)
	}
}
