package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"wasm-as-os/backend/internal/audit"
)

type AuthHandlers struct {
	providers      map[string]AuthProvider
	sessionManager *SessionManager
	rbac           *RBACManager
	auditLogger    *audit.AuditLogger
}

func NewAuthHandlers(auditLogger *audit.AuditLogger) *AuthHandlers {
	handlers := &AuthHandlers{
		providers:      make(map[string]AuthProvider),
		sessionManager: NewSessionManager(),
		rbac:           NewRBACManager(),
		auditLogger:    auditLogger,
	}
	
	// Initialize default providers
	handlers.providers["local"] = NewLocalProvider("your-jwt-secret-key")
	handlers.providers["oauth"] = NewOAuthProvider("client-id", "client-secret", "redirect-url", "jwt-secret")
	handlers.providers["ldap"] = NewLDAPProvider("ldap://localhost", "dc=example,dc=com", "jwt-secret")
	
	return handlers
}

func (h *AuthHandlers) Login(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if creds.Provider == "" {
		creds.Provider = "local"
	}

	provider, exists := h.providers[creds.Provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auth provider"})
		return
	}

	user, err := provider.Authenticate(c.Request.Context(), creds)
	if err != nil {
		h.auditLogger.LogLogin(uuid.Nil, creds.Username, c.ClientIP(), c.GetHeader("User-Agent"), false, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Generate tokens
	if localProvider, ok := provider.(*LocalProvider); ok {
		tokens, err := localProvider.GenerateTokens(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
			return
		}

		// Create session
		session := h.sessionManager.CreateSession(user.ID, c.ClientIP(), c.GetHeader("User-Agent"))
		
		// Log successful login
		h.auditLogger.LogLogin(user.ID, user.Username, c.ClientIP(), c.GetHeader("User-Agent"), true, nil)

		// Get UI permissions
		uiPermissions := h.rbac.GetUIPermissions(user.Role)

		c.JSON(http.StatusOK, gin.H{
			"user":         user,
			"tokens":       tokens,
			"session_id":   session.ID,
			"permissions":  uiPermissions,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Provider not supported"})
	}
}

func (h *AuthHandlers) Logout(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID != "" {
		h.sessionManager.DeleteSession(sessionID)
	}

	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	
	if uid, ok := userID.(uuid.UUID); ok {
		if uname, ok := username.(string); ok {
			h.auditLogger.LogAction(uid, uname, "logout", "auth", audit.Details{}, c.ClientIP(), "", true, nil)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
		Provider     string `json:"provider"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.Provider == "" {
		req.Provider = "local"
	}

	provider, exists := h.providers[req.Provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auth provider"})
		return
	}

	tokens, err := provider.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (h *AuthHandlers) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	profile := gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	}

	if r, ok := role.(Role); ok {
		profile["permissions"] = h.rbac.GetUIPermissions(r)
	}

	c.JSON(http.StatusOK, profile)
}

func (h *AuthHandlers) CreateUser(c *gin.Context) {
	userRole, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.rbac.NewPermissionChecker(h.rbac).RequirePermission(userRole.(Role), PermUserCreate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	localProvider := h.providers["local"].(*LocalProvider)
	if err := localProvider.CreateUser(req.Username, req.Email, req.Password, Role(req.Role)); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Log user creation
	currentUserID, _ := c.Get("user_id")
	currentUsername, _ := c.Get("username")
	
	if uid, ok := currentUserID.(uuid.UUID); ok {
		if uname, ok := currentUsername.(string); ok {
			h.auditLogger.LogAction(uid, uname, "create_user", "user", 
				audit.Details{Parameters: req.Username}, c.ClientIP(), "", true, nil)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

func (h *AuthHandlers) ListUsers(c *gin.Context) {
	userRole, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.rbac.NewPermissionChecker(h.rbac).RequirePermission(userRole.(Role), PermUserRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// In production, would fetch from database
	users := []gin.H{
		{
			"id":       uuid.New(),
			"username": "admin",
			"email":    "admin@localhost",
			"role":     "admin",
			"active":   true,
		},
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *AuthHandlers) GetPermissions(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	permissions := h.rbac.GetRolePermissions(role.(Role))
	uiPermissions := h.rbac.GetUIPermissions(role.(Role))

	c.JSON(http.StatusOK, gin.H{
		"role":            role,
		"permissions":     permissions,
		"ui_permissions":  uiPermissions,
	})
}

func (h *AuthHandlers) ValidateSession(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	session, exists := h.sessionManager.GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"session":    session,
		"expires_at": session.ExpiresAt,
	})
}

func (h *AuthHandlers) GetRoles(c *gin.Context) {
	userRole, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.rbac.NewPermissionChecker(h.rbac).RequirePermission(userRole.(Role), PermUserRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	roles := make([]gin.H, 0)
	for _, role := range GetAllRoles() {
		roles = append(roles, gin.H{
			"name":        role,
			"description": GetRoleDescription(role),
			"permissions": h.rbac.GetRolePermissions(role),
		})
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *AuthHandlers) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		// Try each provider to validate token
		var claims *Claims
		var err error
		
		for _, provider := range h.providers {
			claims, err = provider.ValidateToken(tokenString)
			if err == nil {
				break
			}
		}

		if err != nil || claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("provider", claims.Provider)

		c.Next()
	}
}

func (h *AuthHandlers) RequirePermission(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		if !h.rbac.HasPermission(role.(Role), permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (h *AuthHandlers) RequireRole(roles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if userRole.(Role) == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient role"})
			c.Abort()
			return
		}

		c.Next()
	}
}