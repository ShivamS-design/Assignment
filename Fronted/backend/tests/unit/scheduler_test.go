package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wasm-as-os/internal/scheduler"
)

func TestRoundRobinScheduler(t *testing.T) {
	s := scheduler.NewRoundRobinScheduler(10*time.Millisecond, 100)
	
	task1 := &scheduler.Task{ID: "task1", Priority: 1}
	task2 := &scheduler.Task{ID: "task2", Priority: 1}
	
	require.NoError(t, s.AddTask(task1))
	require.NoError(t, s.AddTask(task2))
	
	next := s.NextTask()
	assert.Equal(t, "task1", next.ID)
	
	next = s.NextTask()
	assert.Equal(t, "task2", next.ID)
}

func TestPriorityScheduler(t *testing.T) {
	s := scheduler.NewPriorityScheduler(100)
	
	lowPrio := &scheduler.Task{ID: "low", Priority: 1}
	highPrio := &scheduler.Task{ID: "high", Priority: 10}
	
	require.NoError(t, s.AddTask(lowPrio))
	require.NoError(t, s.AddTask(highPrio))
	
	next := s.NextTask()
	assert.Equal(t, "high", next.ID)
}

func TestTaskManager(t *testing.T) {
	manager := scheduler.NewTaskManager()
	ctx := context.Background()
	
	task := &scheduler.Task{
		ID: "test-task",
		ModuleID: "test-module",
		Priority: 5,
	}
	
	err := manager.CreateTask(ctx, task)
	require.NoError(t, err)
	
	retrieved, err := manager.GetTask("test-task")
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	
	err = manager.RemoveTask("test-task")
	require.NoError(t, err)
	
	_, err = manager.GetTask("test-task")
	assert.Error(t, err)
}

func BenchmarkSchedulerThroughput(b *testing.B) {
	s := scheduler.NewRoundRobinScheduler(1*time.Millisecond, 1000)
	
	for i := 0; i < 100; i++ {
		task := &scheduler.Task{
			ID: fmt.Sprintf("task-%d", i),
			Priority: 1,
		}
		s.AddTask(task)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.NextTask()
	}
}