package coco

type jobHeap struct {
	heap []*Job
}

func newJobHeap() *jobHeap {
	return &jobHeap{
		heap: make([]*Job, 0),
	}
}

func (h jobHeap) Len() int {
	return len(h.heap)
}

func (h jobHeap) Less(i, j int) bool {
	if h.heap[i].CurrentPriority > h.heap[j].CurrentPriority {
		return true
	}
	if h.heap[i].CurrentPriority < h.heap[j].CurrentPriority {
		return false
	}
	return h.heap[i].order < h.heap[j].order
}

func (h jobHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

func (h *jobHeap) Push(el interface{}) {
	h.heap = append(h.heap, el.(*Job))
}

func (h *jobHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.heap = old[:n-1]
	return el
}
