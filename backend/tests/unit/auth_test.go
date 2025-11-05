package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wasm-as-os/internal/auth"
)

func TestJWTProvider(t *testing.T) {
	provider := auth.NewJWTProvider("test-secret", 24*time.Hour)
	
	user := &auth.User{
		ID: "user1",
		Username: "testuser",
		Roles: []string{"user"},
	}
	
	token, err := provider.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	
	claims, err := provider.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
}

func TestRBACManager(t *testing.T) {
	rbac := auth.NewRBACManager()
	
	// Test role permissions
	assert.True(t, rbac.HasPermission("admin", "modules:create"))
	assert.True(t, rbac.HasPermission("developer", "modules:read"))
	assert.False(t, rbac.HasPermission("viewer", "modules:create"))
	
	// Test resource access
	user := &auth.User{
		ID: "user1",
		Roles: []string{"developer"},
	}
	
	assert.True(t, rbac.CanAccessResource(user, "modules", "read"))
	assert.False(t, rbac.CanAccessResource(user, "system", "write"))
}

func TestOfflineProvider(t *testing.T) {
	provider := auth.NewOfflineProvider("testdata/users.json")
	
	user, err := provider.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, "admin", user.Username)
	assert.Contains(t, user.Roles, "admin")
	
	_, err = provider.Authenticate("admin", "wrongpass")
	assert.Error(t, err)
}