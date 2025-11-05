# API Reference

## Authentication

### POST /api/v1/auth/login
Login with credentials.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T12:00:00Z",
  "user": {
    "id": "user-123",
    "username": "admin",
    "roles": ["admin"]
  }
}
```

## Modules

### POST /api/v1/modules
Upload WASM module.

**Headers:**
- `Authorization: Bearer <token>`
- `Content-Type: application/wasm`

**Request Body:** Binary WASM data

**Response:**
```json
{
  "module_id": "mod-abc123",
  "name": "uploaded-module",
  "size": 1024,
  "functions": ["main", "add", "multiply"],
  "created_at": "2024-01-01T12:00:00Z"
}
```

### GET /api/v1/modules
List all modules.

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `filter`: Name filter

**Response:**
```json
{
  "modules": [
    {
      "id": "mod-abc123",
      "name": "math-module",
      "description": "Basic math operations",
      "size": 1024,
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

### POST /api/v1/modules/{id}/execute
Execute module function.

**Request:**
```json
{
  "function": "add",
  "args": [5, 3],
  "timeout": 30
}
```

**Response:**
```json
{
  "result": [8],
  "execution_time": 0.001,
  "memory_used": 1024,
  "status": "completed"
}
```

## Scheduler

### POST /api/v1/scheduler/tasks
Create scheduled task.

**Request:**
```json
{
  "module_id": "mod-abc123",
  "function": "main",
  "args": [],
  "priority": 5,
  "scheduler_type": "priority",
  "schedule": "0 */5 * * * *"
}
```

**Response:**
```json
{
  "task_id": "task-xyz789",
  "status": "queued",
  "created_at": "2024-01-01T12:00:00Z",
  "next_run": "2024-01-01T12:05:00Z"
}
```

### GET /api/v1/scheduler/tasks/{id}
Get task status.

**Response:**
```json
{
  "id": "task-xyz789",
  "module_id": "mod-abc123",
  "status": "running",
  "progress": 75,
  "started_at": "2024-01-01T12:00:00Z",
  "estimated_completion": "2024-01-01T12:01:00Z"
}
```

## Metrics

### GET /api/v1/metrics/runtime
Get real-time system metrics.

**Response:**
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "system": {
    "cpu_usage": 45.2,
    "memory_usage": 67.8,
    "active_modules": 5,
    "active_tasks": 12
  },
  "modules": {
    "mod-abc123": {
      "operations": 1500,
      "memory_usage": 2048,
      "cpu_time": 0.5,
      "last_execution": "2024-01-01T11:59:30Z"
    }
  }
}
```

### WebSocket /api/v1/metrics/stream
Real-time metrics stream.

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/metrics/stream');
ws.send(JSON.stringify({ token: 'bearer-token' }));
```

**Message Format:**
```json
{
  "type": "metrics_update",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "module_id": "mod-abc123",
    "operations": 1501,
    "memory_usage": 2048
  }
}
```

## Snapshots

### POST /api/v1/snapshots
Create system snapshot.

**Request:**
```json
{
  "name": "test-snapshot-1",
  "description": "Snapshot for regression testing",
  "modules": ["mod-abc123", "mod-def456"],
  "include_metrics": true
}
```

**Response:**
```json
{
  "snapshot_id": "snap-123abc",
  "name": "test-snapshot-1",
  "size": 5242880,
  "created_at": "2024-01-01T12:00:00Z",
  "checksum": "sha256:abc123..."
}
```

### POST /api/v1/snapshots/{id}/restore
Restore from snapshot.

**Response:**
```json
{
  "status": "success",
  "restored_modules": 2,
  "restored_at": "2024-01-01T12:00:00Z"
}
```

## Error Responses

All endpoints return consistent error format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid module format",
    "details": {
      "field": "wasm_data",
      "reason": "Invalid magic number"
    }
  },
  "request_id": "req-123abc"
}
```

## Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| /auth/login | 5 requests | 1 minute |
| /modules | 100 requests | 1 hour |
| /execute | 1000 requests | 1 hour |
| /metrics | 10000 requests | 1 hour |

## Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error