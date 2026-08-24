package model

import "fmt"

// TaskState is one step in the scrape task lifecycle.
type TaskState int

const (
	// TaskIdle means the task has been created but not scheduled.
	TaskIdle TaskState = iota
	// TaskScheduled means the task is queued for a scrape cycle.
	TaskScheduled
	// TaskFetching means the fetch phase is running.
	TaskFetching
	// TaskParsed means the response body has been parsed into samples.
	TaskParsed
	// TaskForwarded means samples reached the downstream sink.
	TaskForwarded
	// TaskFailed means the task ended with an error.
	TaskFailed
)

// String renders a human readable task state.
func (s TaskState) String() string {
	switch s {
	case TaskIdle:
		return "idle"
	case TaskScheduled:
		return "scheduled"
	case TaskFetching:
		return "fetching"
	case TaskParsed:
		return "parsed"
	case TaskForwarded:
		return "forwarded"
	case TaskFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// allowedTransitions maps each state to the states reachable from it.
var allowedTransitions = map[TaskState][]TaskState{
	TaskIdle:      {TaskScheduled},
	TaskScheduled: {TaskFetching, TaskFailed},
	TaskFetching:  {TaskParsed, TaskFailed},
	TaskParsed:    {TaskForwarded, TaskFailed},
	TaskForwarded: {TaskIdle},
	TaskFailed:    {TaskIdle},
}

// Task tracks one scrape attempt for a single target.
type Task struct {
	TargetID string
	Cycle    int64
	State    TaskState
	Attempt  int
	ErrMsg   string
}

// NewTask creates an idle task for the given target and cycle.
func NewTask(targetID string, cycle int64) Task {
	return Task{TargetID: targetID, Cycle: cycle, State: TaskIdle}
}

// Transition moves the task to next if the state machine allows it.
func (t *Task) Transition(next TaskState) error {
	for _, allowed := range allowedTransitions[t.State] {
		if allowed == next {
			t.State = next
			return nil
		}
	}
	return fmt.Errorf("invalid task transition %s -> %s", t.State, next)
}

// RecordError marks the task failed and stores the failure message.
func (t *Task) RecordError(message string) {
	t.State = TaskFailed
	t.ErrMsg = message
}
