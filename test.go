package coco

import "fmt"

// ShouldEqualJob checks that given two jobs (got, want) are equal.
// It returns an error which tell which member is different, when find a difference.
// It doesn't compare pointer addresses, but their values if possible.
// It doesn't compare it's subtasks except the root task.
func ShouldEqualJob(got, want *Job) error {
	if got == nil && want == nil {
		return nil
	}
	if got == nil {
		return fmt.Errorf("only got is nil")
	}
	if want == nil {
		return fmt.Errorf("only want is nil")
	}
	if got.order != want.order {
		return fmt.Errorf("order: got %v, want %v", got.order, want.order)
	}
	if got.Target != want.Target {
		return fmt.Errorf("Target: got %v, want %v", got.Target, want.Target)
	}
	if got.AutoRetry != want.AutoRetry {
		return fmt.Errorf("AutoRetry: got %v, want %v", got.AutoRetry, want.AutoRetry)
	}
	if len(got.tasks) != len(want.tasks) {
		return fmt.Errorf("len(tasks): got %v, want %v", len(got.tasks), len(want.tasks))
	}
	if got.CurrentPriority != want.CurrentPriority {
		return fmt.Errorf("CurrentPriority: got %v, want %v", got.CurrentPriority, want.CurrentPriority)
	}
	return ShouldEqualTask(got.Task, want.Task)
}

// ShouldEqualTask checks that given two tasks are equal and raises an error
// about which parts are different between two.
// It considers the first is 'got' and the second is 'want'.
// It's doesn't compare pointer to pointer directly, but their values.
func ShouldEqualTask(got, want *Task) error {
	if got == nil && want == nil {
		return nil
	}
	if got == nil {
		return fmt.Errorf("only got is nil")
	}
	if want == nil {
		return fmt.Errorf("only want is nil")
	}
	if got.Job.order != want.Job.order {
		return fmt.Errorf("job: got %v, want %v", got.Job.order, want.Job.order)
	}
	if got.num != want.num {
		return fmt.Errorf("num: got %v, want %v", got.num, want.num)
	}
	if !(got.parent == nil && want.parent == nil) {
		if got.parent != nil && want.parent != nil {
			if got.parent.num != want.parent.num {
				return fmt.Errorf("parent: got %v, want %v", got.parent.num, want.parent.num)
			}
		} else if got.parent == nil {
			return fmt.Errorf("only got.parent is nil")
		} else {
			return fmt.Errorf("only want.parent is nil")
		}
	}
	if got.Title != want.Title {
		return fmt.Errorf("Title: got %v, want %v", got.Title, want.Title)
	}
	if got.Priority != want.Priority {
		return fmt.Errorf("Priority: got %v, want %v", got.Priority, want.Priority)
	}
	if got.nthChild != want.nthChild {
		return fmt.Errorf("nthChild: got %v, want %v", got.nthChild, want.nthChild)
	}
	if got.popIdx != want.popIdx {
		return fmt.Errorf("popIdx: got %v, want %v", got.popIdx, want.popIdx)
	}
	if got.isLeaf != want.isLeaf {
		return fmt.Errorf("isLeaf: got %v, want %v", got.isLeaf, want.isLeaf)
	}
	if got.retry != want.retry {
		return fmt.Errorf("retry: got %v, want %v", got.retry, want.retry)
	}
	if len(got.Subtasks) != len(want.Subtasks) {
		return fmt.Errorf("len(Subtasks): got %v, want %v", len(got.Subtasks), len(want.Subtasks))
	}
	if len(got.Commands) != len(want.Commands) {
		return fmt.Errorf("len(Commands): got %v, want %v", len(got.Commands), len(want.Commands))
	}
	return nil
}
