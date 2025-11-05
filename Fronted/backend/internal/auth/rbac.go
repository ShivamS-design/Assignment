package auth

import (
	"fmt"
	"strings"
)

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleResearcher Role = "researcher"
	RoleDeveloper  Role = "developer"
	RoleStudent    Role = "student"
)

type Permission string

const (
	// Module permissions
	PermModuleCreate Permission = "module:create"
	PermModuleRead   Permission = "module:read"
	PermModuleUpdate Permission = "module:update"
	PermModuleDelete Permission = "module:delete"
	PermModuleExecute Permission = "module:execute"

	// Instance permissions
	PermInstanceCreate Permission = "instance:create"
	PermInstanceRead   Permission = "instance:read"
	PermInstanceControl Permission = "instance:control"
	PermInstanceDelete Permission = "instance:delete"

	// Debug permissions
	PermDebugAccess Permission = "debug:access"
	PermDebugControl Permission = "debug:control"

	// System permissions
	PermSystemConfig Permission = "system:config"
	PermSystemLogs   Permission = "system:logs"
	PermSystemMetrics Permission = "system:metrics"

	// User management
	PermUserCreate Permission = "user:create"
	PermUserRead   Permission = "user:read"
	PermUserUpdate Permission = "user:update"
	PermUserDelete Permission = "user:delete"

	// Audit permissions
	PermAuditRead Permission = "audit:read"
	PermAuditExport Permission = "audit:export"
)

type RBACManager struct {
	rolePermissions map[Role][]Permission
	userRoles       map[string]Role
}

func NewRBACManager() *RBACManager {
	rbac := &RBACManager{
		rolePermissions: make(map[Role][]Permission),
		userRoles:       make(map[string]Role),
	}
	rbac.initializeRoles()
	return rbac
}

func (r *RBACManager) initializeRoles() {
	// Admin - full access
	r.rolePermissions[RoleAdmin] = []Permission{
		PermModuleCreate, PermModuleRead, PermModuleUpdate, PermModuleDelete, PermModuleExecute,
		PermInstanceCreate, PermInstanceRead, PermInstanceControl, PermInstanceDelete,
		PermDebugAccess, PermDebugControl,
		PermSystemConfig, PermSystemLogs, PermSystemMetrics,
		PermUserCreate, PermUserRead, PermUserUpdate, PermUserDelete,
		PermAuditRead, PermAuditExport,
	}

	// Researcher - analysis and debugging
	r.rolePermissions[RoleResearcher] = []Permission{
		PermModuleRead, PermModuleExecute,
		PermInstanceCreate, PermInstanceRead, PermInstanceControl,
		PermDebugAccess, PermDebugControl,
		PermSystemMetrics,
		PermAuditRead,
	}

	// Developer - development and testing
	r.rolePermissions[RoleDeveloper] = []Permission{
		PermModuleCreate, PermModuleRead, PermModuleUpdate, PermModuleExecute,
		PermInstanceCreate, PermInstanceRead, PermInstanceControl,
		PermDebugAccess,
		PermSystemMetrics,
	}

	// Student - limited access
	r.rolePermissions[RoleStudent] = []Permission{
		PermModuleRead, PermModuleExecute,
		PermInstanceRead,
		PermSystemMetrics,
	}
}

func (r *RBACManager) HasPermission(role Role, permission Permission) bool {
	permissions, exists := r.rolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (r *RBACManager) HasAnyPermission(role Role, permissions []Permission) bool {
	for _, permission := range permissions {
		if r.HasPermission(role, permission) {
			return true
		}
	}
	return false
}

func (r *RBACManager) GetRolePermissions(role Role) []Permission {
	return r.rolePermissions[role]
}

func (r *RBACManager) CanAccessResource(role Role, resource, action string) bool {
	permission := Permission(fmt.Sprintf("%s:%s", resource, action))
	return r.HasPermission(role, permission)
}

func (r *RBACManager) FilterCommands(role Role, commands []string) []string {
	var allowed []string
	
	for _, cmd := range commands {
		if r.canExecuteCommand(role, cmd) {
			allowed = append(allowed, cmd)
		}
	}
	
	return allowed
}

func (r *RBACManager) canExecuteCommand(role Role, command string) bool {
	// Map CLI commands to permissions
	commandPermissions := map[string]Permission{
		"module create":    PermModuleCreate,
		"module list":      PermModuleRead,
		"module delete":    PermModuleDelete,
		"instance start":   PermInstanceCreate,
		"instance stop":    PermInstanceControl,
		"instance list":    PermInstanceRead,
		"debug start":      PermDebugAccess,
		"debug step":       PermDebugControl,
		"system config":    PermSystemConfig,
		"system logs":      PermSystemLogs,
		"user create":      PermUserCreate,
		"audit export":     PermAuditExport,
	}

	// Check exact match first
	if perm, exists := commandPermissions[command]; exists {
		return r.HasPermission(role, perm)
	}

	// Check prefix matches
	for cmdPattern, perm := range commandPermissions {
		if strings.HasPrefix(command, cmdPattern) {
			return r.HasPermission(role, perm)
		}
	}

	// Default deny
	return false
}

func (r *RBACManager) GetUIPermissions(role Role) UIPermissions {
	return UIPermissions{
		CanCreateModules:    r.HasPermission(role, PermModuleCreate),
		CanDeleteModules:    r.HasPermission(role, PermModuleDelete),
		CanControlInstances: r.HasPermission(role, PermInstanceControl),
		CanDebug:           r.HasPermission(role, PermDebugAccess),
		CanViewLogs:        r.HasPermission(role, PermSystemLogs),
		CanManageUsers:     r.HasPermission(role, PermUserCreate),
		CanViewAudit:       r.HasPermission(role, PermAuditRead),
		CanExportAudit:     r.HasPermission(role, PermAuditExport),
		CanConfigureSystem: r.HasPermission(role, PermSystemConfig),
	}
}

type UIPermissions struct {
	CanCreateModules    bool `json:"can_create_modules"`
	CanDeleteModules    bool `json:"can_delete_modules"`
	CanControlInstances bool `json:"can_control_instances"`
	CanDebug           bool `json:"can_debug"`
	CanViewLogs        bool `json:"can_view_logs"`
	CanManageUsers     bool `json:"can_manage_users"`
	CanViewAudit       bool `json:"can_view_audit"`
	CanExportAudit     bool `json:"can_export_audit"`
	CanConfigureSystem bool `json:"can_configure_system"`
}

type PermissionChecker struct {
	rbac *RBACManager
}

func NewPermissionChecker(rbac *RBACManager) *PermissionChecker {
	return &PermissionChecker{rbac: rbac}
}

func (pc *PermissionChecker) RequirePermission(role Role, permission Permission) error {
	if !pc.rbac.HasPermission(role, permission) {
		return fmt.Errorf("insufficient permissions: %s required", permission)
	}
	return nil
}

func (pc *PermissionChecker) RequireAnyPermission(role Role, permissions []Permission) error {
	if !pc.rbac.HasAnyPermission(role, permissions) {
		return fmt.Errorf("insufficient permissions: one of %v required", permissions)
	}
	return nil
}

func (pc *PermissionChecker) RequireRole(userRole Role, requiredRoles []Role) error {
	for _, role := range requiredRoles {
		if userRole == role {
			return nil
		}
	}
	return fmt.Errorf("insufficient role: one of %v required", requiredRoles)
}

func (pc *PermissionChecker) CanAccessEndpoint(role Role, method, path string) bool {
	// Map HTTP endpoints to permissions
	endpointPermissions := map[string]map[string]Permission{
		"/api/v1/modules": {
			"GET":    PermModuleRead,
			"POST":   PermModuleCreate,
			"DELETE": PermModuleDelete,
		},
		"/api/v1/instances": {
			"GET":    PermInstanceRead,
			"POST":   PermInstanceCreate,
			"DELETE": PermInstanceDelete,
		},
		"/api/v1/debug": {
			"GET":  PermDebugAccess,
			"POST": PermDebugControl,
		},
		"/api/v1/system": {
			"GET": PermSystemMetrics,
			"PUT": PermSystemConfig,
		},
		"/api/v1/users": {
			"GET":    PermUserRead,
			"POST":   PermUserCreate,
			"PUT":    PermUserUpdate,
			"DELETE": PermUserDelete,
		},
		"/api/v1/audit": {
			"GET": PermAuditRead,
		},
	}

	// Check exact path match
	if methods, exists := endpointPermissions[path]; exists {
		if perm, exists := methods[method]; exists {
			return pc.rbac.HasPermission(role, perm)
		}
	}

	// Check prefix matches for parameterized paths
	for endpointPath, methods := range endpointPermissions {
		if strings.HasPrefix(path, endpointPath) {
			if perm, exists := methods[method]; exists {
				return pc.rbac.HasPermission(role, perm)
			}
		}
	}

	// Default deny for unknown endpoints
	return false
}

func IsValidRole(role string) bool {
	validRoles := []Role{RoleAdmin, RoleResearcher, RoleDeveloper, RoleStudent}
	for _, validRole := range validRoles {
		if Role(role) == validRole {
			return true
		}
	}
	return false
}

func GetAllRoles() []Role {
	return []Role{RoleAdmin, RoleResearcher, RoleDeveloper, RoleStudent}
}

func GetRoleDescription(role Role) string {
	descriptions := map[Role]string{
		RoleAdmin:      "Full system access including user management and configuration",
		RoleResearcher: "Analysis and debugging capabilities with read access to modules",
		RoleDeveloper:  "Module development and testing with limited debugging access",
		RoleStudent:    "Basic read-only access for learning and exploration",
	}
	return descriptions[role]
}