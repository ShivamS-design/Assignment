package scheduler

import (
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
)

type TaskState int

const (
	TaskStateReady TaskState = iota
	TaskStateRunning
	TaskStateSuspended
	TaskStateTerminated
)

type Task struct {
	ID          uuid.UUID
	ModuleID    uuid.UUID
	State       TaskState
	Priority    int
	CreatedAt   time.Time
	StartedAt   time.Time
	CPUTime     time.Duration
	MemoryUsage int64
	Context     context.Context
	Cancel      context.CancelFunc
	mu          sync.RWMutex
}

func (t *Task) GetState() TaskState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State
}

func (t *Task) SetState(state TaskState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.State = state
}

func (t *Task) AddCPUTime(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CPUTime += duration
}

func (t *Task) UpdateMemory(usage int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.MemoryUsage = usage
}

type TaskMetrics struct {
	TaskID      uuid.UUID
	CPUTime     time.Duration
	MemoryUsage int64
	Switches    int64
	LastSwitch  time.Time
}

type ResourceLimits struct {
	MaxMemory     int64
	MaxCPUTime    time.Duration
	MaxTasks      int
	TimeSlice     time.Duration
}

type SchedulerConfig struct {
	Algorithm      string
	TimeSlice      time.Duration
	ResourceLimits ResourceLimits
	MetricsEnabled bool
}

type Scheduler interface {
	Schedule(task *Task) error
	Unschedule(taskID uuid.UUID) error
	GetNext() *Task
	GetTasks() []*Task
	GetMetrics() map[uuid.UUID]*TaskMetrics
	Start() error
	Stop() error
}