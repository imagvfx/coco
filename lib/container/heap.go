package container

import "container/heap"

// UniqueHeap is a heap that keeps same values only once.
type UniqueHeap struct {
	has     map[interface{}]bool
	removed map[interface{}]bool
	heap    *interfaceHeap
}

// NewUniqueHeap creates a new UniqueHeap.
func NewUniqueHeap(less func(i, j interface{}) bool) *UniqueHeap {
	return &UniqueHeap{
		has:     make(map[interface{}]bool),
		removed: make(map[interface{}]bool),
		heap:    newInterfaceHeap(less),
	}
}

// Push pushs an element to the heap.
// If the element already exists in the heap, it will just skip it.
func (h *UniqueHeap) Push(el interface{}) {
	if h.removed[el] {
		delete(h.removed, el)
		return
	}
	if h.has[el] {
		return
	}
	h.has[el] = true
	heap.Push(h.heap, el)
}

// Remove marks an element as removed from the heap.
// It doesn't remove the element right away.
// Pop will clean removed elements internally.
func (h *UniqueHeap) Remove(el interface{}) {
	if !h.has[el] {
		return
	}
	h.removed[el] = true
}

// Pop pops an element from the heap.
// It will return nil if the heap has no element.
func (h *UniqueHeap) Pop() interface{} {
	for {
		if h.heap.Len() == 0 {
			return nil
		}
		el := heap.Pop(h.heap)
		delete(h.has, el)
		if h.removed[el] {
			delete(h.removed, el)
			continue
		}
		return el
	}
}

// interfaceHeap is a heap with interfaces with a given less function.
type interfaceHeap struct {
	heap []interface{}
	less func(i, j interface{}) bool
}

// newInterfaceHeap creates a new interfaceHeap.
func newInterfaceHeap(less func(i, j interface{}) bool) *interfaceHeap {
	return &interfaceHeap{
		heap: make([]interface{}, 0),
		less: less,
	}
}

// Len is length of the heap.
func (h interfaceHeap) Len() int {
	return len(h.heap)
}

// Less is less function of the heap.
func (h interfaceHeap) Less(i, j int) bool {
	return h.less(h.heap[i], h.heap[j])
}

// Swap swaps position of two values within the heap.
func (h interfaceHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

// Push pushes an element to the heap.
func (h *interfaceHeap) Push(el interface{}) {
	h.heap = append(h.heap, el)
}

// Pop pops an element from the heap.
func (h *interfaceHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.heap = old[:n-1]
	return el
}
