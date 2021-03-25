package container

import (
	"reflect"
	"testing"
)

func TestUniqueHeap(t *testing.T) {
	cases := []struct {
		less func(i, j interface{}) bool
		vals []interface{}
		want []interface{}
	}{
		{
			less: func(i, j interface{}) bool {
				iv := i.(int)
				jv := j.(int)
				return iv < jv
			},
			vals: []interface{}{1, 1, 2, 3, 2, 4},
			want: []interface{}{1, 2, 3, 4},
		},
	}
	for _, c := range cases {
		h := NewUniqueHeap(c.less)
		for _, v := range c.vals {
			h.Push(v)
		}
		got := []interface{}{}
		for {
			v := h.Pop()
			if v == nil {
				return
			}
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got %v, want %v", got, c.want)
		}
	}
}

func TestUniqueHeapRemove(t *testing.T) {
	cases := []struct {
		less   func(i, j interface{}) bool
		vals   []interface{}
		remove []interface{}
		want   []interface{}
	}{
		{
			less: func(i, j interface{}) bool {
				iv := i.(int)
				jv := j.(int)
				return iv < jv
			},
			vals:   []interface{}{1, 2, 3, 4},
			remove: []interface{}{1, 2},
			want:   []interface{}{3, 4},
		},
	}
	for _, c := range cases {
		h := NewUniqueHeap(c.less)
		for _, v := range c.vals {
			h.Push(v)
		}
		for _, v := range c.remove {
			h.Remove(v)
		}
		got := []interface{}{}
		for {
			v := h.Pop()
			if v == nil {
				return
			}
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got %v, want %v", got, c.want)
		}
	}
}

func TestUniqueHeapRemoveThenPush(t *testing.T) {
	cases := []struct {
		less           func(i, j interface{}) bool
		vals           []interface{}
		removeThenPush []interface{}
		want           []interface{}
	}{
		{
			less: func(i, j interface{}) bool {
				iv := i.(int)
				jv := j.(int)
				return iv < jv
			},
			vals:           []interface{}{1, 2, 3, 4},
			removeThenPush: []interface{}{1, 2},
			want:           []interface{}{1, 2, 3, 4},
		},
	}
	for _, c := range cases {
		h := NewUniqueHeap(c.less)
		for _, v := range c.vals {
			h.Push(v)
		}
		for _, v := range c.removeThenPush {
			h.Remove(v)
			h.Push(v)
		}
		got := []interface{}{}
		for {
			v := h.Pop()
			if v == nil {
				return
			}
			got = append(got, v)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got %v, want %v", got, c.want)
		}
	}
}
