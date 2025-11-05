package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
)

type PriorityTask struct {
	*Task
	index int
}

type PriorityQueue []*PriorityTask

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority // Higher priority first
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityTask)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

type PriorityScheduler struct {
	queue     *PriorityQueue
	tasks     map[uuid.UUID]*PriorityTask
	config    SchedulerConfig
	metrics   map[uuid.UUID]*TaskMetrics
	running   bool
	stopCh    chan struct{}
	taskCh    chan *Task
	mu        sync.RWMutex
	metricsMu sync.RWMutex
}

func NewPriorityScheduler(config SchedulerConfig) *PriorityScheduler {
	pq := &PriorityQueue{}
	heap.Init(pq)
	
	return &PriorityScheduler{
		queue:   pq,
		tasks:   make(map[uuid.UUID]*PriorityTask),
		config:  config,
		metrics: make(map[uuid.UUID]*TaskMetrics),
		stopCh:  make(chan struct{}),
		taskCh:  make(chan *Task, 100),
	}
}

func (s *PriorityScheduler) Schedule(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.tasks) >= s.config.ResourceLimits.MaxTasks {
		return ErrMaxTasksReached
	}

	ptask := &PriorityTask{Task: task}
	s.tasks[task.ID] = ptask
	heap.Push(s.queue, ptask)
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

func (s *PriorityScheduler) Unschedule(taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ptask, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}

	ptask.SetState(TaskStateTerminated)
	if ptask.Cancel != nil {
		ptask.Cancel()
	}

	// Remove from heap
	if ptask.index >= 0 {
		heap.Remove(s.queue, ptask.index)
	}
	delete(s.tasks, taskID)

	s.metricsMu.Lock()
	delete(s.metrics, taskID)
	s.metricsMu.Unlock()

	return nil
}

func (s *PriorityScheduler) GetNext() *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.queue.Len() > 0 {
		ptask := heap.Pop(s.queue).(*PriorityTask)
		
		if ptask.GetState() == TaskStateTerminated {
			continue
		}

		// Re-add to queue for next scheduling
		heap.Push(s.queue, ptask)
		
		if s.config.MetricsEnabled {
			s.updateMetrics(ptask.Task)
		}

		return ptask.Task
	}

	return nil
}

func (s *PriorityScheduler) GetTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*Task, 0, len(s.tasks))
	for _, ptask := range s.tasks {
		result = append(result, ptask.Task)
	}
	return result
}

func (s *PriorityScheduler) GetMetrics() map[uuid.UUID]*TaskMetrics {
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

func (s *PriorityScheduler) Start() error {
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

func (s *PriorityScheduler) Stop() error {
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

func (s *PriorityScheduler) schedulerLoop() {
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

func (s *PriorityScheduler) executeTimeSlice() {
	task := s.GetNext()
	if task == nil {
		return
	}

	s.executeTask(task)
}

func (s *PriorityScheduler) executeTask(task *Task) {
	if task.GetState() == TaskStateTerminated {
		return
	}

	start := time.Now()
	task.SetState(TaskStateRunning)

	// Adjust time slice based on priority
	timeSlice := s.config.TimeSlice
	if task.Priority > 5 {
		timeSlice = timeSlice * 2 // High priority gets more time
	} else if task.Priority < 3 {
		timeSlice = timeSlice / 2 // Low priority gets less time
	}

	ctx, cancel := context.WithTimeout(task.Context, timeSlice)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
		case <-time.After(timeSlice):
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	elapsed := time.Since(start)
	task.AddCPUTime(elapsed)
	task.SetState(TaskStateReady)

	if task.CPUTime > s.config.ResourceLimits.MaxCPUTime {
		s.Unschedule(task.ID)
	}
}

func (s *PriorityScheduler) updateMetrics(task *Task) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	if metrics, exists := s.metrics[task.ID]; exists {
		metrics.CPUTime = task.CPUTime
		metrics.MemoryUsage = task.MemoryUsage
		metrics.Switches++
		metrics.LastSwitch = time.Now()
	}
}

func (s *PriorityScheduler) SetPriority(taskID uuid.UUID, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ptask, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}

	ptask.Priority = priority
	heap.Fix(s.queue, ptask.index)
	return nil
}