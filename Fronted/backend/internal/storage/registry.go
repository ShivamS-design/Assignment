package storage

import (
	"fmt"
	"sync"
	"github.com/google/uuid"
)

type ModuleRegistry struct {
	modules map[uuid.UUID]*WASMModule
	mu      sync.RWMutex
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[uuid.UUID]*WASMModule),
	}
}

func (r *ModuleRegistry) Store(module *WASMModule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.modules[module.ID] = module
	return nil
}

func (r *ModuleRegistry) Get(id uuid.UUID) (*WASMModule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	module, exists := r.modules[id]
	if !exists {
		return nil, fmt.Errorf("module not found")
	}
	return module, nil
}

func (r *ModuleRegistry) List() []*WASMModule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	modules := make([]*WASMModule, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
}

func (r *ModuleRegistry) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.modules[id]; !exists {
		return fmt.Errorf("module not found")
	}
	delete(r.modules, id)
	return nil
}