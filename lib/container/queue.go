package container

// UniqueQueue is a queue that has unique items.
// UniqueQueue is a queue, so the value pushed first will popped first.
// Same values cannot be exist in this queue.
type UniqueQueue struct {
	has     map[interface{}]bool
	removed map[interface{}]bool
	first   *queueItem
	last    *queueItem
}

// queueItem is a queueItem that wraps a value.
// It directs the next queueItem, so the queue can traverse.
type queueItem struct {
	v    interface{}
	next *queueItem
}

// newUniqueQueue creates a new UniqueQueue.
func NewUniqueQueue() *UniqueQueue {
	return &UniqueQueue{
		has:     make(map[interface{}]bool),
		removed: make(map[interface{}]bool),
	}
}

// Push pushs a value to the queue.
// If the same value has already exists in the queue, it does nothing.
func (q *UniqueQueue) Push(v interface{}) {
	if q.removed[v] {
		delete(q.removed, v)
		return
	}
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
// It will clean up any removed value it met.
func (q *UniqueQueue) Pop() interface{} {
	for {
		if q.first == nil {
			return nil
		}
		v := q.first.v
		if q.first == q.last {
			q.first = nil
			q.last = nil
		} else {
			q.first = q.first.next
		}
		delete(q.has, v)
		if q.removed[v] {
			delete(q.removed, v)
			continue
		}
		return v
	}
}

// Remove finds and removes the given value from the queue.
// If the queue has the value, it removes the value and returns true.
// Otherwise, it does nothing and returns false.
// It doesn't remove the element right away.
// Pop will clean removed elements internally.
func (q *UniqueQueue) Remove(v interface{}) bool {
	if !q.has[v] {
		return false
	}
	if q.removed[v] {
		return false
	}
	q.removed[v] = true
	return true
}
