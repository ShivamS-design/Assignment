# Simple Calculator Example

This example demonstrates how to create, compile, and deploy a basic calculator WASM module.

## Source Code

### Rust Implementation
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
    if b != 0.0 {
        a / b
    } else {
        f64::NAN
    }
}

#[no_mangle]
pub extern "C" fn power(base: f64, exp: f64) -> f64 {
    base.powf(exp)
}

#[no_mangle]
pub extern "C" fn sqrt(x: f64) -> f64 {
    if x >= 0.0 {
        x.sqrt()
    } else {
        f64::NAN
    }
}

// Advanced operations
#[no_mangle]
pub extern "C" fn factorial(n: i32) -> f64 {
    if n < 0 {
        return f64::NAN;
    }
    
    let mut result = 1.0;
    for i in 1..=n {
        result *= i as f64;
    }
    result
}

#[no_mangle]
pub extern "C" fn fibonacci(n: i32) -> i32 {
    if n <= 1 {
        return n;
    }
    
    let mut a = 0;
    let mut b = 1;
    
    for _ in 2..=n {
        let temp = a + b;
        a = b;
        b = temp;
    }
    
    b
}
```

## Compilation

### Using Rust
```bash
# Install Rust and WASM target
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
rustup target add wasm32-unknown-unknown

# Compile to WASM
rustc --target wasm32-unknown-unknown -O calculator.rs -o calculator.wasm

# Optimize (optional)
wasm-opt -O3 calculator.wasm -o calculator-optimized.wasm
```

### Using C/C++
```c
// calculator.c
#include <math.h>

double add(double a, double b) {
    return a + b;
}

double subtract(double a, double b) {
    return a - b;
}

double multiply(double a, double b) {
    return a * b;
}

double divide(double a, double b) {
    return b != 0.0 ? a / b : NAN;
}

double power(double base, double exp) {
    return pow(base, exp);
}

double sqrt_func(double x) {
    return x >= 0.0 ? sqrt(x) : NAN;
}
```

```bash
# Compile with Emscripten
emcc calculator.c -o calculator.wasm \
  -s EXPORTED_FUNCTIONS='["_add","_subtract","_multiply","_divide","_power","_sqrt_func"]' \
  -s WASM=1 -O3
```

## Deployment

### Upload via Web Interface
1. Open WASM-as-OS web interface
2. Navigate to "Modules" → "Upload"
3. Select `calculator.wasm` file
4. Fill in module details:
   - **Name**: "Simple Calculator"
   - **Description**: "Basic mathematical operations"
   - **Version**: "1.0.0"
5. Click "Upload"

### Upload via API
```bash
# Get authentication token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  jq -r '.token')

# Upload module
curl -X POST http://localhost:8080/api/v1/modules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/wasm" \
  --data-binary @calculator.wasm
```

### Upload via CLI
```bash
./wasm-as-os modules upload calculator.wasm \
  --name "Simple Calculator" \
  --description "Basic mathematical operations" \
  --version "1.0.0"
```

## Usage Examples

### Basic Operations
```bash
# Addition
curl -X POST http://localhost:8080/api/v1/modules/calculator/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"function":"add","args":[15.5, 24.3]}'

# Division
curl -X POST http://localhost:8080/api/v1/modules/calculator/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"function":"divide","args":[100, 7]}'

# Power
curl -X POST http://localhost:8080/api/v1/modules/calculator/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"function":"power","args":[2, 10]}'
```

### JavaScript Integration
```javascript
// Frontend integration
class Calculator {
    constructor(apiUrl, token) {
        this.apiUrl = apiUrl;
        this.token = token;
        this.moduleId = 'calculator';
    }
    
    async executeFunction(functionName, args) {
        const response = await fetch(`${this.apiUrl}/api/v1/modules/${this.moduleId}/execute`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${this.token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                function: functionName,
                args: args
            })
        });
        
        const result = await response.json();
        return result.result[0];
    }
    
    async add(a, b) {
        return this.executeFunction('add', [a, b]);
    }
    
    async subtract(a, b) {
        return this.executeFunction('subtract', [a, b]);
    }
    
    async multiply(a, b) {
        return this.executeFunction('multiply', [a, b]);
    }
    
    async divide(a, b) {
        return this.executeFunction('divide', [a, b]);
    }
    
    async power(base, exp) {
        return this.executeFunction('power', [base, exp]);
    }
    
    async sqrt(x) {
        return this.executeFunction('sqrt', [x]);
    }
    
    async factorial(n) {
        return this.executeFunction('factorial', [n]);
    }
    
    async fibonacci(n) {
        return this.executeFunction('fibonacci', [n]);
    }
}

// Usage
const calculator = new Calculator('http://localhost:8080', 'your-token');

// Basic operations
const sum = await calculator.add(10, 5);        // 15
const diff = await calculator.subtract(10, 5);  // 5
const product = await calculator.multiply(10, 5); // 50
const quotient = await calculator.divide(10, 5); // 2

// Advanced operations
const squared = await calculator.power(5, 2);    // 25
const root = await calculator.sqrt(25);          // 5
const fact = await calculator.factorial(5);      // 120
const fib = await calculator.fibonacci(10);      // 55
```

### Python Integration
```python
# calculator_client.py
import requests
import json

class CalculatorClient:
    def __init__(self, api_url, token):
        self.api_url = api_url
        self.token = token
        self.module_id = 'calculator'
        self.headers = {
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        }
    
    def _execute_function(self, function_name, args):
        url = f"{self.api_url}/api/v1/modules/{self.module_id}/execute"
        payload = {
            'function': function_name,
            'args': args
        }
        
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        
        result = response.json()
        return result['result'][0]
    
    def add(self, a, b):
        return self._execute_function('add', [a, b])
    
    def subtract(self, a, b):
        return self._execute_function('subtract', [a, b])
    
    def multiply(self, a, b):
        return self._execute_function('multiply', [a, b])
    
    def divide(self, a, b):
        return self._execute_function('divide', [a, b])
    
    def power(self, base, exp):
        return self._execute_function('power', [base, exp])
    
    def sqrt(self, x):
        return self._execute_function('sqrt', [x])
    
    def factorial(self, n):
        return self._execute_function('factorial', [n])
    
    def fibonacci(self, n):
        return self._execute_function('fibonacci', [n])

# Usage
if __name__ == "__main__":
    calc = CalculatorClient('http://localhost:8080', 'your-token')
    
    # Basic operations
    print(f"10 + 5 = {calc.add(10, 5)}")
    print(f"10 - 5 = {calc.subtract(10, 5)}")
    print(f"10 * 5 = {calc.multiply(10, 5)}")
    print(f"10 / 5 = {calc.divide(10, 5)}")
    
    # Advanced operations
    print(f"5^2 = {calc.power(5, 2)}")
    print(f"√25 = {calc.sqrt(25)}")
    print(f"5! = {calc.factorial(5)}")
    print(f"fib(10) = {calc.fibonacci(10)}")
```

## Scheduled Calculations

### Daily Interest Calculation
```json
{
  "name": "Daily Interest Calculation",
  "module_id": "calculator",
  "function": "multiply",
  "args": [1000, 0.05],
  "schedule": "0 0 9 * * *",
  "description": "Calculate daily interest at 9 AM",
  "output_format": "json",
  "notifications": {
    "email": ["finance@company.com"],
    "webhook": "https://finance-system.company.com/webhook"
  }
}
```

### Batch Processing
```json
{
  "batch_name": "Financial Calculations",
  "tasks": [
    {
      "name": "Calculate Interest",
      "module_id": "calculator",
      "function": "multiply",
      "args": [10000, 0.03],
      "order": 1
    },
    {
      "name": "Calculate Tax",
      "module_id": "calculator", 
      "function": "multiply",
      "args": ["${previous_result}", 0.15],
      "order": 2
    },
    {
      "name": "Calculate Total",
      "module_id": "calculator",
      "function": "add",
      "args": [10000, "${task_1_result}", "${task_2_result}"],
      "order": 3
    }
  ],
  "schedule": "0 18 * * 5",
  "output_destination": "/reports/weekly-calculations.json"
}
```

## Performance Optimization

### Optimized Rust Version
```rust
// optimized_calculator.rs
use std::hint::black_box;

// Use const generics for compile-time optimization
#[no_mangle]
pub extern "C" fn add_optimized(a: f64, b: f64) -> f64 {
    // Prevent compiler optimizations for benchmarking
    black_box(a + b)
}

// Vectorized operations for batch calculations
#[no_mangle]
pub extern "C" fn add_array(
    a_ptr: *const f64,
    b_ptr: *const f64,
    result_ptr: *mut f64,
    length: usize
) {
    unsafe {
        let a_slice = std::slice::from_raw_parts(a_ptr, length);
        let b_slice = std::slice::from_raw_parts(b_ptr, length);
        let result_slice = std::slice::from_raw_parts_mut(result_ptr, length);
        
        for i in 0..length {
            result_slice[i] = a_slice[i] + b_slice[i];
        }
    }
}

// Memory-efficient factorial using iterative approach
#[no_mangle]
pub extern "C" fn factorial_iterative(n: i32) -> f64 {
    if n < 0 || n > 170 {  // Prevent overflow
        return f64::NAN;
    }
    
    (1..=n).fold(1.0, |acc, x| acc * x as f64)
}
```

### Benchmarking
```bash
# Benchmark calculator functions
./wasm-as-os benchmark calculator \
  --function add \
  --args "[10.5, 20.3]" \
  --iterations 10000

# Compare optimized vs standard
./wasm-as-os benchmark calculator \
  --function factorial \
  --args "[10]" \
  --compare-with calculator-optimized
```

## Testing

### Unit Tests
```rust
#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_add() {
        assert_eq!(add(2.0, 3.0), 5.0);
        assert_eq!(add(-1.0, 1.0), 0.0);
        assert_eq!(add(0.0, 0.0), 0.0);
    }
    
    #[test]
    fn test_divide() {
        assert_eq!(divide(10.0, 2.0), 5.0);
        assert!(divide(10.0, 0.0).is_nan());
    }
    
    #[test]
    fn test_factorial() {
        assert_eq!(factorial(0), 1.0);
        assert_eq!(factorial(5), 120.0);
        assert!(factorial(-1).is_nan());
    }
    
    #[test]
    fn test_fibonacci() {
        assert_eq!(fibonacci(0), 0);
        assert_eq!(fibonacci(1), 1);
        assert_eq!(fibonacci(10), 55);
    }
}
```

### Integration Tests
```bash
# Test module upload and execution
./test-calculator.sh

# Load test
./wasm-as-os load-test calculator \
  --concurrent-users 100 \
  --duration 60s \
  --function add \
  --args "[random(1,100), random(1,100)]"
```

This example demonstrates a complete workflow from development to deployment and usage of a WASM module in WASM-as-OS.