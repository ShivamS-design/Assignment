package types

import (
	"time"
	"github.com/google/uuid"
)

// Module represents a WASM module
type Module struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Binary      []byte    `json:"-"`
	Hash        string    `json:"hash"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Instance represents a running WASM instance
type Instance struct {
	ID       uuid.UUID `json:"id"`
	ModuleID uuid.UUID `json:"module_id"`
	Status   string    `json:"status"`
	PID      int       `json:"pid"`
	Memory   int64     `json:"memory"`
	CPU      float64   `json:"cpu"`
	StartedAt time.Time `json:"started_at"`
}

// Metrics represents system metrics
type Metrics struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   int64     `json:"memory_usage"`
	ActiveTasks   int       `json:"active_tasks"`
	TotalModules  int       `json:"total_modules"`
}

// User represents a system user
type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Task represents a scheduled task
type Task struct {
	ID         uuid.UUID `json:"id"`
	ModuleID   uuid.UUID `json:"module_id"`
	Priority   int       `json:"priority"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}