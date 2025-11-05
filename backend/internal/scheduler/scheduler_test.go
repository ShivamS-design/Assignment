package scheduler

import (
	"context"
	"testing"
	"time"
	"github.com/google/uuid"
)

func TestRoundRobinScheduler(t *testing.T) {
	config := SchedulerConfig{
		Algorithm: "round-robin",
		TimeSlice: time.Millisecond * 100,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   10,
		},
		MetricsEnabled: true,
	}

	scheduler := NewRoundRobinScheduler(config)
	
	// Test scheduling tasks
	task1 := createTestTask(1)
	task2 := createTestTask(2)
	
	err := scheduler.Schedule(task1)
	if err != nil {
		t.Fatalf("Failed to schedule task1: %v", err)
	}
	
	err = scheduler.Schedule(task2)
	if err != nil {
		t.Fatalf("Failed to schedule task2: %v", err)
	}
	
	// Test getting next task
	next := scheduler.GetNext()
	if next == nil {
		t.Fatal("Expected to get a task")
	}
	
	// Test task list
	tasks := scheduler.GetTasks()
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}
	
	// Test unscheduling
	err = scheduler.Unschedule(task1.ID)
	if err != nil {
		t.Fatalf("Failed to unschedule task: %v", err)
	}
	
	tasks = scheduler.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task after unscheduling, got %d", len(tasks))
	}
}

func TestCooperativeScheduler(t *testing.T) {
	config := SchedulerConfig{
		Algorithm: "cooperative",
		TimeSlice: time.Millisecond * 100,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   10,
		},
		MetricsEnabled: true,
	}

	scheduler := NewCooperativeScheduler(config)
	
	task := createTestTask(1)
	err := scheduler.Schedule(task)
	if err != nil {
		t.Fatalf("Failed to schedule task: %v", err)
	}
	
	next := scheduler.GetNext()
	if next == nil {
		t.Fatal("Expected to get a task")
	}
	
	if next.ID != task.ID {
		t.Fatal("Got wrong task")
	}
}

func TestPriorityScheduler(t *testing.T) {
	config := SchedulerConfig{
		Algorithm: "priority",
		TimeSlice: time.Millisecond * 100,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   10,
		},
		MetricsEnabled: true,
	}

	scheduler := NewPriorityScheduler(config)
	
	// Create tasks with different priorities
	lowPriorityTask := createTestTask(1)
	highPriorityTask := createTestTask(10)
	
	// Schedule low priority first
	err := scheduler.Schedule(lowPriorityTask)
	if err != nil {
		t.Fatalf("Failed to schedule low priority task: %v", err)
	}
	
	// Schedule high priority second
	err = scheduler.Schedule(highPriorityTask)
	if err != nil {
		t.Fatalf("Failed to schedule high priority task: %v", err)
	}
	
	// High priority task should be returned first
	next := scheduler.GetNext()
	if next == nil {
		t.Fatal("Expected to get a task")
	}
	
	if next.Priority != 10 {
		t.Fatalf("Expected high priority task (10), got priority %d", next.Priority)
	}
}

func TestTaskManager(t *testing.T) {
	config := SchedulerConfig{
		Algorithm: "round-robin",
		TimeSlice: time.Millisecond * 100,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   10,
		},
		MetricsEnabled: true,
	}

	manager := NewTaskManager(config)
	
	// Test creating task
	moduleID := uuid.New()
	task, err := manager.CreateTask(moduleID, 5)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	
	if task.ModuleID != moduleID {
		t.Fatal("Task has wrong module ID")
	}
	
	// Test getting task
	retrieved, err := manager.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	
	if retrieved.ID != task.ID {
		t.Fatal("Retrieved wrong task")
	}
	
	// Test listing tasks
	tasks := manager.ListTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}
	
	// Test destroying task
	err = manager.DestroyTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to destroy task: %v", err)
	}
	
	tasks = manager.ListTasks()
	if len(tasks) != 0 {
		t.Fatalf("Expected 0 tasks after destruction, got %d", len(tasks))
	}
}

func TestResourcePool(t *testing.T) {
	limits := ResourceLimits{
		MaxMemory:  1024,
		MaxCPUTime: time.Second * 10,
		MaxTasks:   5,
	}
	
	pool := NewResourcePool(limits)
	
	// Test allocation
	if !pool.CanAllocate(512, time.Second*2) {
		t.Fatal("Should be able to allocate resources")
	}
	
	pool.Allocate(512, time.Second*2)
	
	// Test over-allocation
	if pool.CanAllocate(600, 0) {
		t.Fatal("Should not be able to over-allocate memory")
	}
	
	// Test deallocation
	pool.Deallocate(256, time.Second)
	
	if !pool.CanAllocate(300, 0) {
		t.Fatal("Should be able to allocate after deallocation")
	}
}

func TestSchedulerLifecycle(t *testing.T) {
	config := SchedulerConfig{
		Algorithm: "round-robin",
		TimeSlice: time.Millisecond * 10,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   10,
		},
		MetricsEnabled: true,
	}

	scheduler := NewRoundRobinScheduler(config)
	
	// Test starting scheduler
	err := scheduler.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	
	// Test double start
	err = scheduler.Start()
	if err != ErrSchedulerRunning {
		t.Fatal("Expected ErrSchedulerRunning on double start")
	}
	
	// Add a task and let it run briefly
	task := createTestTask(1)
	scheduler.Schedule(task)
	
	time.Sleep(time.Millisecond * 50)
	
	// Test stopping scheduler
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}
	
	// Test double stop
	err = scheduler.Stop()
	if err != ErrSchedulerNotRunning {
		t.Fatal("Expected ErrSchedulerNotRunning on double stop")
	}
}

func createTestTask(priority int) *Task {
	ctx, cancel := context.WithCancel(context.Background())
	return &Task{
		ID:        uuid.New(),
		ModuleID:  uuid.New(),
		State:     TaskStateReady,
		Priority:  priority,
		CreatedAt: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
	}
}

func BenchmarkRoundRobinScheduling(b *testing.B) {
	config := SchedulerConfig{
		Algorithm: "round-robin",
		TimeSlice: time.Microsecond * 100,
		ResourceLimits: ResourceLimits{
			MaxMemory:  1024 * 1024,
			MaxCPUTime: time.Second * 10,
			MaxTasks:   1000,
		},
		MetricsEnabled: false,
	}

	scheduler := NewRoundRobinScheduler(config)
	
	// Pre-populate with tasks
	for i := 0; i < 100; i++ {
		task := createTestTask(i % 10)
		scheduler.Schedule(task)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		scheduler.GetNext()
	}
}