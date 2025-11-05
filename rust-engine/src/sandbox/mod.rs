pub mod limits;
pub mod capabilities;
pub mod syscall_trap;
pub mod policy;

use crate::error::{WasmError, Result};
use std::time::{Duration, Instant};
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct ResourceLimits {
    pub max_memory_pages: u32,
    pub max_cpu_time: Duration,
    pub max_syscalls: u32,
    pub max_instructions: u64,
}

impl Default for ResourceLimits {
    fn default() -> Self {
        Self {
            max_memory_pages: 256, // 16MB
            max_cpu_time: Duration::from_secs(30),
            max_syscalls: 1000,
            max_instructions: 1_000_000,
        }
    }
}

#[derive(Debug)]
pub struct ResourceUsage {
    pub memory_pages: u32,
    pub cpu_time: Duration,
    pub syscall_count: u32,
    pub instruction_count: u64,
    pub start_time: Instant,
}

impl ResourceUsage {
    pub fn new() -> Self {
        Self {
            memory_pages: 0,
            cpu_time: Duration::ZERO,
            syscall_count: 0,
            instruction_count: 0,
            start_time: Instant::now(),
        }
    }
    
    pub fn update_cpu_time(&mut self) {
        self.cpu_time = self.start_time.elapsed();
    }
    
    pub fn increment_syscall(&mut self) {
        self.syscall_count += 1;
    }
    
    pub fn increment_instructions(&mut self, count: u64) {
        self.instruction_count += count;
    }
}

#[derive(Debug)]
pub struct Sandbox {
    limits: ResourceLimits,
    usage: ResourceUsage,
    capabilities: capabilities::CapabilitySet,
    syscall_trap: syscall_trap::SyscallTrap,
    violations: Vec<SecurityViolation>,
}

#[derive(Debug, Clone)]
pub struct SecurityViolation {
    pub violation_type: ViolationType,
    pub message: String,
    pub timestamp: Instant,
}

#[derive(Debug, Clone)]
pub enum ViolationType {
    MemoryLimit,
    CpuTimeLimit,
    SyscallQuota,
    InstructionLimit,
    UnauthorizedSyscall,
    CapabilityViolation,
}

impl Sandbox {
    pub fn new(limits: ResourceLimits) -> Self {
        Self {
            limits,
            usage: ResourceUsage::new(),
            capabilities: capabilities::CapabilitySet::new(),
            syscall_trap: syscall_trap::SyscallTrap::new(),
            violations: Vec::new(),
        }
    }
    
    pub fn check_limits(&mut self) -> Result<()> {
        self.usage.update_cpu_time();
        
        if self.usage.memory_pages > self.limits.max_memory_pages {
            self.log_violation(ViolationType::MemoryLimit, 
                format!("Memory limit exceeded: {} > {}", 
                    self.usage.memory_pages, self.limits.max_memory_pages));
            return Err(WasmError::Runtime("Memory limit exceeded".to_string()));
        }
        
        if self.usage.cpu_time > self.limits.max_cpu_time {
            self.log_violation(ViolationType::CpuTimeLimit,
                format!("CPU time limit exceeded: {:?} > {:?}",
                    self.usage.cpu_time, self.limits.max_cpu_time));
            return Err(WasmError::Runtime("CPU time limit exceeded".to_string()));
        }
        
        if self.usage.syscall_count > self.limits.max_syscalls {
            self.log_violation(ViolationType::SyscallQuota,
                format!("Syscall quota exceeded: {} > {}",
                    self.usage.syscall_count, self.limits.max_syscalls));
            return Err(WasmError::Runtime("Syscall quota exceeded".to_string()));
        }
        
        if self.usage.instruction_count > self.limits.max_instructions {
            self.log_violation(ViolationType::InstructionLimit,
                format!("Instruction limit exceeded: {} > {}",
                    self.usage.instruction_count, self.limits.max_instructions));
            return Err(WasmError::Runtime("Instruction limit exceeded".to_string()));
        }
        
        Ok(())
    }
    
    pub fn grant_capability(&mut self, capability: capabilities::Capability) {
        self.capabilities.grant(capability);
    }
    
    pub fn check_capability(&self, capability: &capabilities::Capability) -> Result<()> {
        if !self.capabilities.has(capability) {
            return Err(WasmError::Runtime("Capability violation".to_string()));
        }
        Ok(())
    }
    
    pub fn intercept_syscall(&mut self, syscall: &str, args: &[u32]) -> Result<u32> {
        self.usage.increment_syscall();
        self.check_limits()?;
        
        if !self.syscall_trap.is_allowed(syscall) {
            self.log_violation(ViolationType::UnauthorizedSyscall,
                format!("Unauthorized syscall: {}", syscall));
            return Err(WasmError::Runtime("Unauthorized syscall".to_string()));
        }
        
        self.syscall_trap.handle(syscall, args)
    }
    
    pub fn update_memory_usage(&mut self, pages: u32) {
        self.usage.memory_pages = pages;
    }
    
    pub fn increment_instructions(&mut self, count: u64) {
        self.usage.increment_instructions(count);
    }
    
    pub fn get_violations(&self) -> &[SecurityViolation] {
        &self.violations
    }
    
    fn log_violation(&mut self, violation_type: ViolationType, message: String) {
        log::warn!("Security violation: {:?} - {}", violation_type, message);
        self.violations.push(SecurityViolation {
            violation_type,
            message,
            timestamp: Instant::now(),
        });
    }
}