package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *TestClient) Login(username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	
	body, _ := json.Marshal(payload)
	resp, err := c.client.Post(c.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	c.token = result["token"].(string)
	return nil
}

func (c *TestClient) UploadModule(wasmBytes []byte) (string, error) {
	req, _ := http.NewRequest("POST", c.baseURL+"/api/v1/modules", bytes.NewReader(wasmBytes))
	req.Header.Set("Content-Type", "application/wasm")
	req.Header.Set("Authorization", "Bearer "+c.token)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["module_id"].(string), nil
}

func TestCompleteWorkflow(t *testing.T) {
	client := NewTestClient("http://localhost:8080")
	
	// 1. Login
	err := client.Login("admin", "admin123")
	require.NoError(t, err)
	
	// 2. Upload WASM module
	wasmBytes := getComplexTestModule()
	moduleID, err := client.UploadModule(wasmBytes)
	require.NoError(t, err)
	assert.NotEmpty(t, moduleID)
	
	// 3. Create scheduled task
	taskID := createScheduledTask(t, client, moduleID)
	
	// 4. Monitor execution
	monitorExecution(t, client, taskID)
	
	// 5. Check metrics
	checkMetrics(t, client, moduleID)
	
	// 6. Create snapshot
	snapshotID := createSnapshot(t, client, moduleID)
	
	// 7. Restore from snapshot
	restoreSnapshot(t, client, snapshotID)
}

func TestMultiUserScenario(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan error, 3)
	
	users := []struct {
		username string
		password string
		role     string
	}{
		{"admin", "admin123", "admin"},
		{"dev1", "dev123", "developer"},
		{"viewer1", "view123", "viewer"},
	}
	
	for _, user := range users {
		wg.Add(1)
		go func(u struct{ username, password, role string }) {
			defer wg.Done()
			
			client := NewTestClient("http://localhost:8080")
			err := client.Login(u.username, u.password)
			if err != nil {
				results <- err
				return
			}
			
			// Test role-based access
			switch u.role {
			case "admin":
				err = testAdminOperations(client)
			case "developer":
				err = testDeveloperOperations(client)
			case "viewer":
				err = testViewerOperations(client)
			}
			
			results <- err
		}(user)
	}
	
	wg.Wait()
	close(results)
	
	for err := range results {
		assert.NoError(t, err)
	}
}

func TestPerformanceBenchmark(t *testing.T) {
	client := NewTestClient("http://localhost:8080")
	client.Login("admin", "admin123")
	
	// Upload performance test module
	wasmBytes := getFibonacciModule()
	moduleID, err := client.UploadModule(wasmBytes)
	require.NoError(t, err)
	
	// Benchmark function execution
	start := time.Now()
	iterations := 100
	
	for i := 0; i < iterations; i++ {
		executeFunction(t, client, moduleID, "fibonacci", []int{20})
	}
	
	duration := time.Since(start)
	avgTime := duration / time.Duration(iterations)
	
	t.Logf("Average execution time: %v", avgTime)
	assert.Less(t, avgTime, 100*time.Millisecond, "Function execution too slow")
}

func TestResourceExhaustion(t *testing.T) {
	client := NewTestClient("http://localhost:8080")
	client.Login("admin", "admin123")
	
	// Upload memory-intensive module
	wasmBytes := getMemoryTestModule()
	moduleID, err := client.UploadModule(wasmBytes)
	require.NoError(t, err)
	
	// Try to exhaust memory
	_, err = executeFunction(t, client, moduleID, "allocate_large", []int{1024 * 1024 * 100}) // 100MB
	assert.Error(t, err, "Should fail due to memory limits")
	
	// Verify system is still responsive
	_, err = executeFunction(t, client, moduleID, "simple_add", []int{1, 2})
	assert.NoError(t, err, "System should recover after resource exhaustion")
}

func TestWebSocketMetrics(t *testing.T) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://localhost:8080/api/v1/metrics/stream", nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Send authentication
	authMsg := map[string]string{"token": getTestToken()}
	conn.WriteJSON(authMsg)
	
	// Listen for metrics
	done := make(chan bool)
	go func() {
		defer close(done)
		
		for i := 0; i < 5; i++ {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				t.Errorf("WebSocket read error: %v", err)
				return
			}
			
			assert.Contains(t, msg, "timestamp")
			assert.Contains(t, msg, "metrics")
		}
	}()
	
	// Generate some activity
	client := NewTestClient("http://localhost:8080")
	client.Login("admin", "admin123")
	
	wasmBytes := getSimpleTestModule()
	moduleID, _ := client.UploadModule(wasmBytes)
	executeFunction(t, client, moduleID, "test", []int{})
	
	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("WebSocket metrics test timed out")
	}
}

func createScheduledTask(t *testing.T, client *TestClient, moduleID string) string {
	payload := map[string]interface{}{
		"module_id": moduleID,
		"function": "main",
		"args": []interface{}{},
		"priority": 5,
		"scheduler_type": "round_robin",
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", client.baseURL+"/api/v1/scheduler/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.token)
	
	resp, err := client.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["task_id"].(string)
}

func monitorExecution(t *testing.T, client *TestClient, taskID string) {
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", client.baseURL+"/api/v1/scheduler/tasks/"+taskID, nil)
		req.Header.Set("Authorization", "Bearer "+client.token)
		
		resp, err := client.client.Do(req)
		require.NoError(t, err)
		
		var task map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&task)
		resp.Body.Close()
		
		if task["status"] == "completed" {
			return
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	t.Fatal("Task did not complete in time")
}

func checkMetrics(t *testing.T, client *TestClient, moduleID string) {
	req, _ := http.NewRequest("GET", client.baseURL+"/api/v1/metrics/modules/"+moduleID, nil)
	req.Header.Set("Authorization", "Bearer "+client.token)
	
	resp, err := client.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var metrics map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&metrics)
	
	assert.Contains(t, metrics, "operations")
	assert.Contains(t, metrics, "memory_usage")
}

func createSnapshot(t *testing.T, client *TestClient, moduleID string) string {
	payload := map[string]interface{}{
		"module_id": moduleID,
		"name": "test-snapshot",
		"description": "Test snapshot for e2e testing",
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", client.baseURL+"/api/v1/snapshots", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.token)
	
	resp, err := client.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["snapshot_id"].(string)
}

func restoreSnapshot(t *testing.T, client *TestClient, snapshotID string) {
	req, _ := http.NewRequest("POST", client.baseURL+"/api/v1/snapshots/"+snapshotID+"/restore", nil)
	req.Header.Set("Authorization", "Bearer "+client.token)
	
	resp, err := client.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func executeFunction(t *testing.T, client *TestClient, moduleID, function string, args []int) (interface{}, error) {
	payload := map[string]interface{}{
		"function": function,
		"args": args,
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", client.baseURL+"/api/v1/modules/"+moduleID+"/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.token)
	
	resp, err := client.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("execution failed with status %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["result"], nil
}

// Test module generators
func getComplexTestModule() []byte {
	// Returns a WASM module with multiple functions
	return []byte{/* complex WASM bytecode */}
}

func getFibonacciModule() []byte {
	// Returns a WASM module with fibonacci function
	return []byte{/* fibonacci WASM bytecode */}
}

func getMemoryTestModule() []byte {
	// Returns a WASM module for memory testing
	return []byte{/* memory test WASM bytecode */}
}

func getSimpleTestModule() []byte {
	// Returns a simple WASM module
	return []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
}

func getTestToken() string {
	// Returns a valid test token
	return "test-token"
}