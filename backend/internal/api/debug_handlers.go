package api

import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DebugHandlers struct {
	debugger *WasmDebugger
}

type WasmDebugger struct {
	sessions map[string]*DebugSession
	active   string
}

type DebugSession struct {
	ID          string                 `json:"id"`
	ModuleName  string                 `json:"module_name"`
	Breakpoints []Breakpoint          `json:"breakpoints"`
	State       DebugState            `json:"state"`
	Trace       []TraceEntry          `json:"trace"`
}

type Breakpoint struct {
	ID       uint32 `json:"id"`
	Function uint32 `json:"function"`
	Offset   uint32 `json:"offset"`
	Enabled  bool   `json:"enabled"`
	HitCount uint32 `json:"hit_count"`
}

type DebugState struct {
	IP         uint32      `json:"instruction_pointer"`
	SP         uint32      `json:"stack_pointer"`
	Locals     []int32     `json:"locals"`
	CallStack  []CallFrame `json:"call_stack"`
	MemorySize uint32      `json:"memory_size"`
}

type CallFrame struct {
	Function uint32 `json:"function"`
	IP       uint32 `json:"instruction_pointer"`
	Locals   uint32 `json:"locals_start"`
}

type TraceEntry struct {
	Timestamp uint64 `json:"timestamp"`
	Type      string `json:"type"`
	Function  uint32 `json:"function"`
	IP        uint32 `json:"instruction_pointer"`
	Details   string `json:"details"`
}

func NewDebugHandlers() *DebugHandlers {
	return &DebugHandlers{
		debugger: &WasmDebugger{
			sessions: make(map[string]*DebugSession),
		},
	}
}

// Session Management
func (h *DebugHandlers) CreateSession(c *gin.Context) {
	var req struct {
		ModuleName string `json:"module_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := uuid.New().String()
	session := &DebugSession{
		ID:          sessionID,
		ModuleName:  req.ModuleName,
		Breakpoints: make([]Breakpoint, 0),
		State:       DebugState{},
		Trace:       make([]TraceEntry, 0),
	}

	h.debugger.sessions[sessionID] = session
	h.debugger.active = sessionID

	c.JSON(http.StatusCreated, gin.H{"session_id": sessionID})
}

func (h *DebugHandlers) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	
	session, exists := h.debugger.sessions[sessionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *DebugHandlers) ListSessions(c *gin.Context) {
	sessions := make([]map[string]interface{}, 0)
	
	for id, session := range h.debugger.sessions {
		sessions = append(sessions, map[string]interface{}{
			"id":          id,
			"module_name": session.ModuleName,
			"active":      id == h.debugger.active,
		})
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *DebugHandlers) SwitchSession(c *gin.Context) {
	sessionID := c.Param("id")
	
	if _, exists := h.debugger.sessions[sessionID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	h.debugger.active = sessionID
	c.JSON(http.StatusOK, gin.H{"message": "session switched"})
}

// Breakpoint Management
func (h *DebugHandlers) SetBreakpoint(c *gin.Context) {
	var req struct {
		Function uint32 `json:"function" binding:"required"`
		Offset   uint32 `json:"offset" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	breakpoint := Breakpoint{
		ID:       uint32(len(session.Breakpoints) + 1),
		Function: req.Function,
		Offset:   req.Offset,
		Enabled:  true,
		HitCount: 0,
	}

	session.Breakpoints = append(session.Breakpoints, breakpoint)
	c.JSON(http.StatusCreated, breakpoint)
}

func (h *DebugHandlers) ClearBreakpoint(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid breakpoint ID"})
		return
	}

	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	for i, bp := range session.Breakpoints {
		if bp.ID == uint32(id) {
			session.Breakpoints = append(session.Breakpoints[:i], session.Breakpoints[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "breakpoint cleared"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "breakpoint not found"})
}

func (h *DebugHandlers) ListBreakpoints(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"breakpoints": session.Breakpoints})
}

// Execution Control
func (h *DebugHandlers) StepInto(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	// Simulate step into
	session.State.IP++
	h.addTraceEntry(session, "step_into", session.State.IP)

	c.JSON(http.StatusOK, session.State)
}

func (h *DebugHandlers) StepOver(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	// Simulate step over
	session.State.IP += 2
	h.addTraceEntry(session, "step_over", session.State.IP)

	c.JSON(http.StatusOK, session.State)
}

func (h *DebugHandlers) StepOut(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	// Simulate step out
	if len(session.State.CallStack) > 0 {
		session.State.CallStack = session.State.CallStack[:len(session.State.CallStack)-1]
	}
	session.State.IP += 5
	h.addTraceEntry(session, "step_out", session.State.IP)

	c.JSON(http.StatusOK, session.State)
}

func (h *DebugHandlers) Continue(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	// Simulate continue execution
	session.State.IP += 10
	h.addTraceEntry(session, "continue", session.State.IP)

	c.JSON(http.StatusOK, session.State)
}

// State Inspection
func (h *DebugHandlers) GetState(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	c.JSON(http.StatusOK, session.State)
}

func (h *DebugHandlers) InspectMemory(c *gin.Context) {
	addressStr := c.Query("address")
	lengthStr := c.Query("length")

	address, err := strconv.ParseUint(addressStr, 0, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address"})
		return
	}

	length, err := strconv.ParseUint(lengthStr, 10, 32)
	if err != nil {
		length = 64 // default length
	}

	// Simulate memory inspection
	memory := make([]byte, length)
	for i := range memory {
		memory[i] = byte((address + uint64(i)) & 0xFF)
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"length":  length,
		"data":    memory,
	})
}

func (h *DebugHandlers) GetCallStack(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"call_stack": session.State.CallStack})
}

// Trace Management
func (h *DebugHandlers) GetTrace(c *gin.Context) {
	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	trace := session.Trace
	if len(trace) > limit {
		trace = trace[len(trace)-limit:]
	}

	c.JSON(http.StatusOK, gin.H{"trace": trace})
}

func (h *DebugHandlers) ExportTrace(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	session := h.getActiveSession()
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active session"})
		return
	}

	switch format {
	case "json":
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=trace.json")
		c.JSON(http.StatusOK, session.Trace)
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=trace.csv")
		csv := "timestamp,type,function,ip,details\n"
		for _, entry := range session.Trace {
			csv += strconv.FormatUint(entry.Timestamp, 10) + "," +
				entry.Type + "," +
				strconv.FormatUint(uint64(entry.Function), 10) + "," +
				strconv.FormatUint(uint64(entry.IP), 10) + "," +
				entry.Details + "\n"
		}
		c.String(http.StatusOK, csv)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
	}
}

func (h *DebugHandlers) getActiveSession() *DebugSession {
	if h.debugger.active == "" {
		return nil
	}
	return h.debugger.sessions[h.debugger.active]
}

func (h *DebugHandlers) addTraceEntry(session *DebugSession, entryType string, ip uint32) {
	entry := TraceEntry{
		Timestamp: uint64(len(session.Trace) + 1),
		Type:      entryType,
		Function:  0, // Would be determined from IP
		IP:        ip,
		Details:   "",
	}

	session.Trace = append(session.Trace, entry)
	
	// Keep trace size manageable
	if len(session.Trace) > 10000 {
		session.Trace = session.Trace[1000:]
	}
}