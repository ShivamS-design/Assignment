use super::{SuspiciousPattern, RiskLevel};
use crate::parser::WasmModule;
use std::collections::HashMap;

pub struct PatternMatcher {
    patterns: Vec<Pattern>,
}

#[derive(Debug, Clone)]
struct Pattern {
    name: String,
    opcodes: Vec<u8>,
    risk_level: RiskLevel,
    description: String,
}

impl PatternMatcher {
    pub fn new() -> Self {
        let mut matcher = Self {
            patterns: Vec::new(),
        };
        matcher.load_patterns();
        matcher
    }

    pub fn find_patterns(&self, module: &WasmModule) -> Vec<SuspiciousPattern> {
        let mut findings = Vec::new();

        for (func_idx, code_section) in module.code.iter().enumerate() {
            let bytecode = &code_section.body;
            
            for pattern in &self.patterns {
                let matches = self.find_pattern_matches(bytecode, &pattern.opcodes);
                
                for offset in matches {
                    findings.push(SuspiciousPattern {
                        pattern_name: pattern.name.clone(),
                        function_index: func_idx as u32,
                        instruction_offset: offset,
                        description: pattern.description.clone(),
                        risk_level: pattern.risk_level.clone(),
                    });
                }
            }
        }

        findings
    }

    fn load_patterns(&mut self) {
        // Infinite loop pattern
        self.patterns.push(Pattern {
            name: "InfiniteLoop".to_string(),
            opcodes: vec![0x03, 0x40, 0x0C, 0x00, 0x0B], // loop, br 0, end
            risk_level: RiskLevel::Severe,
            description: "Potential infinite loop detected".to_string(),
        });

        // Memory bomb pattern
        self.patterns.push(Pattern {
            name: "MemoryBomb".to_string(),
            opcodes: vec![0x40, 0x00], // memory.grow
            risk_level: RiskLevel::Warning,
            description: "Memory growth operation detected".to_string(),
        });

        // Excessive recursion
        self.patterns.push(Pattern {
            name: "DeepRecursion".to_string(),
            opcodes: vec![0x10], // call
            risk_level: RiskLevel::Warning,
            description: "Function call detected - check for recursion".to_string(),
        });

        // Indirect calls (potential ROP)
        self.patterns.push(Pattern {
            name: "IndirectCall".to_string(),
            opcodes: vec![0x11], // call_indirect
            risk_level: RiskLevel::Warning,
            description: "Indirect function call detected".to_string(),
        });

        // Memory access patterns
        self.patterns.push(Pattern {
            name: "UnalignedAccess".to_string(),
            opcodes: vec![0x28, 0x00], // i32.load with alignment 0
            risk_level: RiskLevel::Warning,
            description: "Potentially unaligned memory access".to_string(),
        });

        // Stack manipulation
        self.patterns.push(Pattern {
            name: "StackManipulation".to_string(),
            opcodes: vec![0x1A, 0x1B], // drop, select
            risk_level: RiskLevel::OK,
            description: "Stack manipulation operations".to_string(),
        });

        // Crypto-like operations
        self.patterns.push(Pattern {
            name: "CryptoOperations".to_string(),
            opcodes: vec![0x73, 0x74, 0x75], // i32.xor, i32.shl, i32.shr_s
            risk_level: RiskLevel::OK,
            description: "Bitwise operations that may indicate cryptographic code".to_string(),
        });
    }

    fn find_pattern_matches(&self, bytecode: &[u8], pattern: &[u8]) -> Vec<u32> {
        let mut matches = Vec::new();
        
        if pattern.is_empty() || bytecode.len() < pattern.len() {
            return matches;
        }

        for i in 0..=bytecode.len() - pattern.len() {
            if bytecode[i..i + pattern.len()] == *pattern {
                matches.push(i as u32);
            }
        }

        matches
    }

    pub fn analyze_control_flow(&self, module: &WasmModule) -> u32 {
        let mut complexity = 0;

        for code_section in &module.code {
            complexity += self.calculate_cyclomatic_complexity(&code_section.body);
        }

        complexity
    }

    fn calculate_cyclomatic_complexity(&self, bytecode: &[u8]) -> u32 {
        let mut complexity = 1; // Base complexity
        let mut i = 0;

        while i < bytecode.len() {
            match bytecode[i] {
                0x04 => complexity += 1, // if
                0x05 => complexity += 1, // else
                0x03 => complexity += 1, // loop
                0x02 => complexity += 1, // block
                0x0C => complexity += 1, // br (conditional branch)
                0x0D => complexity += 1, // br_if
                0x0E => {                // br_table
                    complexity += 1;
                    // Skip br_table operands
                    i += 1;
                    if i < bytecode.len() {
                        let count = bytecode[i] as usize;
                        i += count + 1; // Skip all targets + default
                    }
                }
                _ => {}
            }
            i += 1;
        }

        complexity
    }

    pub fn detect_syscall_patterns(&self, module: &WasmModule) -> Vec<String> {
        let mut syscalls = Vec::new();

        // Check exports for syscall-like names
        for export in &module.exports {
            if self.is_syscall_like(&export.name) {
                syscalls.push(export.name.clone());
            }
        }

        syscalls
    }

    fn is_syscall_like(&self, name: &str) -> bool {
        let syscall_patterns = [
            "wasm_", "host_", "env_", "sys_", "os_",
            "read", "write", "open", "close", "socket",
            "connect", "bind", "listen", "accept",
            "malloc", "free", "mmap", "munmap",
            "exit", "abort", "signal", "fork", "exec"
        ];

        syscall_patterns.iter().any(|pattern| name.contains(pattern))
    }

    pub fn analyze_data_flow(&self, module: &WasmModule) -> DataFlowAnalysis {
        let mut analysis = DataFlowAnalysis {
            global_reads: 0,
            global_writes: 0,
            memory_reads: 0,
            memory_writes: 0,
            table_accesses: 0,
        };

        for code_section in &module.code {
            let bytecode = &code_section.body;
            let mut i = 0;

            while i < bytecode.len() {
                match bytecode[i] {
                    // Global operations
                    0x23 => analysis.global_reads += 1,  // global.get
                    0x24 => analysis.global_writes += 1, // global.set
                    
                    // Memory operations
                    0x28..=0x35 => analysis.memory_reads += 1,  // load operations
                    0x36..=0x3E => analysis.memory_writes += 1, // store operations
                    
                    // Table operations
                    0x25 => analysis.table_accesses += 1, // table.get
                    0x26 => analysis.table_accesses += 1, // table.set
                    
                    _ => {}
                }
                i += 1;
            }
        }

        analysis
    }
}

#[derive(Debug, Clone)]
pub struct DataFlowAnalysis {
    pub global_reads: u32,
    pub global_writes: u32,
    pub memory_reads: u32,
    pub memory_writes: u32,
    pub table_accesses: u32,
}

impl DataFlowAnalysis {
    pub fn get_risk_indicators(&self) -> Vec<String> {
        let mut indicators = Vec::new();

        if self.global_writes > 100 {
            indicators.push("Excessive global variable modifications".to_string());
        }

        if self.memory_writes > self.memory_reads * 2 {
            indicators.push("Write-heavy memory access pattern".to_string());
        }

        if self.table_accesses > 50 {
            indicators.push("Frequent table access operations".to_string());
        }

        indicators
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser::{WasmModule, CodeSection, Export, ExportKind};

    #[test]
    fn test_pattern_matching() {
        let matcher = PatternMatcher::new();
        let bytecode = vec![0x03, 0x40, 0x0C, 0x00, 0x0B]; // loop pattern
        
        let matches = matcher.find_pattern_matches(&bytecode, &[0x03, 0x40]);
        assert_eq!(matches, vec![0]);
    }

    #[test]
    fn test_control_flow_complexity() {
        let matcher = PatternMatcher::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: None,
            exports: vec![],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x04, 0x05, 0x03, 0x0B], // if, else, loop, end
            }],
        };

        let complexity = matcher.analyze_control_flow(&module);
        assert!(complexity > 1);
    }

    #[test]
    fn test_syscall_detection() {
        let matcher = PatternMatcher::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: None,
            exports: vec![
                Export {
                    name: "wasm_log".to_string(),
                    kind: ExportKind::Function,
                    index: 0,
                },
                Export {
                    name: "normal_func".to_string(),
                    kind: ExportKind::Function,
                    index: 1,
                }
            ],
            code: vec![],
        };

        let syscalls = matcher.detect_syscall_patterns(&module);
        assert_eq!(syscalls.len(), 1);
        assert_eq!(syscalls[0], "wasm_log");
    }

    #[test]
    fn test_data_flow_analysis() {
        let matcher = PatternMatcher::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: None,
            exports: vec![],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x23, 0x24, 0x28, 0x36], // global.get, global.set, i32.load, i32.store
            }],
        };

        let analysis = matcher.analyze_data_flow(&module);
        assert_eq!(analysis.global_reads, 1);
        assert_eq!(analysis.global_writes, 1);
        assert_eq!(analysis.memory_reads, 1);
        assert_eq!(analysis.memory_writes, 1);
    }
}