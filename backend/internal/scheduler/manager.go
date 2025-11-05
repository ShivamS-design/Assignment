package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/google/uuid"
)

type TaskManager struct {
	scheduler     Scheduler
	instances     map[uuid.UUID]*WasmInstance
	resourcePool  *ResourcePool
	config        SchedulerConfig
	running       bool
	stopCh        chan struct{}
	mu            sync.RWMutex
}

type WasmInstance struct {
	ID          uuid.UUID
	ModuleID    uuid.UUID
	Memory      int64
	State       string
	CreatedAt   time.Time
	LastActive  time.Time
}

type ResourcePool struct {
	totalMemory    int64
	usedMemory     int64
	totalCPU       time.Duration
	usedCPU        time.Duration
	activeTasks    int
	maxTasks       int
	mu             sync.RWMutex
}

func NewTaskManager(config SchedulerConfig) *TaskManager {
	var scheduler Scheduler
	
	switch config.Algorithm {
	case "round-robin":
		scheduler = NewRoundRobinScheduler(config)
	case "cooperative":
		scheduler = NewCooperativeScheduler(config)
	case "priority":
		scheduler = NewPriorityScheduler(config)
	default:
		scheduler = NewRoundRobinScheduler(config)
	}

	return &TaskManager{
		scheduler:    scheduler,
		instances:    make(map[uuid.UUID]*WasmInstance),
		resourcePool: NewResourcePool(config.ResourceLimits),
		config:       config,
		stopCh:       make(chan struct{}),
	}
}

func NewResourcePool(limits ResourceLimits) *ResourcePool {
	return &ResourcePool{
		totalMemory: limits.MaxMemory,
		totalCPU:    limits.MaxCPUTime,
		maxTasks:    limits.MaxTasks,
	}
}

func (tm *TaskManager) CreateTask(moduleID uuid.UUID, priority int) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.resourcePool.CanAllocate(0, 0) {
		return nil, ErrResourceExhausted
	}

	taskID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		ID:        taskID,
		ModuleID:  moduleID,
		State:     TaskStateReady,
		Priority:  priority,
		CreatedAt: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
	}

	instance := &WasmInstance{
		ID:         taskID,
		ModuleID:   moduleID,
		State:      "created",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	tm.instances[taskID] = instance
	
	if err := tm.scheduler.Schedule(task); err != nil {
		delete(tm.instances, taskID)
		cancel()
		return nil, err
	}

	tm.resourcePool.Allocate(0, 0)
	return task, nil
}

func (tm *TaskManager) DestroyTask(taskID uuid.UUID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	instance, exists := tm.instances[taskID]
	if !exists {
		return ErrTaskNotFound
	}

	if err := tm.scheduler.Unschedule(taskID); err != nil {
		return err
	}

	tm.resourcePool.Deallocate(instance.Memory, 0)
	delete(tm.instances, taskID)
	
	return nil
}

func (tm *TaskManager) GetTask(taskID uuid.UUID) (*Task, error) {
	tasks := tm.scheduler.GetTasks()
	for _, task := range tasks {
		if task.ID == taskID {
			return task, nil
		}
	}
	return nil, ErrTaskNotFound
}

func (tm *TaskManager) ListTasks() []*Task {
	return tm.scheduler.GetTasks()
}

func (tm *TaskManager) GetMetrics() map[uuid.UUID]*TaskMetrics {
	return tm.scheduler.GetMetrics()
}

func (tm *TaskManager) GetResourceUsage() ResourceUsage {
	tm.resourcePool.mu.RLock()
	defer tm.resourcePool.mu.RUnlock()

	return ResourceUsage{
		MemoryUsed:  tm.resourcePool.usedMemory,
		MemoryTotal: tm.resourcePool.totalMemory,
		CPUUsed:     tm.resourcePool.usedCPU,
		CPUTotal:    tm.resourcePool.totalCPU,
		ActiveTasks: tm.resourcePool.activeTasks,
		MaxTasks:    tm.resourcePool.maxTasks,
	}
}

func (tm *TaskManager) Start() error {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return ErrSchedulerRunning
	}
	tm.running = true
	tm.mu.Unlock()

	if err := tm.scheduler.Start(); err != nil {
		tm.mu.Lock()
		tm.running = false
		tm.mu.Unlock()
		return err
	}

	go tm.monitorResources()
	return nil
}

func (tm *TaskManager) Stop() error {
	tm.mu.Lock()
	if !tm.running {
		tm.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	tm.running = false
	tm.mu.Unlock()

	close(tm.stopCh)
	return tm.scheduler.Stop()
}

func (tm *TaskManager) monitorResources() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stopCh:
			return
		case <-ticker.C:
			tm.updateResourceUsage()
		}
	}
}

func (tm *TaskManager) updateResourceUsage() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var totalMemory int64
	var activeTasks int

	for _, instance := range tm.instances {
		totalMemory += instance.Memory
		if instance.State == "running" {
			activeTasks++
		}
	}

	tm.resourcePool.mu.Lock()
	tm.resourcePool.usedMemory = totalMemory
	tm.resourcePool.activeTasks = activeTasks
	tm.resourcePool.mu.Unlock()
}

func (rp *ResourcePool) CanAllocate(memory int64, cpu time.Duration) bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	return rp.usedMemory+memory <= rp.totalMemory &&
		rp.usedCPU+cpu <= rp.totalCPU &&
		rp.activeTasks < rp.maxTasks
}

func (rp *ResourcePool) Allocate(memory int64, cpu time.Duration) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.usedMemory += memory
	rp.usedCPU += cpu
	rp.activeTasks++
}

func (rp *ResourcePool) Deallocate(memory int64, cpu time.Duration) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.usedMemory -= memory
	rp.usedCPU -= cpu
	rp.activeTasks--
}

type ResourceUsage struct {
	MemoryUsed  int64         `json:"memory_used"`
	MemoryTotal int64         `json:"memory_total"`
	CPUUsed     time.Duration `json:"cpu_used"`
	CPUTotal    time.Duration `json:"cpu_total"`
	ActiveTasks int           `json:"active_tasks"`
	MaxTasks    int           `json:"max_tasks"`
}

func (ru ResourceUsage) MemoryUtilization() float64 {
	if ru.MemoryTotal == 0 {
		return 0
	}
	return float64(ru.MemoryUsed) / float64(ru.MemoryTotal)
}

func (ru ResourceUsage) CPUUtilization() float64 {
	if ru.CPUTotal == 0 {
		return 0
	}
	return float64(ru.CPUUsed) / float64(ru.CPUTotal)
}

func (ru ResourceUsage) TaskUtilization() float64 {
	if ru.MaxTasks == 0 {
		return 0
	}
	return float64(ru.ActiveTasks) / float64(ru.MaxTasks)
}