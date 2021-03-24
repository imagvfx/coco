package coco

// uniqueHeap is a heap that keeps same values only once.
type uniqueHeap struct {
	has     map[interface{}]bool
	removed map[interface{}]bool
	heap    []interface{}
	less    func(i, j interface{}) bool
}

// newUniqueHeap creates a new uniqueHeap.
func newUniqueHeap(less func(i, j interface{}) bool) *uniqueHeap {
	return &uniqueHeap{
		has:     make(map[interface{}]bool),
		removed: make(map[interface{}]bool),
		heap:    make([]interface{}, 0),
		less:    less,
	}
}

// Len is length of the heap.
func (h uniqueHeap) Len() int {
	return len(h.heap)
}

// Less is less function of the heap.
func (h uniqueHeap) Less(i, j int) bool {
	return h.less(h.heap[i], h.heap[j])
}

// Swap swaps position of two values within the heap.
func (h uniqueHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

// Push pushs an element to the heap.
// If the element already exists in the heap, it will just skip it.
func (h *uniqueHeap) Push(el interface{}) {
	if h.removed[el] {
		delete(h.removed, el)
		return
	}
	if h.has[el] {
		return
	}
	h.has[el] = true
	h.heap = append(h.heap, el)
}

// Remove marks an element as removed from the heap.
// It doesn't remove the element right away.
// Pop will clean removed elements internally.
func (h *uniqueHeap) Remove(el interface{}) {
	h.removed[el] = true
}

// Pop pops an element from the heap.
// It will panic if the heap has no element.
func (h *uniqueHeap) Pop() interface{} {
	for {
		old := h.heap
		n := len(old)
		el := old[n-1]
		old[n-1] = nil // avoid memory leak
		h.heap = old[:n-1]
		delete(h.has, el)
		if h.removed[el] {
			delete(h.removed, el)
			continue
		}
		return el
	}
}
