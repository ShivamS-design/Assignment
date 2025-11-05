package api

import (
	"github.com/gin-gonic/gin"
	"wasm-as-os/backend/internal/auth"
)

func SetupDebugRoutes(r *gin.Engine, debugHandlers *DebugHandlers, authMiddleware *auth.AuthMiddleware) {
	debug := r.Group("/api/v1/debug")
	debug.Use(authMiddleware.RequireAuth())

	// Session Management
	sessions := debug.Group("/sessions")
	{
		sessions.POST("/", debugHandlers.CreateSession)
		sessions.GET("/", debugHandlers.ListSessions)
		sessions.GET("/:id", debugHandlers.GetSession)
		sessions.PUT("/:id/switch", debugHandlers.SwitchSession)
	}

	// Breakpoint Management
	breakpoints := debug.Group("/breakpoints")
	{
		breakpoints.POST("/", debugHandlers.SetBreakpoint)
		breakpoints.GET("/", debugHandlers.ListBreakpoints)
		breakpoints.DELETE("/:id", debugHandlers.ClearBreakpoint)
	}

	// Execution Control
	execution := debug.Group("/execution")
	{
		execution.POST("/step-into", debugHandlers.StepInto)
		execution.POST("/step-over", debugHandlers.StepOver)
		execution.POST("/step-out", debugHandlers.StepOut)
		execution.POST("/continue", debugHandlers.Continue)
	}

	// State Inspection
	inspect := debug.Group("/inspect")
	{
		inspect.GET("/state", debugHandlers.GetState)
		inspect.GET("/memory", debugHandlers.InspectMemory)
		inspect.GET("/callstack", debugHandlers.GetCallStack)
	}

	// Trace Management
	trace := debug.Group("/trace")
	{
		trace.GET("/", debugHandlers.GetTrace)
		trace.GET("/export", debugHandlers.ExportTrace)
	}

	// WebSocket endpoint for real-time debugging
	debug.GET("/ws", debugHandlers.HandleWebSocket)
}

func (h *DebugHandlers) HandleWebSocket(c *gin.Context) {
	// WebSocket implementation for real-time debugging
	c.JSON(200, gin.H{"message": "WebSocket endpoint - implementation needed"})
}