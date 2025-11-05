package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Auth endpoints
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"token":      "demo-token-123",
			"expires_at": "2024-12-31T23:59:59Z",
			"user": gin.H{
				"id":       "admin",
				"username": "admin",
				"roles":    []string{"admin"},
			},
		})
	})

	// Modules endpoints
	r.GET("/api/v1/modules", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"modules": []gin.H{},
			"total":   0,
			"page":    1,
			"limit":   20,
		})
	})

	r.POST("/api/v1/modules", func(c *gin.Context) {
		c.JSON(201, gin.H{
			"module_id": "demo-module-123",
			"name":      "uploaded-module",
			"size":      1024,
			"functions": []string{"main"},
		})
	})

	// Metrics endpoints
	r.GET("/api/v1/metrics/runtime", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"timestamp": "2024-01-01T12:00:00Z",
			"system": gin.H{
				"cpu_usage":      45.2,
				"memory_usage":   67.8,
				"active_modules": 0,
				"active_tasks":   0,
			},
			"modules": gin.H{},
		})
	})

	log.Printf("🚀 WASM-as-OS API Server starting on port %s", port)
	log.Printf("   Health: http://localhost:%s/health", port)
	log.Printf("   API: http://localhost:%s/api/v1/", port)
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}