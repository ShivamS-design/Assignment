package scheduler

import (
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
)

type CooperativeScheduler struct {
	tasks     []*Task
	ready     chan *Task
	config    SchedulerConfig
	metrics   map[uuid.UUID]*TaskMetrics
	running   bool
	stopCh    chan struct{}
	mu        sync.RWMutex
	metricsMu sync.RWMutex
}

func NewCooperativeScheduler(config SchedulerConfig) *CooperativeScheduler {
	return &CooperativeScheduler{
		tasks:   make([]*Task, 0),
		ready:   make(chan *Task, 100),
		config:  config,
		metrics: make(map[uuid.UUID]*TaskMetrics),
		stopCh:  make(chan struct{}),
	}
}

func (s *CooperativeScheduler) Schedule(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.tasks) >= s.config.ResourceLimits.MaxTasks {
		return ErrMaxTasksReached
	}

	s.tasks = append(s.tasks, task)
	task.SetState(TaskStateReady)

	if s.config.MetricsEnabled {
		s.metricsMu.Lock()
		s.metrics[task.ID] = &TaskMetrics{
			TaskID:     task.ID,
			LastSwitch: time.Now(),
		}
		s.metricsMu.Unlock()
	}

	select {
	case s.ready <- task:
	default:
	}

	return nil
}

func (s *CooperativeScheduler) Unschedule(taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.tasks {
		if task.ID == taskID {
			task.SetState(TaskStateTerminated)
			if task.Cancel != nil {
				task.Cancel()
			}
			
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			
			s.metricsMu.Lock()
			delete(s.metrics, taskID)
			s.metricsMu.Unlock()
			
			return nil
		}
	}
	return ErrTaskNotFound
}

func (s *CooperativeScheduler) GetNext() *Task {
	select {
	case task := <-s.ready:
		if task.GetState() != TaskStateTerminated {
			return task
		}
	default:
	}
	return nil
}

func (s *CooperativeScheduler) GetTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*Task, len(s.tasks))
	copy(result, s.tasks)
	return result
}

func (s *CooperativeScheduler) GetMetrics() map[uuid.UUID]*TaskMetrics {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	
	result := make(map[uuid.UUID]*TaskMetrics)
	for k, v := range s.metrics {
		result[k] = &TaskMetrics{
			TaskID:      v.TaskID,
			CPUTime:     v.CPUTime,
			MemoryUsage: v.MemoryUsage,
			Switches:    v.Switches,
			LastSwitch:  v.LastSwitch,
		}
	}
	return result
}

func (s *CooperativeScheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrSchedulerRunning
	}
	s.running = true
	s.mu.Unlock()

	go s.schedulerLoop()
	return nil
}

func (s *CooperativeScheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	
	for _, task := range s.GetTasks() {
		s.Unschedule(task.ID)
	}
	
	return nil
}

func (s *CooperativeScheduler) schedulerLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case task := <-s.ready:
			if task.GetState() == TaskStateTerminated {
				continue
			}
			s.executeTask(task)
		}
	}
}

func (s *CooperativeScheduler) executeTask(task *Task) {
	start := time.Now()
	task.SetState(TaskStateRunning)

	// Execute until task yields or completes
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Simulate cooperative execution
		time.Sleep(time.Millisecond * 10) // Simulate work
	}()

	// Wait for task to complete or yield
	select {
	case <-done:
		// Task completed or yielded
	case <-time.After(s.config.ResourceLimits.MaxCPUTime):
		// Force termination if task runs too long
		s.Unschedule(task.ID)
		return
	}

	elapsed := time.Since(start)
	task.AddCPUTime(elapsed)
	task.SetState(TaskStateReady)

	// Update metrics
	if s.config.MetricsEnabled {
		s.updateMetrics(task)
	}

	// Re-queue task if still ready
	if task.GetState() == TaskStateReady {
		select {
		case s.ready <- task:
		default:
		}
	}
}

func (s *CooperativeScheduler) updateMetrics(task *Task) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	if metrics, exists := s.metrics[task.ID]; exists {
		metrics.CPUTime = task.CPUTime
		metrics.MemoryUsage = task.MemoryUsage
		metrics.Switches++
		metrics.LastSwitch = time.Now()
	}
}

func (s *CooperativeScheduler) Yield(taskID uuid.UUID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, task := range s.tasks {
		if task.ID == taskID && task.GetState() == TaskStateRunning {
			task.SetState(TaskStateReady)
			select {
			case s.ready <- task:
			default:
			}
			return nil
		}
	}
	return ErrTaskNotFound
}