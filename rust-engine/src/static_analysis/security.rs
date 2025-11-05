use super::{SecurityAssessment, MemoryPattern, SuspiciousPattern, SyscallFunction, ResourceRequirements, RiskLevel};
use super::patterns::{PatternMatcher, DataFlowAnalysis};
use crate::parser::WasmModule;
use crate::error::{WasmError, Result};

pub struct SecurityAnalyzer {
    pattern_matcher: PatternMatcher,
}

impl SecurityAnalyzer {
    pub fn new() -> Self {
        Self {
            pattern_matcher: PatternMatcher::new(),
        }
    }

    pub fn analyze(&self, module: &WasmModule) -> Result<SecurityAssessment> {
        let memory_patterns = self.analyze_memory_patterns(module);
        let control_flow_complexity = self.pattern_matcher.analyze_control_flow(module);
        let suspicious_patterns = self.pattern_matcher.find_patterns(module);
        let syscall_functions = self.analyze_syscall_functions(module);
        let resource_requirements = self.estimate_resource_requirements(module);

        Ok(SecurityAssessment {
            memory_patterns,
            control_flow_complexity,
            suspicious_patterns,
            syscall_functions,
            resource_requirements,
        })
    }

    fn analyze_memory_patterns(&self, module: &WasmModule) -> Vec<MemoryPattern> {
        let mut patterns = Vec::new();
        let data_flow = self.pattern_matcher.analyze_data_flow(module);

        // Check for excessive memory operations
        if data_flow.memory_writes > 1000 {
            patterns.push(MemoryPattern {
                pattern_type: "ExcessiveWrites".to_string(),
                locations: vec![0], // Would track actual locations
                risk_level: RiskLevel::Warning,
                description: "Module performs excessive memory write operations".to_string(),
            });
        }

        // Check for memory growth patterns
        for (func_idx, code_section) in module.code.iter().enumerate() {
            if self.contains_memory_grow(&code_section.body) {
                patterns.push(MemoryPattern {
                    pattern_type: "MemoryGrowth".to_string(),
                    locations: vec![func_idx as u32],
                    risk_level: RiskLevel::Warning,
                    description: "Function contains memory growth operations".to_string(),
                });
            }
        }

        // Check for unaligned access patterns
        patterns.extend(self.detect_unaligned_access(module));

        // Check for buffer overflow patterns
        patterns.extend(self.detect_buffer_overflow_patterns(module));

        patterns
    }

    fn contains_memory_grow(&self, bytecode: &[u8]) -> bool {
        bytecode.contains(&0x40) // memory.grow opcode
    }

    fn detect_unaligned_access(&self, module: &WasmModule) -> Vec<MemoryPattern> {
        let mut patterns = Vec::new();

        for (func_idx, code_section) in module.code.iter().enumerate() {
            let bytecode = &code_section.body;
            let mut i = 0;

            while i < bytecode.len() {
                match bytecode[i] {
                    // Load operations with potential alignment issues
                    0x28..=0x35 => {
                        if i + 1 < bytecode.len() {
                            let alignment = bytecode[i + 1];
                            if alignment > 2 { // Suspicious alignment
                                patterns.push(MemoryPattern {
                                    pattern_type: "UnalignedAccess".to_string(),
                                    locations: vec![i as u32],
                                    risk_level: RiskLevel::Warning,
                                    description: format!("Potentially unaligned memory access in function {}", func_idx),
                                });
                            }
                        }
                    }
                    _ => {}
                }
                i += 1;
            }
        }

        patterns
    }

    fn detect_buffer_overflow_patterns(&self, module: &WasmModule) -> Vec<MemoryPattern> {
        let mut patterns = Vec::new();

        for (func_idx, code_section) in module.code.iter().enumerate() {
            let risk_score = self.analyze_bounds_checking(&code_section.body);
            
            if risk_score > 0.7 {
                patterns.push(MemoryPattern {
                    pattern_type: "BufferOverflow".to_string(),
                    locations: vec![func_idx as u32],
                    risk_level: RiskLevel::Severe,
                    description: "Function shows patterns consistent with buffer overflow vulnerabilities".to_string(),
                });
            } else if risk_score > 0.4 {
                patterns.push(MemoryPattern {
                    pattern_type: "PotentialBufferOverflow".to_string(),
                    locations: vec![func_idx as u32],
                    risk_level: RiskLevel::Warning,
                    description: "Function may have insufficient bounds checking".to_string(),
                });
            }
        }

        patterns
    }

    fn analyze_bounds_checking(&self, bytecode: &[u8]) -> f32 {
        let mut memory_accesses = 0;
        let mut bounds_checks = 0;
        let mut i = 0;

        while i < bytecode.len() {
            match bytecode[i] {
                // Memory load/store operations
                0x28..=0x3E => {
                    memory_accesses += 1;
                    
                    // Look for bounds checking patterns before memory access
                    if i >= 10 {
                        let prev_slice = &bytecode[i-10..i];
                        if self.contains_bounds_check_pattern(prev_slice) {
                            bounds_checks += 1;
                        }
                    }
                }
                _ => {}
            }
            i += 1;
        }

        if memory_accesses == 0 {
            0.0
        } else {
            1.0 - (bounds_checks as f32 / memory_accesses as f32)
        }
    }

    fn contains_bounds_check_pattern(&self, bytecode: &[u8]) -> bool {
        // Look for comparison operations that might indicate bounds checking
        bytecode.iter().any(|&opcode| matches!(opcode, 0x46..=0x51)) // i32 comparison ops
    }

    fn analyze_syscall_functions(&self, module: &WasmModule) -> Vec<SyscallFunction> {
        let mut syscalls = Vec::new();
        let syscall_names = self.pattern_matcher.detect_syscall_patterns(module);

        for (idx, name) in syscall_names.iter().enumerate() {
            let risk_level = self.assess_syscall_risk(name);
            let usage_count = self.count_syscall_usage(module, name);

            syscalls.push(SyscallFunction {
                name: name.clone(),
                import_index: idx as u32,
                usage_count,
                risk_level,
            });
        }

        syscalls
    }

    fn assess_syscall_risk(&self, name: &str) -> RiskLevel {
        let high_risk = ["exec", "fork", "system", "open", "write", "socket", "connect"];
        let medium_risk = ["read", "malloc", "free", "mmap", "signal"];

        if high_risk.iter().any(|&risk| name.contains(risk)) {
            RiskLevel::Severe
        } else if medium_risk.iter().any(|&risk| name.contains(risk)) {
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        }
    }

    fn count_syscall_usage(&self, module: &WasmModule, syscall_name: &str) -> u32 {
        let mut count = 0;

        // Find the export index for this syscall
        let export_index = module.exports.iter()
            .position(|export| export.name == *syscall_name);

        if let Some(idx) = export_index {
            // Count call instructions to this function
            for code_section in &module.code {
                count += self.count_function_calls(&code_section.body, idx as u32);
            }
        }

        count
    }

    fn count_function_calls(&self, bytecode: &[u8], target_function: u32) -> u32 {
        let mut count = 0;
        let mut i = 0;

        while i < bytecode.len() {
            if bytecode[i] == 0x10 { // call instruction
                if i + 1 < bytecode.len() {
                    // In real implementation, would decode LEB128 function index
                    if bytecode[i + 1] == target_function as u8 {
                        count += 1;
                    }
                }
            }
            i += 1;
        }

        count
    }

    fn estimate_resource_requirements(&self, module: &WasmModule) -> ResourceRequirements {
        let mut estimated_memory = 0u64;
        let mut estimated_cpu_cycles = 0u64;
        let mut max_stack_depth = 0u32;
        let mut max_call_depth = 0u32;

        // Estimate memory requirements
        if let Some(memory) = &module.memory {
            estimated_memory = (memory.min as u64) * 65536; // Pages to bytes
            if let Some(max) = memory.max {
                estimated_memory = estimated_memory.max((max as u64) * 65536);
            }
        }

        // Estimate CPU cycles and stack usage
        for code_section in &module.code {
            let (cycles, stack_depth, call_depth) = self.analyze_function_complexity(&code_section.body);
            estimated_cpu_cycles += cycles;
            max_stack_depth = max_stack_depth.max(stack_depth);
            max_call_depth = max_call_depth.max(call_depth);
        }

        ResourceRequirements {
            estimated_memory,
            estimated_cpu_cycles,
            max_stack_depth,
            max_call_depth,
        }
    }

    fn analyze_function_complexity(&self, bytecode: &[u8]) -> (u64, u32, u32) {
        let mut cycles = 0u64;
        let mut stack_depth = 0u32;
        let mut max_stack_depth = 0u32;
        let mut call_depth = 0u32;
        let mut max_call_depth = 0u32;

        for &opcode in bytecode {
            // Estimate cycles per instruction
            cycles += match opcode {
                // Simple operations
                0x01..=0x11 => 1,
                // Memory operations
                0x28..=0x3E => 3,
                // Control flow
                0x02..=0x05 => 2,
                // Arithmetic
                0x6A..=0xC4 => 1,
                _ => 1,
            };

            // Track stack depth
            match opcode {
                // Instructions that push to stack
                0x41..=0x44 => stack_depth += 1, // const operations
                0x20..=0x22 => stack_depth += 1, // local.get, etc.
                0x23 => stack_depth += 1,        // global.get
                0x28..=0x35 => stack_depth += 1, // load operations
                
                // Instructions that pop from stack
                0x1A => stack_depth = stack_depth.saturating_sub(1), // drop
                0x21..=0x22 => stack_depth = stack_depth.saturating_sub(1), // local.set, local.tee
                0x24 => stack_depth = stack_depth.saturating_sub(1), // global.set
                0x36..=0x3E => stack_depth = stack_depth.saturating_sub(2), // store operations
                
                // Function calls
                0x10 | 0x11 => {
                    call_depth += 1;
                    max_call_depth = max_call_depth.max(call_depth);
                }
                
                // Return
                0x0F => call_depth = call_depth.saturating_sub(1),
                
                _ => {}
            }

            max_stack_depth = max_stack_depth.max(stack_depth);
        }

        (cycles, max_stack_depth, max_call_depth)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser::{WasmModule, CodeSection, MemoryType};

    #[test]
    fn test_security_analyzer() {
        let analyzer = SecurityAnalyzer::new();
        let module = create_test_module();
        
        let result = analyzer.analyze(&module);
        assert!(result.is_ok());
        
        let assessment = result.unwrap();
        assert!(assessment.control_flow_complexity >= 1);
    }

    #[test]
    fn test_memory_pattern_detection() {
        let analyzer = SecurityAnalyzer::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: Some(MemoryType { min: 1, max: Some(10) }),
            exports: vec![],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x40, 0x00], // memory.grow
            }],
        };

        let patterns = analyzer.analyze_memory_patterns(&module);
        assert!(!patterns.is_empty());
        assert_eq!(patterns[0].pattern_type, "MemoryGrowth");
    }

    #[test]
    fn test_resource_estimation() {
        let analyzer = SecurityAnalyzer::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: Some(MemoryType { min: 2, max: Some(4) }),
            exports: vec![],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x41, 0x01, 0x10, 0x00], // i32.const 1, call 0
            }],
        };

        let requirements = analyzer.estimate_resource_requirements(&module);
        assert_eq!(requirements.estimated_memory, 4 * 65536); // 4 pages max
        assert!(requirements.estimated_cpu_cycles > 0);
        assert!(requirements.max_call_depth > 0);
    }

    fn create_test_module() -> WasmModule {
        WasmModule {
            types: vec![],
            functions: vec![],
            memory: None,
            exports: vec![],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x41, 0x01], // i32.const 1
            }],
        }
    }
}