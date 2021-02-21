package coco

// uniqueQueue is a queue that has unique items.
// uniqueQueue is a queue, so the value pushed first will popped first.
// Same values cannot be exist in this queue.
type uniqueQueue struct {
	has   map[interface{}]bool
	first *queueItem
	last  *queueItem
}

// queueItem is a queueItem that wraps a value.
// It directs the next queueItem, so the queue can traverse.
type queueItem struct {
	v    interface{}
	next *queueItem
}

// newUniqueQueue creates a new uniqueQueue.
func newUniqueQueue() *uniqueQueue {
	return &uniqueQueue{
		has: make(map[interface{}]bool),
	}
}

// Push pushs a value to the queue.
// If the same value has already exists in the queue, it does nothing.
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

// Pop pops a value from the queue.
// If there isn't any value in the queue, it returns nil.
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

// Remove finds and removes the given value from the queue.
// If the queue has the value, it removes the value and returns true.
// Otherwise, it does nothing and returns false.
func (q *uniqueQueue) Remove(v interface{}) bool {
	if !q.has[v] {
		return false
	}
	delete(q.has, v)
	var prev *queueItem
	for it := q.first; it != nil; it = it.next {
		if it.v == v {
			if it == q.first {
				q.first = q.first.next
			} else {
				prev.next = it.next
			}
			if it == q.last {
				q.last = prev
			}
			break
		}
		prev = it
	}
	return true
}
