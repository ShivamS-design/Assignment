use wasm_engine::debugger::*;
use wasm_engine::parser::{WasmModule, FunctionType};

#[test]
fn test_debugger_creation() {
    let mut debugger = WasmDebugger::new();
    debugger.enable();
    
    assert!(debugger.enabled);
}

#[test]
fn test_breakpoint_management() {
    let mut debugger = WasmDebugger::new();
    
    let bp_id = debugger.set_breakpoint(0, 10);
    assert_eq!(bp_id, 1);
    
    let breakpoints = debugger.list_breakpoints();
    assert_eq!(breakpoints.len(), 1);
    assert_eq!(breakpoints[0].function_index, 0);
    assert_eq!(breakpoints[0].instruction_offset, 10);
    
    assert!(debugger.clear_breakpoint(bp_id));
    assert_eq!(debugger.list_breakpoints().len(), 0);
}

#[test]
fn test_stepping() {
    let mut debugger = WasmDebugger::new();
    debugger.enable();
    
    let result = debugger.step(StepMode::Into);
    assert!(result.is_ok());
    
    let result = debugger.step(StepMode::Over);
    assert!(result.is_ok());
}

#[test]
fn test_execution_tracer() {
    let mut tracer = tracer::ExecutionTracer::new();
    tracer.start();
    
    tracer.trace_instruction();
    tracer.trace_syscall("wasm_get_time", &[]);
    tracer.trace_function_call(1, 0, &[42]);
    tracer.trace_function_return(Some(0));
    
    let trace = tracer.get_trace();
    assert_eq!(trace.instructions.len(), 1);
    assert_eq!(trace.syscalls.len(), 1);
    assert_eq!(trace.function_calls.len(), 1);
    
    let stats = tracer.get_performance_stats();
    assert_eq!(stats.total_instructions, 1);
    assert_eq!(stats.total_syscalls, 1);
    assert_eq!(stats.total_function_calls, 1);
}

#[test]
fn test_call_graph() {
    let mut graph = tracer::CallGraph::new();
    
    graph.add_call(0, 1);
    graph.add_call(1, 2);
    graph.add_call(0, 2);
    
    let node1 = graph.get_node(1).unwrap();
    assert_eq!(node1.callers.len(), 1);
    assert_eq!(node1.callees.len(), 1);
    assert!(node1.callers.contains(&0));
    assert!(node1.callees.contains(&2));
    
    let node2 = graph.get_node(2).unwrap();
    assert_eq!(node2.callers.len(), 2);
    assert!(node2.callers.contains(&0));
    assert!(node2.callers.contains(&1));
}

#[test]
fn test_state_inspector() {
    let mut inspector = inspector::StateInspector::new();
    
    inspector.update_state(100, 200, vec![1, 2, 3]);
    inspector.push_call_frame(0, 100, 0);
    inspector.push_call_frame(1, 150, 16);
    
    let state = inspector.get_current_state();
    assert_eq!(state.instruction_pointer, 100);
    assert_eq!(state.stack_pointer, 200);
    assert_eq!(state.locals, vec![1, 2, 3]);
    assert_eq!(state.call_stack.len(), 2);
    
    let trace = inspector.get_stack_trace();
    assert_eq!(trace.frames.len(), 2);
    assert_eq!(trace.total_depth, 2);
    
    assert_eq!(inspector.get_call_depth(), 2);
    
    let frame = inspector.pop_call_frame().unwrap();
    assert_eq!(frame.function_index, 1);
    assert_eq!(inspector.get_call_depth(), 1);
}

#[test]
fn test_memory_inspection() {
    use wasm_engine::memory::LinearMemory;
    
    let mut inspector = inspector::StateInspector::new();
    let mut memory = LinearMemory::new(1, None).unwrap();
    
    // Write test data
    memory.write_u32(0, 0x12345678).unwrap();
    memory.write_u32(4, 0xABCDEF00).unwrap();
    
    inspector.set_memory(memory);
    
    let result = inspector.read_memory(0, 8);
    assert!(result.is_ok());
    
    let data = result.unwrap();
    assert_eq!(data.len(), 8);
    
    let view = inspector.get_memory_view(0, 16).unwrap();
    assert_eq!(view.start_address, 0);
    assert_eq!(view.data.len(), 16);
    
    let formatted = view.format_hex(16);
    assert!(formatted.contains("78 56 34 12")); // Little endian
}

#[test]
fn test_debug_session() {
    let mut session = session::DebugSession::with_module("test_module");
    
    let bookmark_id = session.add_bookmark("main", 0, 10, "Entry point");
    assert_eq!(session.bookmarks.len(), 1);
    
    let note_id = session.add_note("This is a test note", None);
    assert_eq!(session.notes.len(), 1);
    
    session.set_variable("test_var", "42");
    assert_eq!(session.get_variable("test_var"), Some("42"));
    
    assert!(session.remove_bookmark(bookmark_id));
    assert_eq!(session.bookmarks.len(), 0);
}

#[test]
fn test_session_manager() {
    let mut manager = session::SessionManager::new();
    
    let id1 = manager.create_session("module1");
    let id2 = manager.create_session("module2");
    
    assert_eq!(manager.list_sessions().len(), 2);
    assert_eq!(manager.get_current_session().unwrap().module_name, "module2");
    
    assert!(manager.switch_session(&id1));
    assert_eq!(manager.get_current_session().unwrap().module_name, "module1");
    
    assert!(manager.remove_session(&id1));
    assert_eq!(manager.list_sessions().len(), 1);
}

#[test]
fn test_debug_context() {
    let module = WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![],
        memory: None,
        exports: vec![],
        code: vec![],
    };
    
    let mut context = core::DebugContext::new(module);
    context.set_custom_section("name".to_string(), vec![1, 2, 3, 4]);
    
    assert!(context.custom_sections.contains_key("name"));
    
    let info = context.resolve_address(0x00010020);
    assert_eq!(info.function_index, 1);
    assert_eq!(info.instruction_offset, 32);
}

#[test]
fn test_variable_inspector() {
    let module = WasmModule {
        types: vec![],
        functions: vec![],
        memory: None,
        exports: vec![],
        code: vec![],
    };
    
    let context = core::DebugContext::new(module);
    let mut inspector = core::VariableInspector::new(context);
    
    inspector.set_current_frame(0);
    let variables = inspector.list_variables();
    
    // Should be empty for test module
    assert_eq!(variables.len(), 0);
}

#[test]
fn test_trace_export() {
    let mut tracer = tracer::ExecutionTracer::new();
    tracer.start();
    
    tracer.trace_instruction();
    tracer.trace_syscall("wasm_get_time", &[]);
    tracer.trace_function_call(1, 0, &[42]);
    
    let json_trace = tracer.export_trace(tracer::TraceFormat::Json);
    assert!(json_trace.contains("instructions"));
    
    let csv_trace = tracer.export_trace(tracer::TraceFormat::Csv);
    assert!(csv_trace.contains("timestamp,type,details"));
    
    let chrome_trace = tracer.export_trace(tracer::TraceFormat::Chrome);
    assert!(chrome_trace.contains("traceEvents"));
}

#[test]
fn test_hotspot_detection() {
    let mut tracer = tracer::ExecutionTracer::new();
    tracer.start();
    
    // Simulate multiple executions of the same instruction
    for _ in 0..10 {
        tracer.trace_instruction();
    }
    
    tracer.stop();
    
    let stats = tracer.get_performance_stats();
    assert!(!stats.hotspots.is_empty());
    
    let hotspot = &stats.hotspots[0];
    assert_eq!(hotspot.hit_count, 10);
}

#[test]
fn test_performance_overhead() {
    use std::time::Instant;
    
    let mut tracer = tracer::ExecutionTracer::new();
    
    // Test without tracing
    let start = Instant::now();
    for _ in 0..1000 {
        // Simulate instruction execution
    }
    let without_tracing = start.elapsed();
    
    // Test with tracing
    tracer.start();
    let start = Instant::now();
    for _ in 0..1000 {
        tracer.trace_instruction();
    }
    let with_tracing = start.elapsed();
    
    // Tracing overhead should be reasonable (less than 10x)
    assert!(with_tracing < without_tracing * 10);
}