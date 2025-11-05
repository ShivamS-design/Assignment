# Feature Tutorials

## Tutorial 1: Building a Math Calculator Module

### Step 1: Create the Rust Source
```rust
// calculator.rs
#[no_mangle]
pub extern "C" fn add(a: f64, b: f64) -> f64 {
    a + b
}

#[no_mangle]
pub extern "C" fn subtract(a: f64, b: f64) -> f64 {
    a - b
}

#[no_mangle]
pub extern "C" fn multiply(a: f64, b: f64) -> f64 {
    a * b
}

#[no_mangle]
pub extern "C" fn divide(a: f64, b: f64) -> f64 {
    if b != 0.0 { a / b } else { f64::NAN }
}

#[no_mangle]
pub extern "C" fn power(base: f64, exp: f64) -> f64 {
    base.powf(exp)
}

#[no_mangle]
pub extern "C" fn sqrt(x: f64) -> f64 {
    x.sqrt()
}
```

### Step 2: Compile to WASM
```bash
rustc --target wasm32-unknown-unknown -O calculator.rs -o calculator.wasm
```

### Step 3: Upload and Test
1. **Upload Module:**
   - Go to Modules → Upload
   - Select `calculator.wasm`
   - Name: "Advanced Calculator"
   - Description: "Mathematical operations with floating point precision"

2. **Test Functions:**
   ```json
   // Test addition
   {"function": "add", "args": [15.5, 24.3]}
   
   // Test division
   {"function": "divide", "args": [100, 7]}
   
   // Test power
   {"function": "power", "args": [2, 10]}
   ```

### Step 4: Create Scheduled Calculations
```json
{
  "name": "Daily Interest Calculation",
  "module_id": "calculator-module",
  "function": "multiply",
  "args": [1000, 0.05],
  "schedule": "0 0 9 * * *",
  "description": "Calculate daily interest at 9 AM"
}
```

## Tutorial 2: Image Processing Pipeline

### Step 1: Create Processing Module
```rust
// image_processor.rs
use std::slice;

#[repr(C)]
pub struct ImageData {
    width: u32,
    height: u32,
    channels: u32,
}

#[no_mangle]
pub extern "C" fn grayscale(
    data_ptr: *mut u8,
    width: u32,
    height: u32
) -> u32 {
    unsafe {
        let len = (width * height * 3) as usize;
        let data = slice::from_raw_parts_mut(data_ptr, len);
        
        for i in (0..len).step_by(3) {
            let r = data[i] as f32;
            let g = data[i + 1] as f32;
            let b = data[i + 2] as f32;
            
            let gray = (0.299 * r + 0.587 * g + 0.114 * b) as u8;
            
            data[i] = gray;
            data[i + 1] = gray;
            data[i + 2] = gray;
        }
    }
    
    width * height
}

#[no_mangle]
pub extern "C" fn blur(
    data_ptr: *mut u8,
    width: u32,
    height: u32,
    radius: u32
) -> u32 {
    // Gaussian blur implementation
    width * height
}
```

### Step 2: Web Interface Integration
```javascript
// Frontend integration
async function processImage(file, operation) {
    const formData = new FormData();
    formData.append('image', file);
    formData.append('operation', operation);
    
    const response = await fetch('/api/v1/image/process', {
        method: 'POST',
        headers: {
            'Authorization': `Bearer ${token}`
        },
        body: formData
    });
    
    return response.blob();
}

// Usage
document.getElementById('processBtn').onclick = async () => {
    const file = document.getElementById('imageInput').files[0];
    const processed = await processImage(file, 'grayscale');
    
    const url = URL.createObjectURL(processed);
    document.getElementById('result').src = url;
};
```

### Step 3: Batch Processing Setup
```json
{
  "batch_name": "Image Processing Pipeline",
  "input_directory": "/uploads/images",
  "output_directory": "/processed/images",
  "operations": [
    {
      "module_id": "image-processor",
      "function": "grayscale",
      "order": 1
    },
    {
      "module_id": "image-processor", 
      "function": "blur",
      "args": [5],
      "order": 2
    }
  ],
  "schedule": "0 2 * * *"
}
```

## Tutorial 3: Real-time Data Analytics

### Step 1: Analytics Module
```rust
// analytics.rs
use std::collections::HashMap;

#[no_mangle]
pub extern "C" fn calculate_statistics(
    data_ptr: *const f64,
    length: usize
) -> *mut StatResult {
    unsafe {
        let data = std::slice::from_raw_parts(data_ptr, length);
        
        let sum: f64 = data.iter().sum();
        let mean = sum / length as f64;
        
        let variance: f64 = data.iter()
            .map(|x| (x - mean).powi(2))
            .sum::<f64>() / length as f64;
        
        let std_dev = variance.sqrt();
        
        Box::into_raw(Box::new(StatResult {
            mean,
            std_dev,
            min: data.iter().fold(f64::INFINITY, |a, &b| a.min(b)),
            max: data.iter().fold(f64::NEG_INFINITY, |a, &b| a.max(b)),
            count: length as u32,
        }))
    }
}

#[repr(C)]
pub struct StatResult {
    mean: f64,
    std_dev: f64,
    min: f64,
    max: f64,
    count: u32,
}
```

### Step 2: WebSocket Integration
```javascript
// Real-time analytics dashboard
class AnalyticsDashboard {
    constructor() {
        this.ws = new WebSocket('ws://localhost:8080/api/v1/analytics/stream');
        this.chart = new Chart(document.getElementById('chart'), {
            type: 'line',
            data: { datasets: [] }
        });
    }
    
    connect() {
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.updateChart(data);
        };
        
        this.ws.onopen = () => {
            this.ws.send(JSON.stringify({
                token: localStorage.getItem('auth_token'),
                subscribe: ['statistics', 'trends']
            }));
        };
    }
    
    updateChart(data) {
        this.chart.data.datasets[0].data.push({
            x: new Date(data.timestamp),
            y: data.value
        });
        
        // Keep only last 100 points
        if (this.chart.data.datasets[0].data.length > 100) {
            this.chart.data.datasets[0].data.shift();
        }
        
        this.chart.update('none');
    }
}
```

### Step 3: Automated Reporting
```json
{
  "report_name": "Daily Analytics Summary",
  "data_sources": [
    {
      "module_id": "analytics-module",
      "function": "calculate_statistics",
      "data_query": "SELECT value FROM metrics WHERE date = CURRENT_DATE"
    }
  ],
  "output_format": "pdf",
  "recipients": ["admin@company.com", "analytics@company.com"],
  "schedule": "0 18 * * *"
}
```

## Tutorial 4: Custom Scheduler Implementation

### Step 1: Priority-based Scheduler
```go
// custom_scheduler.go
type CustomScheduler struct {
    queues map[int]*TaskQueue
    mutex  sync.RWMutex
}

func (s *CustomScheduler) AddTask(task *Task) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    priority := task.Priority
    if priority < 1 {
        priority = 1
    }
    if priority > 10 {
        priority = 10
    }
    
    if s.queues[priority] == nil {
        s.queues[priority] = NewTaskQueue()
    }
    
    return s.queues[priority].Enqueue(task)
}

func (s *CustomScheduler) NextTask() *Task {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    // Check highest priority queues first
    for priority := 10; priority >= 1; priority-- {
        if queue := s.queues[priority]; queue != nil {
            if task := queue.Dequeue(); task != nil {
                return task
            }
        }
    }
    
    return nil
}
```

### Step 2: Register Custom Scheduler
```json
{
  "scheduler_config": {
    "name": "custom_priority",
    "type": "plugin",
    "config": {
      "max_concurrent_tasks": 50,
      "priority_boost_interval": "5m",
      "starvation_prevention": true
    }
  }
}
```

### Step 3: Use Custom Scheduler
```json
{
  "task_name": "High Priority Processing",
  "module_id": "data-processor",
  "function": "process_urgent_data",
  "scheduler_type": "custom_priority",
  "priority": 9,
  "max_retries": 3
}
```

## Tutorial 5: Performance Optimization

### Step 1: Memory-Efficient WASM
```rust
// optimized.rs
#[no_mangle]
pub extern "C" fn process_large_dataset(
    data_ptr: *const u8,
    length: usize,
    chunk_size: usize
) -> u32 {
    unsafe {
        let data = std::slice::from_raw_parts(data_ptr, length);
        let mut processed = 0u32;
        
        // Process in chunks to avoid memory spikes
        for chunk in data.chunks(chunk_size) {
            processed += process_chunk(chunk);
            
            // Yield control periodically
            if processed % 1000 == 0 {
                std::hint::spin_loop();
            }
        }
        
        processed
    }
}

fn process_chunk(chunk: &[u8]) -> u32 {
    // Efficient chunk processing
    chunk.len() as u32
}
```

### Step 2: Monitoring Setup
```yaml
# monitoring.yml
performance_alerts:
  - name: "High Memory Usage"
    condition: "memory_usage > 80%"
    action: "scale_down"
    
  - name: "Slow Execution"
    condition: "avg_execution_time > 5s"
    action: "optimize_scheduler"
    
  - name: "Queue Backlog"
    condition: "pending_tasks > 1000"
    action: "add_workers"
```

### Step 3: Optimization Dashboard
```javascript
// Performance monitoring
class PerformanceMonitor {
    constructor() {
        this.metrics = new Map();
        this.thresholds = {
            memory: 80,
            cpu: 70,
            latency: 1000
        };
    }
    
    checkPerformance(moduleId) {
        const stats = this.getModuleStats(moduleId);
        
        if (stats.memoryUsage > this.thresholds.memory) {
            this.triggerAlert('memory', moduleId, stats.memoryUsage);
        }
        
        if (stats.avgLatency > this.thresholds.latency) {
            this.suggestOptimization(moduleId, 'latency');
        }
    }
    
    suggestOptimization(moduleId, issue) {
        const suggestions = {
            memory: "Consider using streaming processing or reducing data size",
            latency: "Optimize algorithms or increase resource allocation",
            cpu: "Review computational complexity or add caching"
        };
        
        this.showNotification(suggestions[issue]);
    }
}
```

## Best Practices Summary

### Module Development
- Keep functions small and focused
- Use appropriate data types
- Implement error handling
- Add comprehensive logging

### Performance
- Monitor resource usage
- Use efficient algorithms
- Implement caching where appropriate
- Profile and optimize bottlenecks

### Security
- Validate all inputs
- Use sandbox restrictions
- Implement proper authentication
- Regular security audits

### Maintenance
- Version your modules
- Document all functions
- Create comprehensive tests
- Monitor production usage