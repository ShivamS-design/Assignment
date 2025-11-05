package api

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wasm-as-os/backend/internal/storage"
	"wasm-as-os/backend/internal/scheduler"
)

type Handlers struct {
	registry  *storage.ModuleRegistry
	scheduler scheduler.Scheduler
}

func NewHandlers(registry *storage.ModuleRegistry, sched scheduler.Scheduler) *Handlers {
	return &Handlers{
		registry:  registry,
		scheduler: sched,
	}
}

// Module Management
func (h *Handlers) LoadModule(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Version string `json:"version" binding:"required"`
		Binary  []byte `json:"binary" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	module := &storage.WASMModule{
		ID:        uuid.New(),
		Name:      req.Name,
		Version:   req.Version,
		Binary:    req.Binary,
		Size:      int64(len(req.Binary)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.registry.Store(module); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, module)
}

func (h *Handlers) ListModules(c *gin.Context) {
	modules := h.registry.List()
	c.JSON(http.StatusOK, gin.H{"modules": modules})
}

func (h *Handlers) DeleteModule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module ID"})
		return
	}

	if err := h.registry.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "module deleted"})
}

// Instance Control
func (h *Handlers) StartInstance(c *gin.Context) {
	var req struct {
		ModuleID uuid.UUID `json:"module_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance := &storage.RuntimeInstance{
		ID:        uuid.New(),
		ModuleID:  req.ModuleID,
		Status:    "starting",
		StartedAt: time.Now(),
	}

	if err := h.scheduler.Schedule(instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, instance)
}

func (h *Handlers) StopInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance ID"})
		return
	}

	if err := h.scheduler.Stop(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "instance stopped"})
}

func (h *Handlers) ListInstances(c *gin.Context) {
	instances := h.scheduler.List()
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

// Metrics
func (h *Handlers) GetMetrics(c *gin.Context) {
	instances := h.scheduler.List()
	
	metrics := gin.H{
		"timestamp":      time.Now(),
		"active_tasks":   len(instances),
		"total_modules":  len(h.registry.List()),
		"cpu_usage":      0.0,
		"memory_usage":   int64(0),
	}

	c.JSON(http.StatusOK, metrics)
}

// Health Check
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	})
}