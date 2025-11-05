package scheduler

import (
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
)

type RoundRobinScheduler struct {
	tasks       []*Task
	current     int
	config      SchedulerConfig
	metrics     map[uuid.UUID]*TaskMetrics
	running     bool
	stopCh      chan struct{}
	taskCh      chan *Task
	mu          sync.RWMutex
	metricsMu   sync.RWMutex
}

func NewRoundRobinScheduler(config SchedulerConfig) *RoundRobinScheduler {
	return &RoundRobinScheduler{
		tasks:   make([]*Task, 0),
		current: 0,
		config:  config,
		metrics: make(map[uuid.UUID]*TaskMetrics),
		stopCh:  make(chan struct{}),
		taskCh:  make(chan *Task, 100),
	}
}

func (s *RoundRobinScheduler) Schedule(task *Task) error {
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
	case s.taskCh <- task:
	default:
	}

	return nil
}

func (s *RoundRobinScheduler) Unschedule(taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.tasks {
		if task.ID == taskID {
			task.SetState(TaskStateTerminated)
			if task.Cancel != nil {
				task.Cancel()
			}
			
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			if s.current >= len(s.tasks) && len(s.tasks) > 0 {
				s.current = 0
			}
			
			s.metricsMu.Lock()
			delete(s.metrics, taskID)
			s.metricsMu.Unlock()
			
			return nil
		}
	}
	return ErrTaskNotFound
}

func (s *RoundRobinScheduler) GetNext() *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.tasks) == 0 {
		return nil
	}

	task := s.tasks[s.current]
	s.current = (s.current + 1) % len(s.tasks)

	if s.config.MetricsEnabled {
		s.updateMetrics(task)
	}

	return task
}

func (s *RoundRobinScheduler) GetTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*Task, len(s.tasks))
	copy(result, s.tasks)
	return result
}

func (s *RoundRobinScheduler) GetMetrics() map[uuid.UUID]*TaskMetrics {
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

func (s *RoundRobinScheduler) Start() error {
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

func (s *RoundRobinScheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	
	// Terminate all tasks
	for _, task := range s.GetTasks() {
		s.Unschedule(task.ID)
	}
	
	return nil
}

func (s *RoundRobinScheduler) schedulerLoop() {
	ticker := time.NewTicker(s.config.TimeSlice)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.executeTimeSlice()
		case task := <-s.taskCh:
			s.executeTask(task)
		}
	}
}

func (s *RoundRobinScheduler) executeTimeSlice() {
	task := s.GetNext()
	if task == nil {
		return
	}

	if task.GetState() == TaskStateTerminated {
		return
	}

	s.executeTask(task)
}

func (s *RoundRobinScheduler) executeTask(task *Task) {
	if task.GetState() == TaskStateTerminated {
		return
	}

	start := time.Now()
	task.SetState(TaskStateRunning)

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(task.Context, s.config.TimeSlice)
	defer cancel()

	// Execute task in goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Simulate WASM execution
		select {
		case <-ctx.Done():
		case <-time.After(s.config.TimeSlice):
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	// Update task metrics
	elapsed := time.Since(start)
	task.AddCPUTime(elapsed)
	task.SetState(TaskStateReady)

	// Check resource limits
	if task.CPUTime > s.config.ResourceLimits.MaxCPUTime {
		s.Unschedule(task.ID)
	}
}

func (s *RoundRobinScheduler) updateMetrics(task *Task) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	if metrics, exists := s.metrics[task.ID]; exists {
		metrics.CPUTime = task.CPUTime
		metrics.MemoryUsage = task.MemoryUsage
		metrics.Switches++
		metrics.LastSwitch = time.Now()
	}
}