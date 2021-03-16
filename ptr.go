package coco

// ptr functions are used for creating pointer to const and untyped values,
// which cannot take their address directly.
// It will be merged as one function 'ptr' when go has generics.

// ptrString returns a pointer to a string.
func ptrString(v string) *string {
	return &v
}

// ptrTaskStatus returns a pointer to a TaskStatus.
func ptrTaskStatus(v TaskStatus) *TaskStatus {
	return &v
}

// ptrWorkerStatus returns a pointer to a WorkerStatus.
func ptrWorkerStatus(v WorkerStatus) *WorkerStatus {
	return &v
}
