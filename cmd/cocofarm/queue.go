package main

type uniqueQueue struct {
	has   map[interface{}]bool
	first *queueItem
	last  *queueItem
}

type queueItem struct {
	v    interface{}
	next *queueItem
}

func newUniqueQueue() *uniqueQueue {
	return &uniqueQueue{
		has: make(map[interface{}]bool),
	}
}

func (q *uniqueQueue) Push(v interface{}) {
	if q.has[v] {
		return
	}
	q.has[v] = true
	item := &queueItem{v: v}
	if q.first == nil {
		q.first = item
	} else {
		q.last.next = item
	}
	q.last = item
}

func (q *uniqueQueue) Pop() interface{} {
	if q.first == nil {
		return nil
	}
	v := q.first.v
	delete(q.has, v)
	if q.first == q.last {
		q.first = nil
		q.last = nil
		return v
	}
	q.first = q.first.next
	return v
}

func (q *uniqueQueue) Remove(v interface{}) bool {
	if !q.has[v] {
		return false
	}
	delete(q.has, v)
	var prev *queueItem
	for it := q.first; it != q.last; it = it.next {
		if it.v == v {
			prev.next = it.next
			break
		}
		prev = it
	}
	return true
}
