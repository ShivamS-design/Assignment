package storage

import (
	"time"
	"github.com/google/uuid"
)

type WASMModule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Version     string    `json:"version" db:"version"`
	Binary      []byte    `json:"-" db:"binary"`
	Hash        string    `json:"hash" db:"hash"`
	Size        int64     `json:"size" db:"size"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RuntimeInstance struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ModuleID  uuid.UUID `json:"module_id" db:"module_id"`
	Status    string    `json:"status" db:"status"`
	PID       int       `json:"pid" db:"pid"`
	Memory    int64     `json:"memory" db:"memory"`
	CPU       float64   `json:"cpu" db:"cpu"`
	StartedAt time.Time `json:"started_at" db:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty" db:"stopped_at"`
}

type ExecutionMetrics struct {
	ID           uuid.UUID `json:"id" db:"id"`
	InstanceID   uuid.UUID `json:"instance_id" db:"instance_id"`
	CPUUsage     float64   `json:"cpu_usage" db:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage" db:"memory_usage"`
	Instructions int64     `json:"instructions" db:"instructions"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

type UserSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"-" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}