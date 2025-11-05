package scheduler

import (
	"sync"
	"github.com/google/uuid"
	"wasm-as-os/backend/internal/storage"
)

type Scheduler interface {
	Schedule(instance *storage.RuntimeInstance) error
	Stop(instanceID uuid.UUID) error
	List() []*storage.RuntimeInstance
	GetNext() *storage.RuntimeInstance
}

type RoundRobinScheduler struct {
	instances []*storage.RuntimeInstance
	current   int
	mu        sync.RWMutex
}

func NewRoundRobinScheduler() *RoundRobinScheduler {
	return &RoundRobinScheduler{
		instances: make([]*storage.RuntimeInstance, 0),
		current:   0,
	}
}

func (s *RoundRobinScheduler) Schedule(instance *storage.RuntimeInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	instance.Status = "running"
	s.instances = append(s.instances, instance)
	return nil
}

func (s *RoundRobinScheduler) Stop(instanceID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, instance := range s.instances {
		if instance.ID == instanceID {
			instance.Status = "stopped"
			s.instances = append(s.instances[:i], s.instances[i+1:]...)
			if s.current >= len(s.instances) && len(s.instances) > 0 {
				s.current = 0
			}
			return nil
		}
	}
	return nil
}

func (s *RoundRobinScheduler) List() []*storage.RuntimeInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]*storage.RuntimeInstance, len(s.instances))
	copy(result, s.instances)
	return result
}

func (s *RoundRobinScheduler) GetNext() *storage.RuntimeInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.instances) == 0 {
		return nil
	}
	
	instance := s.instances[s.current]
	s.current = (s.current + 1) % len(s.instances)
	return instance
}