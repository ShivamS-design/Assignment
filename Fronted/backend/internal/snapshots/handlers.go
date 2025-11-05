package snapshots

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type SnapshotHandlers struct {
	manager *SnapshotManager
}

func NewSnapshotHandlers(manager *SnapshotManager) *SnapshotHandlers {
	return &SnapshotHandlers{
		manager: manager,
	}
}

func (h *SnapshotHandlers) CreateSnapshot(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		State       struct {
			InstructionPointer uint32            `json:"instruction_pointer"`
			StackPointer       uint32            `json:"stack_pointer"`
			Registers          map[string]uint32 `json:"registers"`
			Memory             []byte            `json:"memory"`
			CallStack          []CallFrame       `json:"call_stack"`
			Locals             []int32           `json:"locals"`
			Globals            []int32           `json:"globals"`
			ModuleID           string            `json:"module_id"`
			InstanceID         string            `json:"instance_id"`
		} `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state := &ExecutionState{
		InstructionPointer: req.State.InstructionPointer,
		StackPointer:       req.State.StackPointer,
		Registers:          req.State.Registers,
		Memory:             req.State.Memory,
		CallStack:          req.State.CallStack,
		Locals:             req.State.Locals,
		Globals:            req.State.Globals,
		ModuleID:           req.State.ModuleID,
		InstanceID:         req.State.InstanceID,
	}

	snapshot, err := h.manager.CreateSnapshot(req.Name, req.Description, state, req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          snapshot.ID,
		"name":        snapshot.Name,
		"created_at":  snapshot.CreatedAt,
		"size":        snapshot.Size,
		"compressed":  snapshot.Compressed,
		"checksum":    snapshot.Checksum,
	})
}

func (h *SnapshotHandlers) ListSnapshots(c *gin.Context) {
	tag := c.Query("tag")
	
	var snapshots []*SnapshotMetadata
	if tag != "" {
		snapshots = h.manager.FindSnapshotsByTag(tag)
	} else {
		snapshots = h.manager.ListSnapshots()
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

func (h *SnapshotHandlers) GetSnapshot(c *gin.Context) {
	id := c.Param("id")
	
	snapshot, err := h.manager.LoadSnapshot(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (h *SnapshotHandlers) DeleteSnapshot(c *gin.Context) {
	id := c.Param("id")
	
	if err := h.manager.DeleteSnapshot(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "snapshot deleted"})
}

func (h *SnapshotHandlers) RestoreSnapshot(c *gin.Context) {
	id := c.Param("id")
	
	state, err := h.manager.RestoreSnapshot(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "snapshot restored",
		"state":   state,
	})
}

func (h *SnapshotHandlers) TagSnapshot(c *gin.Context) {
	id := c.Param("id")
	
	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.TagSnapshot(id, req.Tags); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tags added"})
}

func (h *SnapshotHandlers) SearchSnapshots(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
		return
	}

	allSnapshots := h.manager.ListSnapshots()
	var results []*SnapshotMetadata

	queryLower := strings.ToLower(query)
	for _, snapshot := range allSnapshots {
		if strings.Contains(strings.ToLower(snapshot.Name), queryLower) ||
			strings.Contains(strings.ToLower(snapshot.Description), queryLower) {
			results = append(results, snapshot)
			continue
		}

		for _, tag := range snapshot.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, snapshot)
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": results})
}

func (h *SnapshotHandlers) GetSnapshotStats(c *gin.Context) {
	snapshots := h.manager.ListSnapshots()
	
	var totalSize int64
	tagCounts := make(map[string]int)
	
	for _, snapshot := range snapshots {
		totalSize += snapshot.Size
		for _, tag := range snapshot.Tags {
			tagCounts[tag]++
		}
	}

	stats := gin.H{
		"total_snapshots": len(snapshots),
		"total_size":      totalSize,
		"tag_counts":      tagCounts,
	}

	c.JSON(http.StatusOK, stats)
}