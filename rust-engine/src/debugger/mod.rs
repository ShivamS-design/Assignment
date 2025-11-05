pub mod core;
pub mod breakpoints;
pub mod tracer;
pub mod inspector;
pub mod session;

use crate::error::{WasmError, Result};
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct DebugInfo {
    pub instruction_pointer: u32,
    pub stack_pointer: u32,
    pub locals: Vec<i32>,
    pub memory_size: u32,
    pub call_stack: Vec<CallFrame>,
}

#[derive(Debug, Clone)]
pub struct CallFrame {
    pub function_index: u32,
    pub instruction_pointer: u32,
    pub locals_start: u32,
}

#[derive(Debug, Clone)]
pub enum StepMode {
    Into,
    Over,
    Out,
}

#[derive(Debug, Clone)]
pub struct Breakpoint {
    pub id: u32,
    pub function_index: u32,
    pub instruction_offset: u32,
    pub enabled: bool,
    pub hit_count: u32,
}

#[derive(Debug)]
pub struct WasmDebugger {
    breakpoints: breakpoints::BreakpointManager,
    tracer: tracer::ExecutionTracer,
    inspector: inspector::StateInspector,
    session: session::DebugSession,
    enabled: bool,
}

impl WasmDebugger {
    pub fn new() -> Self {
        Self {
            breakpoints: breakpoints::BreakpointManager::new(),
            tracer: tracer::ExecutionTracer::new(),
            inspector: inspector::StateInspector::new(),
            session: session::DebugSession::new(),
            enabled: false,
        }
    }

    pub fn enable(&mut self) {
        self.enabled = true;
        self.tracer.start();
    }

    pub fn disable(&mut self) {
        self.enabled = false;
        self.tracer.stop();
    }

    pub fn set_breakpoint(&mut self, function_index: u32, offset: u32) -> u32 {
        self.breakpoints.set(function_index, offset)
    }

    pub fn clear_breakpoint(&mut self, id: u32) -> bool {
        self.breakpoints.clear(id)
    }

    pub fn list_breakpoints(&self) -> Vec<&Breakpoint> {
        self.breakpoints.list()
    }

    pub fn step(&mut self, mode: StepMode) -> Result<DebugInfo> {
        if !self.enabled {
            return Err(WasmError::Runtime("Debugger not enabled".to_string()));
        }
        
        match mode {
            StepMode::Into => self.step_into(),
            StepMode::Over => self.step_over(),
            StepMode::Out => self.step_out(),
        }
    }

    pub fn continue_execution(&mut self) -> Result<DebugInfo> {
        if !self.enabled {
            return Err(WasmError::Runtime("Debugger not enabled".to_string()));
        }
        
        // Continue until breakpoint or completion
        loop {
            let info = self.step_into()?;
            
            if self.breakpoints.should_break(info.instruction_pointer) {
                return Ok(info);
            }
        }
    }

    pub fn get_debug_info(&self) -> DebugInfo {
        self.inspector.get_current_state()
    }

    pub fn inspect_memory(&self, address: u32, length: u32) -> Result<Vec<u8>> {
        self.inspector.read_memory(address, length)
    }

    pub fn get_call_stack(&self) -> Vec<CallFrame> {
        self.inspector.get_call_stack()
    }

    pub fn get_trace(&self) -> &tracer::ExecutionTrace {
        self.tracer.get_trace()
    }

    fn step_into(&mut self) -> Result<DebugInfo> {
        // Execute single instruction
        self.tracer.trace_instruction();
        Ok(self.inspector.get_current_state())
    }

    fn step_over(&mut self) -> Result<DebugInfo> {
        let current_depth = self.inspector.get_call_depth();
        
        loop {
            self.step_into()?;
            if self.inspector.get_call_depth() <= current_depth {
                break;
            }
        }
        
        Ok(self.inspector.get_current_state())
    }

    fn step_out(&mut self) -> Result<DebugInfo> {
        let current_depth = self.inspector.get_call_depth();
        
        if current_depth == 0 {
            return Err(WasmError::Runtime("Already at top level".to_string()));
        }
        
        loop {
            self.step_into()?;
            if self.inspector.get_call_depth() < current_depth {
                break;
            }
        }
        
        Ok(self.inspector.get_current_state())
    }
}