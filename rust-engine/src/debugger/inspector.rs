use super::{DebugInfo, CallFrame};
use crate::error::{WasmError, Result};
use crate::memory::LinearMemory;
use std::collections::HashMap;

#[derive(Debug)]
pub struct StateInspector {
    current_ip: u32,
    stack_pointer: u32,
    locals: Vec<i32>,
    call_stack: Vec<CallFrame>,
    memory: Option<LinearMemory>,
    registers: HashMap<String, i32>,
}

impl StateInspector {
    pub fn new() -> Self {
        Self {
            current_ip: 0,
            stack_pointer: 0,
            locals: Vec::new(),
            call_stack: Vec::new(),
            memory: None,
            registers: HashMap::new(),
        }
    }

    pub fn set_memory(&mut self, memory: LinearMemory) {
        self.memory = Some(memory);
    }

    pub fn update_state(&mut self, ip: u32, sp: u32, locals: Vec<i32>) {
        self.current_ip = ip;
        self.stack_pointer = sp;
        self.locals = locals;
    }

    pub fn push_call_frame(&mut self, function_index: u32, ip: u32, locals_start: u32) {
        let frame = CallFrame {
            function_index,
            instruction_pointer: ip,
            locals_start,
        };
        self.call_stack.push(frame);
    }

    pub fn pop_call_frame(&mut self) -> Option<CallFrame> {
        self.call_stack.pop()
    }

    pub fn get_current_state(&self) -> DebugInfo {
        DebugInfo {
            instruction_pointer: self.current_ip,
            stack_pointer: self.stack_pointer,
            locals: self.locals.clone(),
            memory_size: self.memory.as_ref().map(|m| m.size()).unwrap_or(0),
            call_stack: self.call_stack.clone(),
        }
    }

    pub fn get_call_stack(&self) -> Vec<CallFrame> {
        self.call_stack.clone()
    }

    pub fn get_call_depth(&self) -> usize {
        self.call_stack.len()
    }

    pub fn read_memory(&self, address: u32, length: u32) -> Result<Vec<u8>> {
        if let Some(memory) = &self.memory {
            let bytes = memory.read_bytes(address, length)?;
            Ok(bytes.to_vec())
        } else {
            Err(WasmError::Runtime("No memory attached".to_string()))
        }
    }

    pub fn read_memory_u32(&self, address: u32) -> Result<u32> {
        if let Some(memory) = &self.memory {
            memory.read_u32(address)
        } else {
            Err(WasmError::Runtime("No memory attached".to_string()))
        }
    }

    pub fn get_memory_view(&self, start: u32, end: u32) -> Result<MemoryView> {
        if let Some(memory) = &self.memory {
            let length = end - start;
            let data = memory.read_bytes(start, length)?;
            
            Ok(MemoryView {
                start_address: start,
                data: data.to_vec(),
                annotations: self.get_memory_annotations(start, end),
            })
        } else {
            Err(WasmError::Runtime("No memory attached".to_string()))
        }
    }

    pub fn get_stack_trace(&self) -> StackTrace {
        let mut trace = StackTrace {
            frames: Vec::new(),
            total_depth: self.call_stack.len(),
        };

        for (i, frame) in self.call_stack.iter().enumerate() {
            trace.frames.push(StackFrame {
                index: i,
                function_index: frame.function_index,
                instruction_pointer: frame.instruction_pointer,
                locals_start: frame.locals_start,
                function_name: format!("func_{}", frame.function_index),
            });
        }

        trace
    }

    pub fn inspect_locals(&self, frame_index: Option<usize>) -> Result<Vec<LocalVariable>> {
        let frame = if let Some(index) = frame_index {
            self.call_stack.get(index)
        } else {
            self.call_stack.last()
        };

        if let Some(frame) = frame {
            let mut variables = Vec::new();
            
            // In a real implementation, we'd use debug info to get variable names
            for (i, &value) in self.locals.iter().enumerate() {
                variables.push(LocalVariable {
                    index: i,
                    name: format!("local_{}", i),
                    value: VariableValue::I32(value),
                    type_name: "i32".to_string(),
                });
            }
            
            Ok(variables)
        } else {
            Err(WasmError::Runtime("Invalid frame index".to_string()))
        }
    }

    pub fn set_register(&mut self, name: &str, value: i32) {
        self.registers.insert(name.to_string(), value);
    }

    pub fn get_register(&self, name: &str) -> Option<i32> {
        self.registers.get(name).copied()
    }

    pub fn list_registers(&self) -> Vec<Register> {
        self.registers.iter().map(|(name, &value)| Register {
            name: name.clone(),
            value,
        }).collect()
    }

    fn get_memory_annotations(&self, start: u32, end: u32) -> Vec<MemoryAnnotation> {
        let mut annotations = Vec::new();
        
        // Add stack pointer annotation
        if self.stack_pointer >= start && self.stack_pointer < end {
            annotations.push(MemoryAnnotation {
                address: self.stack_pointer,
                label: "SP".to_string(),
                description: "Stack Pointer".to_string(),
            });
        }

        // Add local variable annotations
        for (i, frame) in self.call_stack.iter().enumerate() {
            let locals_addr = frame.locals_start;
            if locals_addr >= start && locals_addr < end {
                annotations.push(MemoryAnnotation {
                    address: locals_addr,
                    label: format!("F{}", i),
                    description: format!("Frame {} locals", i),
                });
            }
        }

        annotations
    }
}

#[derive(Debug, Clone)]
pub struct MemoryView {
    pub start_address: u32,
    pub data: Vec<u8>,
    pub annotations: Vec<MemoryAnnotation>,
}

#[derive(Debug, Clone)]
pub struct MemoryAnnotation {
    pub address: u32,
    pub label: String,
    pub description: String,
}

#[derive(Debug, Clone)]
pub struct StackTrace {
    pub frames: Vec<StackFrame>,
    pub total_depth: usize,
}

#[derive(Debug, Clone)]
pub struct StackFrame {
    pub index: usize,
    pub function_index: u32,
    pub instruction_pointer: u32,
    pub locals_start: u32,
    pub function_name: String,
}

#[derive(Debug, Clone)]
pub struct LocalVariable {
    pub index: usize,
    pub name: String,
    pub value: VariableValue,
    pub type_name: String,
}

#[derive(Debug, Clone)]
pub enum VariableValue {
    I32(i32),
    I64(i64),
    F32(f32),
    F64(f64),
}

#[derive(Debug, Clone)]
pub struct Register {
    pub name: String,
    pub value: i32,
}

impl MemoryView {
    pub fn format_hex(&self, bytes_per_line: usize) -> String {
        let mut output = String::new();
        
        for (i, chunk) in self.data.chunks(bytes_per_line).enumerate() {
            let address = self.start_address + (i * bytes_per_line) as u32;
            output.push_str(&format!("{:08x}: ", address));
            
            // Hex bytes
            for (j, &byte) in chunk.iter().enumerate() {
                output.push_str(&format!("{:02x} ", byte));
                if j == bytes_per_line / 2 - 1 {
                    output.push(' ');
                }
            }
            
            // Pad if incomplete line
            for _ in chunk.len()..bytes_per_line {
                output.push_str("   ");
            }
            
            output.push_str(" |");
            
            // ASCII representation
            for &byte in chunk {
                if byte >= 32 && byte <= 126 {
                    output.push(byte as char);
                } else {
                    output.push('.');
                }
            }
            
            output.push_str("|\n");
            
            // Add annotations for this line
            for annotation in &self.annotations {
                let line_start = self.start_address + (i * bytes_per_line) as u32;
                let line_end = line_start + bytes_per_line as u32;
                
                if annotation.address >= line_start && annotation.address < line_end {
                    let offset = annotation.address - line_start;
                    output.push_str(&format!("         {}{} {}\n", 
                        " ".repeat(offset as usize * 3),
                        "^".repeat(annotation.label.len()),
                        annotation.description));
                }
            }
        }
        
        output
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_state_inspector() {
        let mut inspector = StateInspector::new();
        
        inspector.update_state(100, 200, vec![1, 2, 3]);
        inspector.push_call_frame(0, 100, 0);
        
        let state = inspector.get_current_state();
        assert_eq!(state.instruction_pointer, 100);
        assert_eq!(state.stack_pointer, 200);
        assert_eq!(state.locals, vec![1, 2, 3]);
        assert_eq!(state.call_stack.len(), 1);
    }

    #[test]
    fn test_memory_view_formatting() {
        let view = MemoryView {
            start_address: 0x1000,
            data: vec![0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64, 0x21],
            annotations: vec![
                MemoryAnnotation {
                    address: 0x1000,
                    label: "STR".to_string(),
                    description: "Hello World string".to_string(),
                }
            ],
        };
        
        let formatted = view.format_hex(16);
        assert!(formatted.contains("48 65 6c 6c"));
        assert!(formatted.contains("Hello"));
    }

    #[test]
    fn test_call_stack() {
        let mut inspector = StateInspector::new();
        
        inspector.push_call_frame(0, 100, 0);
        inspector.push_call_frame(1, 200, 16);
        
        let trace = inspector.get_stack_trace();
        assert_eq!(trace.frames.len(), 2);
        assert_eq!(trace.total_depth, 2);
        
        let frame = inspector.pop_call_frame().unwrap();
        assert_eq!(frame.function_index, 1);
        assert_eq!(inspector.get_call_depth(), 1);
    }
}