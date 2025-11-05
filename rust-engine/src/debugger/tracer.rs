use std::collections::VecDeque;
use std::time::{Duration, Instant};

#[derive(Debug, Clone)]
pub struct TraceEntry {
    pub timestamp: Instant,
    pub instruction_pointer: u32,
    pub opcode: u8,
    pub args: Vec<u32>,
    pub stack_depth: u32,
    pub execution_time: Duration,
}

#[derive(Debug, Clone)]
pub struct SyscallTrace {
    pub timestamp: Instant,
    pub name: String,
    pub args: Vec<u32>,
    pub result: Option<u32>,
    pub duration: Duration,
}

#[derive(Debug, Clone)]
pub struct FunctionCall {
    pub timestamp: Instant,
    pub function_index: u32,
    pub caller_ip: u32,
    pub args: Vec<i32>,
    pub duration: Option<Duration>,
}

#[derive(Debug)]
pub struct ExecutionTrace {
    pub instructions: VecDeque<TraceEntry>,
    pub syscalls: VecDeque<SyscallTrace>,
    pub function_calls: VecDeque<FunctionCall>,
    pub hotspots: Vec<Hotspot>,
    max_entries: usize,
}

#[derive(Debug, Clone)]
pub struct Hotspot {
    pub function_index: u32,
    pub instruction_offset: u32,
    pub hit_count: u64,
    pub total_time: Duration,
    pub avg_time: Duration,
}

#[derive(Debug)]
pub struct ExecutionTracer {
    trace: ExecutionTrace,
    enabled: bool,
    start_time: Option<Instant>,
    current_function: Option<u32>,
    call_stack: Vec<FunctionCall>,
    hotspot_map: std::collections::HashMap<u32, Hotspot>,
}

impl ExecutionTracer {
    pub fn new() -> Self {
        Self {
            trace: ExecutionTrace {
                instructions: VecDeque::with_capacity(10000),
                syscalls: VecDeque::with_capacity(1000),
                function_calls: VecDeque::with_capacity(1000),
                hotspots: Vec::new(),
                max_entries: 10000,
            },
            enabled: false,
            start_time: None,
            current_function: None,
            call_stack: Vec::new(),
            hotspot_map: std::collections::HashMap::new(),
        }
    }

    pub fn start(&mut self) {
        self.enabled = true;
        self.start_time = Some(Instant::now());
    }

    pub fn stop(&mut self) {
        self.enabled = false;
        self.update_hotspots();
    }

    pub fn trace_instruction(&mut self) {
        if !self.enabled {
            return;
        }

        let start = Instant::now();
        
        // Simulate instruction execution
        let entry = TraceEntry {
            timestamp: start,
            instruction_pointer: 0, // Would be actual IP
            opcode: 0x20, // i32.const
            args: vec![42],
            stack_depth: self.call_stack.len() as u32,
            execution_time: start.elapsed(),
        };

        self.add_instruction_trace(entry);
        self.update_hotspot(0, 0, start.elapsed());
    }

    pub fn trace_syscall(&mut self, name: &str, args: &[u32]) -> Option<u32> {
        if !self.enabled {
            return None;
        }

        let start = Instant::now();
        
        // Simulate syscall execution
        let result = match name {
            "wasm_get_time" => Some(42),
            "wasm_random" => Some(123),
            _ => Some(0),
        };

        let syscall = SyscallTrace {
            timestamp: start,
            name: name.to_string(),
            args: args.to_vec(),
            result,
            duration: start.elapsed(),
        };

        self.add_syscall_trace(syscall);
        result
    }

    pub fn trace_function_call(&mut self, function_index: u32, caller_ip: u32, args: &[i32]) {
        if !self.enabled {
            return;
        }

        let call = FunctionCall {
            timestamp: Instant::now(),
            function_index,
            caller_ip,
            args: args.to_vec(),
            duration: None,
        };

        self.call_stack.push(call.clone());
        self.current_function = Some(function_index);
        self.add_function_call_trace(call);
    }

    pub fn trace_function_return(&mut self, return_value: Option<i32>) {
        if !self.enabled || self.call_stack.is_empty() {
            return;
        }

        if let Some(mut call) = self.call_stack.pop() {
            call.duration = Some(call.timestamp.elapsed());
            
            // Update function call trace with duration
            if let Some(last_call) = self.trace.function_calls.back_mut() {
                if last_call.function_index == call.function_index {
                    last_call.duration = call.duration;
                }
            }
        }

        self.current_function = self.call_stack.last().map(|c| c.function_index);
    }

    pub fn get_trace(&self) -> &ExecutionTrace {
        &self.trace
    }

    pub fn get_call_graph(&self) -> CallGraph {
        let mut graph = CallGraph::new();
        
        for call in &self.trace.function_calls {
            graph.add_call(call.caller_ip, call.function_index);
        }
        
        graph
    }

    pub fn get_performance_stats(&self) -> PerformanceStats {
        let total_instructions = self.trace.instructions.len();
        let total_syscalls = self.trace.syscalls.len();
        let total_function_calls = self.trace.function_calls.len();
        
        let avg_instruction_time = if total_instructions > 0 {
            self.trace.instructions.iter()
                .map(|e| e.execution_time)
                .sum::<Duration>() / total_instructions as u32
        } else {
            Duration::ZERO
        };

        PerformanceStats {
            total_instructions: total_instructions as u64,
            total_syscalls: total_syscalls as u64,
            total_function_calls: total_function_calls as u64,
            avg_instruction_time,
            hotspots: self.trace.hotspots.clone(),
        }
    }

    pub fn export_trace(&self, format: TraceFormat) -> String {
        match format {
            TraceFormat::Json => self.export_json(),
            TraceFormat::Csv => self.export_csv(),
            TraceFormat::Chrome => self.export_chrome_trace(),
        }
    }

    fn add_instruction_trace(&mut self, entry: TraceEntry) {
        if self.trace.instructions.len() >= self.trace.max_entries {
            self.trace.instructions.pop_front();
        }
        self.trace.instructions.push_back(entry);
    }

    fn add_syscall_trace(&mut self, syscall: SyscallTrace) {
        if self.trace.syscalls.len() >= self.trace.max_entries {
            self.trace.syscalls.pop_front();
        }
        self.trace.syscalls.push_back(syscall);
    }

    fn add_function_call_trace(&mut self, call: FunctionCall) {
        if self.trace.function_calls.len() >= self.trace.max_entries {
            self.trace.function_calls.pop_front();
        }
        self.trace.function_calls.push_back(call);
    }

    fn update_hotspot(&mut self, function_index: u32, instruction_offset: u32, duration: Duration) {
        let key = (function_index << 16) | instruction_offset;
        
        let hotspot = self.hotspot_map.entry(key).or_insert(Hotspot {
            function_index,
            instruction_offset,
            hit_count: 0,
            total_time: Duration::ZERO,
            avg_time: Duration::ZERO,
        });

        hotspot.hit_count += 1;
        hotspot.total_time += duration;
        hotspot.avg_time = hotspot.total_time / hotspot.hit_count as u32;
    }

    fn update_hotspots(&mut self) {
        self.trace.hotspots = self.hotspot_map.values().cloned().collect();
        self.trace.hotspots.sort_by(|a, b| b.hit_count.cmp(&a.hit_count));
    }

    fn export_json(&self) -> String {
        // Simplified JSON export
        format!("{{\"instructions\":{},\"syscalls\":{},\"functions\":{}}}", 
                self.trace.instructions.len(),
                self.trace.syscalls.len(),
                self.trace.function_calls.len())
    }

    fn export_csv(&self) -> String {
        let mut csv = String::from("timestamp,type,details\n");
        
        for entry in &self.trace.instructions {
            csv.push_str(&format!("{:?},instruction,ip:{} opcode:{}\n", 
                entry.timestamp, entry.instruction_pointer, entry.opcode));
        }
        
        csv
    }

    fn export_chrome_trace(&self) -> String {
        // Chrome DevTools trace format
        let mut events = Vec::new();
        
        for call in &self.trace.function_calls {
            if let Some(duration) = call.duration {
                events.push(format!(
                    "{{\"name\":\"func_{}\",\"ph\":\"X\",\"ts\":{},\"dur\":{}}}",
                    call.function_index,
                    call.timestamp.elapsed().as_micros(),
                    duration.as_micros()
                ));
            }
        }
        
        format!("{{\"traceEvents\":[{}]}}", events.join(","))
    }
}

#[derive(Debug)]
pub struct CallGraph {
    nodes: std::collections::HashMap<u32, CallNode>,
}

#[derive(Debug, Clone)]
pub struct CallNode {
    pub function_index: u32,
    pub callers: Vec<u32>,
    pub callees: Vec<u32>,
    pub call_count: u64,
}

impl CallGraph {
    pub fn new() -> Self {
        Self {
            nodes: std::collections::HashMap::new(),
        }
    }

    pub fn add_call(&mut self, caller: u32, callee: u32) {
        let caller_node = self.nodes.entry(caller).or_insert(CallNode {
            function_index: caller,
            callers: Vec::new(),
            callees: Vec::new(),
            call_count: 0,
        });
        
        if !caller_node.callees.contains(&callee) {
            caller_node.callees.push(callee);
        }

        let callee_node = self.nodes.entry(callee).or_insert(CallNode {
            function_index: callee,
            callers: Vec::new(),
            callees: Vec::new(),
            call_count: 0,
        });
        
        if !callee_node.callers.contains(&caller) {
            callee_node.callers.push(caller);
        }
        
        callee_node.call_count += 1;
    }

    pub fn get_node(&self, function_index: u32) -> Option<&CallNode> {
        self.nodes.get(&function_index)
    }
}

#[derive(Debug)]
pub struct PerformanceStats {
    pub total_instructions: u64,
    pub total_syscalls: u64,
    pub total_function_calls: u64,
    pub avg_instruction_time: Duration,
    pub hotspots: Vec<Hotspot>,
}

#[derive(Debug)]
pub enum TraceFormat {
    Json,
    Csv,
    Chrome,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_execution_tracer() {
        let mut tracer = ExecutionTracer::new();
        tracer.start();
        
        tracer.trace_instruction();
        tracer.trace_syscall("wasm_get_time", &[]);
        tracer.trace_function_call(1, 0, &[42]);
        tracer.trace_function_return(Some(0));
        
        let trace = tracer.get_trace();
        assert_eq!(trace.instructions.len(), 1);
        assert_eq!(trace.syscalls.len(), 1);
        assert_eq!(trace.function_calls.len(), 1);
    }

    #[test]
    fn test_call_graph() {
        let mut graph = CallGraph::new();
        graph.add_call(0, 1);
        graph.add_call(1, 2);
        graph.add_call(0, 2);
        
        let node1 = graph.get_node(1).unwrap();
        assert_eq!(node1.callers.len(), 1);
        assert_eq!(node1.callees.len(), 1);
    }
}