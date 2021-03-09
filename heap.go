package coco

type uniqueHeap struct {
	has  map[interface{}]bool
	heap []interface{}
	less func(i, j interface{}) bool
}

func newUniqueHeap(less func(i, j interface{}) bool) *uniqueHeap {
	return &uniqueHeap{
		has:  make(map[interface{}]bool),
		heap: make([]interface{}, 0),
		less: less,
	}
}

func (h uniqueHeap) Len() int {
	return len(h.heap)
}

func (h uniqueHeap) Less(i, j int) bool {
	return h.less(h.heap[i], h.heap[j])
}

func (h uniqueHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

func (h *uniqueHeap) Push(el interface{}) {
	if h.has[el] {
		return
	}
	h.has[el] = true
	h.heap = append(h.heap, el)
}

func (h *uniqueHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	el := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.heap = old[:n-1]
	delete(h.has, el)
	return el
}
