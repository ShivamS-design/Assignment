# Quick Start Guide

## Getting Started in 5 Minutes

### 1. Installation
```bash
# Download and start WASM-as-OS
curl -sSL https://get.wasm-as-os.com | bash
cd wasm-as-os
make start
```

### 2. Access the Interface
Open your browser to: `http://localhost:3000`

**Default Credentials:**
- Username: `admin`
- Password: `admin123`

### 3. Upload Your First Module

#### Create a Simple WASM Module
```rust
// hello.rs
#[no_mangle]
pub extern "C" fn greet(name_ptr: *const u8, name_len: usize) -> i32 {
    // Simple greeting function
    42
}

#[no_mangle]
pub extern "C" fn add(a: i32, b: i32) -> i32 {
    a + b
}
```

#### Compile to WASM
```bash
# Install Rust and wasm-pack
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
cargo install wasm-pack

# Compile
rustc --target wasm32-unknown-unknown -O hello.rs -o hello.wasm
```

#### Upload via Web Interface
1. Click **"Upload Module"** button
2. Select your `hello.wasm` file
3. Add description: "My first WASM module"
4. Click **"Upload"**

### 4. Execute Functions

#### Via Web Interface
1. Go to **"Modules"** tab
2. Find your uploaded module
3. Click **"Execute"**
4. Select function: `add`
5. Enter arguments: `[5, 3]`
6. Click **"Run"**

#### Via API
```bash
# Get authentication token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# Execute function
curl -X POST http://localhost:8080/api/v1/modules/YOUR_MODULE_ID/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"function":"add","args":[5,3]}'
```

### 5. Monitor Performance

#### Real-time Metrics
1. Go to **"Metrics"** tab
2. Enable **"Real-time Updates"**
3. Watch live performance data

#### View Execution History
1. Click **"History"** in metrics panel
2. Select time range
3. Export data as CSV/JSON

## Common Use Cases

### Web Assembly Calculator
```javascript
// calculator.js - compile to WASM
export function calculate(operation, a, b) {
    switch(operation) {
        case 'add': return a + b;
        case 'subtract': return a - b;
        case 'multiply': return a * b;
        case 'divide': return b !== 0 ? a / b : 0;
    }
}
```

### Image Processing Module
```rust
// image_processor.rs
#[no_mangle]
pub extern "C" fn resize_image(
    width: u32, 
    height: u32, 
    new_width: u32, 
    new_height: u32
) -> u32 {
    // Image resizing logic
    new_width * new_height
}

#[no_mangle]
pub extern "C" fn apply_filter(filter_type: u32, intensity: f32) -> u32 {
    // Filter application logic
    (intensity * 100.0) as u32
}
```

### Data Processing Pipeline
```python
# data_processor.py - compile with Pyodide
def process_data(data_array):
    """Process array of numbers"""
    return {
        'sum': sum(data_array),
        'average': sum(data_array) / len(data_array),
        'max': max(data_array),
        'min': min(data_array)
    }
```

## Scheduling Tasks

### One-time Execution
```json
{
  "module_id": "calc-module",
  "function": "calculate",
  "args": ["add", 10, 20],
  "priority": 5
}
```

### Recurring Tasks
```json
{
  "module_id": "data-processor",
  "function": "process_batch",
  "args": [],
  "schedule": "0 */15 * * * *",
  "scheduler_type": "priority"
}
```

### Batch Processing
```json
{
  "batch_id": "daily-reports",
  "tasks": [
    {
      "module_id": "report-gen",
      "function": "generate_sales_report",
      "args": ["2024-01-01"]
    },
    {
      "module_id": "report-gen", 
      "function": "generate_user_report",
      "args": ["2024-01-01"]
    }
  ]
}
```

## Security Best Practices

### Module Validation
- Always review WASM modules before upload
- Use static analysis results to assess security
- Set appropriate resource limits

### Access Control
```json
{
  "user": "developer1",
  "roles": ["developer"],
  "permissions": [
    "modules:read",
    "modules:execute",
    "tasks:create"
  ]
}
```

### Resource Limits
```yaml
limits:
  memory: 100MB
  execution_time: 30s
  cpu_usage: 80%
  network_access: false
  file_access: false
```

## Troubleshooting

### Module Upload Issues
**Problem:** "Invalid WASM format"
**Solution:** 
```bash
# Verify WASM file
wasm-validate your-module.wasm

# Check file size (max 10MB)
ls -lh your-module.wasm
```

### Execution Timeouts
**Problem:** Function execution times out
**Solution:**
1. Optimize your WASM code
2. Increase timeout in settings
3. Break large operations into smaller chunks

### Memory Errors
**Problem:** "Memory limit exceeded"
**Solution:**
```rust
// Optimize memory usage
#[no_mangle]
pub extern "C" fn process_data(size: usize) -> i32 {
    // Use stack allocation for small data
    if size < 1024 {
        let buffer = [0u8; 1024];
        // Process with buffer
    }
    0
}
```

## Next Steps

1. **Explore Advanced Features:**
   - Custom schedulers
   - Snapshot management
   - Performance optimization

2. **Integration:**
   - REST API usage
   - WebSocket connections
   - Webhook notifications

3. **Development:**
   - Create custom modules
   - Build processing pipelines
   - Implement business logic

4. **Administration:**
   - User management
   - Security configuration
   - Performance tuning

## Support

- **Documentation:** [docs.wasm-as-os.com](https://docs.wasm-as-os.com)
- **Community:** [forum.wasm-as-os.com](https://forum.wasm-as-os.com)
- **Issues:** [github.com/company/wasm-as-os/issues](https://github.com/company/wasm-as-os/issues)
- **Email:** support@wasm-as-os.com