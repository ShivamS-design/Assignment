package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"wasm-as-os/internal/api"
	"wasm-as-os/internal/auth"
)

func setupTestServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	authProvider := auth.NewJWTProvider("test-secret", time.Hour)
	rbac := auth.NewRBACManager()
	
	api.SetupRoutes(router, authProvider, rbac)
	return router
}

func TestModuleUploadAPI(t *testing.T) {
	router := setupTestServer()
	
	// Create test WASM module
	wasmData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	
	req := httptest.NewRequest("POST", "/api/v1/modules", bytes.NewReader(wasmData))
	req.Header.Set("Content-Type", "application/wasm")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "module_id")
}

func TestModuleExecutionAPI(t *testing.T) {
	router := setupTestServer()
	
	// First upload a module
	moduleID := uploadTestModule(t, router)
	
	// Execute function
	payload := map[string]interface{}{
		"function": "main",
		"args": []interface{}{},
	}
	
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/modules/"+moduleID+"/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSchedulerAPI(t *testing.T) {
	router := setupTestServer()
	
	// Create task
	task := map[string]interface{}{
		"module_id": "test-module",
		"priority": 5,
		"scheduler_type": "round_robin",
	}
	
	body, _ := json.Marshal(task)
	req := httptest.NewRequest("POST", "/api/v1/scheduler/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	taskID := response["task_id"].(string)
	
	// Get task status
	req = httptest.NewRequest("GET", "/api/v1/scheduler/tasks/"+taskID, nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetricsAPI(t *testing.T) {
	router := setupTestServer()
	
	req := httptest.NewRequest("GET", "/api/v1/metrics/runtime", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var metrics map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &metrics)
	require.NoError(t, err)
	
	assert.Contains(t, metrics, "timestamp")
	assert.Contains(t, metrics, "modules")
}

func getTestToken(t *testing.T) string {
	provider := auth.NewJWTProvider("test-secret", time.Hour)
	user := &auth.User{
		ID: "test-user",
		Username: "testuser",
		Roles: []string{"developer"},
	}
	
	token, err := provider.GenerateToken(user)
	require.NoError(t, err)
	return token
}

func uploadTestModule(t *testing.T, router *gin.Engine) string {
	wasmData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	
	req := httptest.NewRequest("POST", "/api/v1/modules", bytes.NewReader(wasmData))
	req.Header.Set("Content-Type", "application/wasm")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t))
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response["module_id"].(string)
}