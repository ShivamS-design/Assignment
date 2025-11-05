package api

import (
	"github.com/gin-gonic/gin"
	"wasm-as-os/backend/internal/auth"
)

func SetupRoutes(r *gin.Engine, handlers *Handlers, authMiddleware *auth.AuthMiddleware) {
	// Health check (no auth required)
	r.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	v1.Use(auth.CORSMiddleware())

	// Public routes
	public := v1.Group("/")
	{
		public.GET("/health", handlers.HealthCheck)
		public.GET("/metrics", handlers.GetMetrics)
	}

	// Protected routes
	protected := v1.Group("/")
	protected.Use(authMiddleware.RequireAuth())
	{
		// Module management
		modules := protected.Group("/modules")
		{
			modules.POST("/", handlers.LoadModule)
			modules.GET("/", handlers.ListModules)
			modules.DELETE("/:id", handlers.DeleteModule)
		}

		// Instance control
		instances := protected.Group("/instances")
		{
			instances.POST("/", handlers.StartInstance)
			instances.GET("/", handlers.ListInstances)
			instances.DELETE("/:id", handlers.StopInstance)
		}
	}
}