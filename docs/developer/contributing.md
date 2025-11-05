# Contributing Guidelines

## Getting Started

### Development Environment Setup
```bash
# Clone repository
git clone https://github.com/company/wasm-as-os.git
cd wasm-as-os

# Install dependencies
make install-deps

# Setup development environment
make dev-setup

# Run tests
make test
```

### Prerequisites
- **Go 1.21+** for backend development
- **Rust 1.70+** for WASM engine
- **Node.js 18+** for frontend
- **Docker** for containerization
- **PostgreSQL 15+** for database
- **Redis 7+** for caching

## Code Organization

```
wasm-as-os/
├── backend/                 # Go backend services
│   ├── cmd/                # Application entry points
│   ├── internal/           # Private application code
│   │   ├── api/           # REST API handlers
│   │   ├── auth/          # Authentication & authorization
│   │   ├── scheduler/     # Task scheduling
│   │   ├── metrics/       # Performance monitoring
│   │   └── security/      # Security middleware
│   ├── pkg/               # Public libraries
│   └── tests/             # Test suites
├── rust-engine/            # Rust WASM execution engine
│   ├── src/               # Source code
│   │   ├── parser/        # WASM binary parser
│   │   ├── vm/           # Virtual machine
│   │   ├── sandbox/      # Security sandbox
│   │   └── memory/       # Memory management
│   └── tests/            # Rust tests
├── frontend/              # Vue.js frontend
│   ├── src/              # Source code
│   │   ├── components/   # Vue components
│   │   ├── views/        # Page views
│   │   └── stores/       # State management
│   └── tests/            # Frontend tests
└── docs/                 # Documentation
```

## Development Workflow

### 1. Issue Creation
- Use issue templates for bugs and features
- Include detailed reproduction steps
- Add appropriate labels and milestones
- Assign to relevant team members

### 2. Branch Strategy
```bash
# Feature branches
git checkout -b feature/add-new-scheduler
git checkout -b bugfix/memory-leak-fix
git checkout -b hotfix/security-patch

# Branch naming convention
feature/short-description
bugfix/issue-description
hotfix/critical-fix
docs/documentation-update
```

### 3. Development Process
```bash
# Start development
git checkout develop
git pull origin develop
git checkout -b feature/my-feature

# Make changes
# ... code changes ...

# Run tests
make test
make lint

# Commit changes
git add .
git commit -m "feat: add new scheduler algorithm

- Implement weighted round-robin scheduler
- Add configuration options
- Include comprehensive tests
- Update documentation

Closes #123"
```

### 4. Pull Request Process
1. **Create PR** against `develop` branch
2. **Fill PR template** with detailed description
3. **Ensure CI passes** all checks
4. **Request reviews** from code owners
5. **Address feedback** and update code
6. **Merge** after approval

## Coding Standards

### Go Code Style
```go
// Package documentation
// Package scheduler implements task scheduling algorithms
package scheduler

import (
    "context"
    "fmt"
    "time"
    
    "github.com/company/wasm-as-os/pkg/types"
)

// Scheduler defines the interface for task schedulers
type Scheduler interface {
    // AddTask adds a new task to the scheduler queue
    AddTask(ctx context.Context, task *types.Task) error
    
    // NextTask returns the next task to execute
    NextTask() *types.Task
    
    // RemoveTask removes a task from the scheduler
    RemoveTask(taskID string) error
}

// RoundRobinScheduler implements round-robin scheduling
type RoundRobinScheduler struct {
    tasks    []*types.Task
    current  int
    mutex    sync.RWMutex
    maxTasks int
}

// NewRoundRobinScheduler creates a new round-robin scheduler
func NewRoundRobinScheduler(maxTasks int) *RoundRobinScheduler {
    return &RoundRobinScheduler{
        tasks:    make([]*types.Task, 0, maxTasks),
        maxTasks: maxTasks,
    }
}

// AddTask implements Scheduler.AddTask
func (s *RoundRobinScheduler) AddTask(ctx context.Context, task *types.Task) error {
    if task == nil {
        return fmt.Errorf("task cannot be nil")
    }
    
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if len(s.tasks) >= s.maxTasks {
        return fmt.Errorf("scheduler queue full")
    }
    
    s.tasks = append(s.tasks, task)
    return nil
}
```

### Rust Code Style
```rust
//! WASM parser module
//! 
//! This module provides functionality for parsing WebAssembly binary format
//! and validating module structure.

use std::collections::HashMap;
use std::io::{self, Read};

/// WASM magic number (0x6d736100)
const WASM_MAGIC: u32 = 0x6d736100;

/// WASM version (1)
const WASM_VERSION: u32 = 1;

/// Represents a parsed WASM module
#[derive(Debug, Clone)]
pub struct WasmModule {
    /// Module header information
    pub header: WasmHeader,
    /// Module sections
    pub sections: Vec<WasmSection>,
    /// Function signatures
    pub functions: HashMap<String, FunctionSignature>,
}

/// WASM module header
#[derive(Debug, Clone, PartialEq)]
pub struct WasmHeader {
    /// Magic number
    pub magic: u32,
    /// Version number
    pub version: u32,
}

/// WASM parser implementation
pub struct WasmParser {
    strict_mode: bool,
}

impl WasmParser {
    /// Creates a new WASM parser
    pub fn new() -> Self {
        Self {
            strict_mode: true,
        }
    }
    
    /// Parses a WASM module from bytes
    pub fn parse_module(&self, data: &[u8]) -> Result<WasmModule, ParseError> {
        if data.len() < 8 {
            return Err(ParseError::InvalidFormat("File too small".into()));
        }
        
        let header = self.parse_header(data)?;
        let sections = self.parse_sections(&data[8..])?;
        let functions = self.extract_functions(&sections)?;
        
        Ok(WasmModule {
            header,
            sections,
            functions,
        })
    }
    
    /// Parses the WASM header
    fn parse_header(&self, data: &[u8]) -> Result<WasmHeader, ParseError> {
        let magic = u32::from_le_bytes([data[0], data[1], data[2], data[3]]);
        let version = u32::from_le_bytes([data[4], data[5], data[6], data[7]]);
        
        if magic != WASM_MAGIC {
            return Err(ParseError::InvalidMagic(magic));
        }
        
        if self.strict_mode && version != WASM_VERSION {
            return Err(ParseError::UnsupportedVersion(version));
        }
        
        Ok(WasmHeader { magic, version })
    }
}

/// Parse error types
#[derive(Debug, thiserror::Error)]
pub enum ParseError {
    #[error("Invalid WASM format: {0}")]
    InvalidFormat(String),
    
    #[error("Invalid magic number: {0:#x}")]
    InvalidMagic(u32),
    
    #[error("Unsupported version: {0}")]
    UnsupportedVersion(u32),
    
    #[error("IO error: {0}")]
    Io(#[from] io::Error),
}
```

### JavaScript/Vue.js Style
```javascript
// ModuleUpload.vue
<template>
  <div class="module-upload">
    <div class="upload-area" @drop="handleDrop" @dragover.prevent>
      <input
        ref="fileInput"
        type="file"
        accept=".wasm"
        @change="handleFileSelect"
        class="file-input"
      >
      <div class="upload-content">
        <Icon name="upload" size="48" />
        <p>Drop WASM file here or click to browse</p>
      </div>
    </div>
    
    <div v-if="uploadProgress > 0" class="progress-bar">
      <div 
        class="progress-fill" 
        :style="{ width: `${uploadProgress}%` }"
      ></div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useModuleStore } from '@/stores/modules'
import { useNotificationStore } from '@/stores/notifications'

// Props and emits
const props = defineProps({
  maxSize: {
    type: Number,
    default: 10 * 1024 * 1024 // 10MB
  }
})

const emit = defineEmits(['uploaded', 'error'])

// Stores
const moduleStore = useModuleStore()
const notificationStore = useNotificationStore()

// Reactive state
const fileInput = ref(null)
const uploadProgress = ref(0)
const isUploading = ref(false)

// Computed properties
const canUpload = computed(() => !isUploading.value)

// Methods
const handleFileSelect = (event) => {
  const file = event.target.files[0]
  if (file) {
    uploadFile(file)
  }
}

const handleDrop = (event) => {
  event.preventDefault()
  const file = event.dataTransfer.files[0]
  if (file) {
    uploadFile(file)
  }
}

const uploadFile = async (file) => {
  if (!validateFile(file)) {
    return
  }
  
  try {
    isUploading.value = true
    uploadProgress.value = 0
    
    const result = await moduleStore.uploadModule(file, {
      onProgress: (progress) => {
        uploadProgress.value = progress
      }
    })
    
    notificationStore.success('Module uploaded successfully')
    emit('uploaded', result)
    
  } catch (error) {
    notificationStore.error(`Upload failed: ${error.message}`)
    emit('error', error)
    
  } finally {
    isUploading.value = false
    uploadProgress.value = 0
    fileInput.value.value = ''
  }
}

const validateFile = (file) => {
  if (!file.name.endsWith('.wasm')) {
    notificationStore.error('Please select a WASM file')
    return false
  }
  
  if (file.size > props.maxSize) {
    notificationStore.error('File size exceeds maximum limit')
    return false
  }
  
  return true
}
</script>

<style scoped>
.module-upload {
  @apply w-full max-w-md mx-auto;
}

.upload-area {
  @apply border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer transition-colors;
  
  &:hover {
    @apply border-blue-400 bg-blue-50;
  }
}

.file-input {
  @apply hidden;
}

.progress-bar {
  @apply w-full bg-gray-200 rounded-full h-2 mt-4;
}

.progress-fill {
  @apply bg-blue-600 h-2 rounded-full transition-all duration-300;
}
</style>
```

## Testing Guidelines

### Unit Tests
```go
// scheduler_test.go
func TestRoundRobinScheduler_AddTask(t *testing.T) {
    tests := []struct {
        name        string
        maxTasks    int
        tasks       []*types.Task
        expectError bool
    }{
        {
            name:     "add single task",
            maxTasks: 10,
            tasks:    []*types.Task{{ID: "task1"}},
        },
        {
            name:        "exceed capacity",
            maxTasks:    1,
            tasks:       []*types.Task{{ID: "task1"}, {ID: "task2"}},
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            scheduler := NewRoundRobinScheduler(tt.maxTasks)
            
            var err error
            for _, task := range tt.tasks {
                err = scheduler.AddTask(context.Background(), task)
                if err != nil {
                    break
                }
            }
            
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests
```go
// api_test.go
func TestModuleAPI_Integration(t *testing.T) {
    // Setup test server
    server := setupTestServer(t)
    defer server.Close()
    
    client := &http.Client{Timeout: 30 * time.Second}
    
    // Test module upload
    wasmData := testdata.SimpleAddModule()
    resp, err := client.Post(
        server.URL+"/api/v1/modules",
        "application/wasm",
        bytes.NewReader(wasmData),
    )
    require.NoError(t, err)
    defer resp.Body.Close()
    
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)
    
    moduleID := result["module_id"].(string)
    assert.NotEmpty(t, moduleID)
}
```

## Documentation Standards

### Code Documentation
```go
// Package-level documentation
// Package scheduler provides task scheduling functionality for WASM-as-OS.
//
// The scheduler package implements multiple scheduling algorithms including
// round-robin, priority-based, and cooperative scheduling. Each scheduler
// type is optimized for different use cases and workload patterns.
//
// Example usage:
//
//     scheduler := scheduler.NewRoundRobinScheduler(100)
//     task := &types.Task{
//         ID: "task-1",
//         ModuleID: "module-abc",
//         Priority: 5,
//     }
//     err := scheduler.AddTask(context.Background(), task)
//
package scheduler

// AddTask adds a new task to the scheduler queue.
//
// The task will be validated before being added to the queue. If the queue
// is full or the task is invalid, an error will be returned.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - task: The task to be scheduled (must not be nil)
//
// Returns:
//   - error: nil on success, error describing the failure otherwise
//
// Example:
//
//     task := &types.Task{ID: "task-1", Priority: 5}
//     err := scheduler.AddTask(ctx, task)
//     if err != nil {
//         log.Printf("Failed to add task: %v", err)
//     }
func (s *RoundRobinScheduler) AddTask(ctx context.Context, task *types.Task) error {
    // Implementation...
}
```

### API Documentation
```yaml
# OpenAPI specification
openapi: 3.0.0
info:
  title: WASM-as-OS API
  version: 1.0.0
  description: REST API for WASM module management and execution

paths:
  /api/v1/modules:
    post:
      summary: Upload WASM module
      description: |
        Uploads a WebAssembly module to the system. The module will be
        validated and stored for future execution.
      requestBody:
        required: true
        content:
          application/wasm:
            schema:
              type: string
              format: binary
      responses:
        '201':
          description: Module uploaded successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ModuleResponse'
        '400':
          description: Invalid module format
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
```

## Release Process

### Version Management
```bash
# Semantic versioning: MAJOR.MINOR.PATCH
# MAJOR: Breaking changes
# MINOR: New features (backward compatible)
# PATCH: Bug fixes (backward compatible)

# Create release branch
git checkout -b release/v1.2.0

# Update version files
echo "1.2.0" > VERSION
go mod edit -module=github.com/company/wasm-as-os/v1

# Update changelog
vim CHANGELOG.md

# Commit and tag
git commit -am "chore: bump version to 1.2.0"
git tag -a v1.2.0 -m "Release version 1.2.0"

# Push release
git push origin release/v1.2.0
git push origin v1.2.0
```

### Changelog Format
```markdown
# Changelog

## [1.2.0] - 2024-01-15

### Added
- New priority-based scheduler algorithm
- WebSocket support for real-time metrics
- Snapshot management functionality
- Static analysis for WASM modules

### Changed
- Improved memory management in WASM engine
- Enhanced error handling in API responses
- Updated authentication middleware

### Fixed
- Memory leak in task scheduler
- Race condition in metrics collector
- CORS configuration issues

### Security
- Added input validation for all API endpoints
- Implemented rate limiting
- Enhanced sandbox security

## [1.1.0] - 2024-01-01
...
```

## Community Guidelines

### Code of Conduct
- Be respectful and inclusive
- Provide constructive feedback
- Help newcomers learn and contribute
- Follow project guidelines and standards

### Communication Channels
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Slack**: Real-time team communication
- **Email**: security@wasm-as-os.com for security issues

### Recognition
Contributors are recognized in:
- CONTRIBUTORS.md file
- Release notes
- Annual contributor awards
- Conference speaking opportunities