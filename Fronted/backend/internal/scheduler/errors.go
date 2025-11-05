package scheduler

import "errors"

var (
	ErrTaskNotFound         = errors.New("task not found")
	ErrMaxTasksReached      = errors.New("maximum number of tasks reached")
	ErrResourceExhausted    = errors.New("insufficient resources")
	ErrSchedulerRunning     = errors.New("scheduler is already running")
	ErrSchedulerNotRunning  = errors.New("scheduler is not running")
	ErrInvalidPriority      = errors.New("invalid task priority")
	ErrTaskAlreadyExists    = errors.New("task already exists")
	ErrInvalidTimeSlice     = errors.New("invalid time slice duration")
	ErrContextCancelled     = errors.New("task context was cancelled")
)