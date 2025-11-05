package security

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type SecurityMiddleware struct {
	rateLimiters map[string]*rate.Limiter
	mu           sync.RWMutex
	config       SecurityConfig
}

type SecurityConfig struct {
	RateLimit struct {
		RequestsPerSecond int           `json:"requests_per_second"`
		BurstSize         int           `json:"burst_size"`
		WindowSize        time.Duration `json:"window_size"`
	} `json:"rate_limit"`
	
	CORS struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
		MaxAge         int      `json:"max_age"`
	} `json:"cors"`
	
	Security struct {
		ContentTypeNosniff   bool `json:"content_type_nosniff"`
		FrameOptions         bool `json:"frame_options"`
		XSSProtection        bool `json:"xss_protection"`
		StrictTransportSecurity bool `json:"strict_transport_security"`
	} `json:"security"`
}

func NewSecurityMiddleware(config SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		rateLimiters: make(map[string]*rate.Limiter),
		config:       config,
	}
}

func (sm *SecurityMiddleware) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		sm.mu.Lock()
		limiter, exists := sm.rateLimiters[clientIP]
		if !exists {
			limiter = rate.NewLimiter(
				rate.Limit(sm.config.RateLimit.RequestsPerSecond),
				sm.config.RateLimit.BurstSize,
			)
			sm.rateLimiters[clientIP] = limiter
		}
		sm.mu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": int(time.Second / rate.Limit(sm.config.RateLimit.RequestsPerSecond)),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (sm *SecurityMiddleware) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sm.config.Security.ContentTypeNosniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}
		
		if sm.config.Security.FrameOptions {
			c.Header("X-Frame-Options", "DENY")
		}
		
		if sm.config.Security.XSSProtection {
			c.Header("X-XSS-Protection", "1; mode=block")
		}
		
		if sm.config.Security.StrictTransportSecurity {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Header("X-Powered-By", "")
		c.Header("Server", "")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

func (sm *SecurityMiddleware) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		if sm.isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", strings.Join(sm.config.CORS.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(sm.config.CORS.AllowedHeaders, ", "))
		c.Header("Access-Control-Max-Age", strconv.Itoa(sm.config.CORS.MaxAge))
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (sm *SecurityMiddleware) isAllowedOrigin(origin string) bool {
	for _, allowed := range sm.config.CORS.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (sm *SecurityMiddleware) InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > 10*1024*1024 {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			c.Abort()
			return
		}

		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !sm.isValidContentType(contentType) {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "Unsupported content type",
				})
				c.Abort()
				return
			}
		}

		if err := sm.validateQueryParams(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid query parameters",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (sm *SecurityMiddleware) isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
	}

	for _, valid := range validTypes {
		if strings.HasPrefix(contentType, valid) {
			return true
		}
	}
	return false
}

func (sm *SecurityMiddleware) validateQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		if len(key) > 100 {
			return &ValidationError{Message: "Query parameter key too long"}
		}
		
		for _, value := range values {
			if len(value) > 1000 {
				return &ValidationError{Message: "Query parameter value too long"}
			}
			
			if sm.containsSuspiciousContent(value) {
				return &ValidationError{Message: "Suspicious content detected"}
			}
		}
	}
	return nil
}

func (sm *SecurityMiddleware) containsSuspiciousContent(value string) bool {
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"expression(",
		"../",
		"..\\",
	}

	lowerValue := strings.ToLower(value)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerValue, pattern) {
			return true
		}
	}
	return false
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func GetDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RateLimit: struct {
			RequestsPerSecond int           `json:"requests_per_second"`
			BurstSize         int           `json:"burst_size"`
			WindowSize        time.Duration `json:"window_size"`
		}{
			RequestsPerSecond: 100,
			BurstSize:         10,
			WindowSize:        time.Minute,
		},
		CORS: struct {
			AllowedOrigins []string `json:"allowed_origins"`
			AllowedMethods []string `json:"allowed_methods"`
			AllowedHeaders []string `json:"allowed_headers"`
			MaxAge         int      `json:"max_age"`
		}{
			AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Origin", "Content-Type", "Authorization"},
			MaxAge:         86400,
		},
		Security: struct {
			ContentTypeNosniff      bool `json:"content_type_nosniff"`
			FrameOptions            bool `json:"frame_options"`
			XSSProtection           bool `json:"xss_protection"`
			StrictTransportSecurity bool `json:"strict_transport_security"`
		}{
			ContentTypeNosniff:      true,
			FrameOptions:            true,
			XSSProtection:           true,
			StrictTransportSecurity: true,
		},
	}
}