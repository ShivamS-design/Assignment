use super::{CapabilityRequirements, Permission, SecurityAssessment, RiskLevel};
use crate::parser::WasmModule;
use crate::error::{WasmError, Result};
use std::collections::HashSet;

pub struct CapabilityInferrer {
    capability_rules: Vec<CapabilityRule>,
}

#[derive(Debug, Clone)]
struct CapabilityRule {
    name: String,
    triggers: Vec<Trigger>,
    required: bool,
    risk_level: RiskLevel,
    description: String,
}

#[derive(Debug, Clone)]
enum Trigger {
    ExportName(String),
    ImportName(String),
    MemoryUsage(u32),
    SyscallPattern(String),
    InstructionPattern(Vec<u8>),
}

impl CapabilityInferrer {
    pub fn new() -> Self {
        let mut inferrer = Self {
            capability_rules: Vec::new(),
        };
        inferrer.load_capability_rules();
        inferrer
    }

    pub fn infer(&self, module: &WasmModule, security: &SecurityAssessment) -> Result<CapabilityRequirements> {
        let mut required_capabilities = HashSet::new();
        let mut optional_capabilities = HashSet::new();
        let mut inferred_permissions = Vec::new();

        // Analyze based on exports
        for export in &module.exports {
            self.analyze_export(&export.name, &mut required_capabilities, &mut optional_capabilities, &mut inferred_permissions);
        }

        // Analyze based on memory usage
        if let Some(memory) = &module.memory {
            self.analyze_memory_requirements(memory, &mut required_capabilities, &mut inferred_permissions);
        }

        // Analyze based on security assessment
        self.analyze_security_patterns(security, &mut required_capabilities, &mut inferred_permissions);

        // Analyze syscall functions
        for syscall in &security.syscall_functions {
            self.analyze_syscall_capability(&syscall.name, syscall.risk_level.clone(), 
                                          &mut required_capabilities, &mut inferred_permissions);
        }

        Ok(CapabilityRequirements {
            required_capabilities: required_capabilities.into_iter().collect(),
            optional_capabilities: optional_capabilities.into_iter().collect(),
            inferred_permissions,
        })
    }

    fn load_capability_rules(&mut self) {
        // Logging capability
        self.capability_rules.push(CapabilityRule {
            name: "Log".to_string(),
            triggers: vec![
                Trigger::ExportName("wasm_log".to_string()),
                Trigger::ImportName("log".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::OK,
            description: "Module requires logging capability".to_string(),
        });

        // Time access capability
        self.capability_rules.push(CapabilityRule {
            name: "GetTime".to_string(),
            triggers: vec![
                Trigger::ExportName("wasm_get_time".to_string()),
                Trigger::ImportName("time".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::OK,
            description: "Module requires time access".to_string(),
        });

        // Random number generation
        self.capability_rules.push(CapabilityRule {
            name: "Random".to_string(),
            triggers: vec![
                Trigger::ExportName("wasm_random".to_string()),
                Trigger::ImportName("random".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::OK,
            description: "Module requires random number generation".to_string(),
        });

        // Sensor access
        self.capability_rules.push(CapabilityRule {
            name: "ReadSensor".to_string(),
            triggers: vec![
                Trigger::ExportName("wasm_read_sensor".to_string()),
                Trigger::SyscallPattern("sensor".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::Warning,
            description: "Module requires sensor access".to_string(),
        });

        // Alert system
        self.capability_rules.push(CapabilityRule {
            name: "SendAlert".to_string(),
            triggers: vec![
                Trigger::ExportName("wasm_send_alert".to_string()),
                Trigger::SyscallPattern("alert".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::Warning,
            description: "Module requires alert sending capability".to_string(),
        });

        // Memory growth
        self.capability_rules.push(CapabilityRule {
            name: "MemoryGrow".to_string(),
            triggers: vec![
                Trigger::InstructionPattern(vec![0x40]), // memory.grow
                Trigger::MemoryUsage(100), // > 100 pages
            ],
            required: true,
            risk_level: RiskLevel::Warning,
            description: "Module requires memory growth capability".to_string(),
        });

        // Network access (high risk)
        self.capability_rules.push(CapabilityRule {
            name: "NetworkAccess".to_string(),
            triggers: vec![
                Trigger::SyscallPattern("socket".to_string()),
                Trigger::SyscallPattern("connect".to_string()),
                Trigger::ImportName("network".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::Severe,
            description: "Module requires network access - HIGH RISK".to_string(),
        });

        // File system access (high risk)
        self.capability_rules.push(CapabilityRule {
            name: "FileSystemAccess".to_string(),
            triggers: vec![
                Trigger::SyscallPattern("open".to_string()),
                Trigger::SyscallPattern("read".to_string()),
                Trigger::SyscallPattern("write".to_string()),
                Trigger::ImportName("fs".to_string()),
            ],
            required: true,
            risk_level: RiskLevel::Severe,
            description: "Module requires file system access - HIGH RISK".to_string(),
        });
    }

    fn analyze_export(&self, export_name: &str, required: &mut HashSet<String>, 
                     optional: &mut HashSet<String>, permissions: &mut Vec<Permission>) {
        for rule in &self.capability_rules {
            for trigger in &rule.triggers {
                if let Trigger::ExportName(name) = trigger {
                    if export_name.contains(name) {
                        if rule.required {
                            required.insert(rule.name.clone());
                        } else {
                            optional.insert(rule.name.clone());
                        }
                        
                        permissions.push(Permission {
                            name: rule.name.clone(),
                            required: rule.required,
                            reason: format!("Export '{}' detected", export_name),
                        });
                    }
                }
            }
        }
    }

    fn analyze_memory_requirements(&self, memory: &crate::parser::MemoryType, 
                                 required: &mut HashSet<String>, permissions: &mut Vec<Permission>) {
        if memory.min > 10 || memory.max.unwrap_or(0) > 100 {
            required.insert("MemoryGrow".to_string());
            permissions.push(Permission {
                name: "MemoryGrow".to_string(),
                required: true,
                reason: format!("Large memory requirement: {} pages", memory.min),
            });
        }

        if memory.max.is_none() {
            required.insert("UnlimitedMemory".to_string());
            permissions.push(Permission {
                name: "UnlimitedMemory".to_string(),
                required: true,
                reason: "Module requests unlimited memory growth".to_string(),
            });
        }
    }

    fn analyze_security_patterns(&self, security: &SecurityAssessment, 
                               required: &mut HashSet<String>, permissions: &mut Vec<Permission>) {
        // Check for memory growth patterns
        for pattern in &security.memory_patterns {
            if pattern.pattern_type == "MemoryGrowth" {
                required.insert("MemoryGrow".to_string());
                permissions.push(Permission {
                    name: "MemoryGrow".to_string(),
                    required: true,
                    reason: "Memory growth operations detected".to_string(),
                });
            }
        }

        // Check for high complexity
        if security.control_flow_complexity > 100 {
            required.insert("HighComplexity".to_string());
            permissions.push(Permission {
                name: "HighComplexity".to_string(),
                required: true,
                reason: format!("High control flow complexity: {}", security.control_flow_complexity),
            });
        }

        // Check for suspicious patterns
        for pattern in &security.suspicious_patterns {
            match pattern.pattern_name.as_str() {
                "InfiniteLoop" => {
                    required.insert("LongRunning".to_string());
                    permissions.push(Permission {
                        name: "LongRunning".to_string(),
                        required: true,
                        reason: "Potential infinite loop detected".to_string(),
                    });
                }
                "IndirectCall" => {
                    required.insert("DynamicExecution".to_string());
                    permissions.push(Permission {
                        name: "DynamicExecution".to_string(),
                        required: true,
                        reason: "Indirect function calls detected".to_string(),
                    });
                }
                _ => {}
            }
        }
    }

    fn analyze_syscall_capability(&self, syscall_name: &str, risk_level: RiskLevel,
                                required: &mut HashSet<String>, permissions: &mut Vec<Permission>) {
        for rule in &self.capability_rules {
            for trigger in &rule.triggers {
                if let Trigger::SyscallPattern(pattern) = trigger {
                    if syscall_name.contains(pattern) {
                        required.insert(rule.name.clone());
                        permissions.push(Permission {
                            name: rule.name.clone(),
                            required: true,
                            reason: format!("Syscall '{}' requires capability", syscall_name),
                        });
                    }
                }
            }
        }

        // Add specific syscall capability
        let capability_name = format!("Syscall_{}", syscall_name);
        required.insert(capability_name.clone());
        permissions.push(Permission {
            name: capability_name,
            required: true,
            reason: format!("Direct syscall access: {}", syscall_name),
        });
    }

    pub fn recommend_sandbox_constraints(&self, capabilities: &CapabilityRequirements, 
                                       security: &SecurityAssessment) -> SandboxConstraints {
        let mut constraints = SandboxConstraints::default();

        // Memory constraints
        if capabilities.required_capabilities.contains("MemoryGrow") {
            constraints.max_memory_pages = 256; // 16MB
        } else {
            constraints.max_memory_pages = 64; // 4MB
        }

        if capabilities.required_capabilities.contains("UnlimitedMemory") {
            constraints.max_memory_pages = 1024; // 64MB but still limited
            constraints.memory_growth_rate_limit = Some(10); // pages per second
        }

        // CPU constraints
        if capabilities.required_capabilities.contains("HighComplexity") {
            constraints.max_cpu_time_ms = 10000; // 10 seconds
            constraints.max_instructions = 10_000_000;
        } else {
            constraints.max_cpu_time_ms = 5000; // 5 seconds
            constraints.max_instructions = 1_000_000;
        }

        if capabilities.required_capabilities.contains("LongRunning") {
            constraints.max_cpu_time_ms = 30000; // 30 seconds
            constraints.preemption_interval_ms = 100; // Frequent preemption
        }

        // Syscall constraints
        if capabilities.required_capabilities.iter().any(|c| c.starts_with("Syscall_")) {
            constraints.max_syscalls = 1000;
            constraints.syscall_rate_limit = Some(100); // per second
        }

        // Network/FS constraints (very restrictive)
        if capabilities.required_capabilities.contains("NetworkAccess") {
            constraints.network_allowed = false; // Default deny
            constraints.max_network_connections = 0;
        }

        if capabilities.required_capabilities.contains("FileSystemAccess") {
            constraints.filesystem_allowed = false; // Default deny
            constraints.allowed_paths = vec![];
        }

        constraints
    }
}

#[derive(Debug, Clone)]
pub struct SandboxConstraints {
    pub max_memory_pages: u32,
    pub max_cpu_time_ms: u64,
    pub max_instructions: u64,
    pub max_syscalls: u32,
    pub preemption_interval_ms: u64,
    pub memory_growth_rate_limit: Option<u32>,
    pub syscall_rate_limit: Option<u32>,
    pub network_allowed: bool,
    pub max_network_connections: u32,
    pub filesystem_allowed: bool,
    pub allowed_paths: Vec<String>,
}

impl Default for SandboxConstraints {
    fn default() -> Self {
        Self {
            max_memory_pages: 64,
            max_cpu_time_ms: 5000,
            max_instructions: 1_000_000,
            max_syscalls: 100,
            preemption_interval_ms: 1000,
            memory_growth_rate_limit: None,
            syscall_rate_limit: None,
            network_allowed: false,
            max_network_connections: 0,
            filesystem_allowed: false,
            allowed_paths: vec![],
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser::{WasmModule, Export, ExportKind, MemoryType};

    #[test]
    fn test_capability_inference() {
        let inferrer = CapabilityInferrer::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: Some(MemoryType { min: 1, max: Some(10) }),
            exports: vec![
                Export {
                    name: "wasm_log".to_string(),
                    kind: ExportKind::Function,
                    index: 0,
                }
            ],
            code: vec![],
        };

        let security = SecurityAssessment {
            memory_patterns: vec![],
            control_flow_complexity: 5,
            suspicious_patterns: vec![],
            syscall_functions: vec![],
            resource_requirements: super::ResourceRequirements {
                estimated_memory: 65536,
                estimated_cpu_cycles: 1000,
                max_stack_depth: 10,
                max_call_depth: 5,
            },
        };

        let result = inferrer.infer(&module, &security);
        assert!(result.is_ok());

        let capabilities = result.unwrap();
        assert!(capabilities.required_capabilities.contains(&"Log".to_string()));
    }

    #[test]
    fn test_sandbox_constraints() {
        let inferrer = CapabilityInferrer::new();
        let capabilities = CapabilityRequirements {
            required_capabilities: vec!["MemoryGrow".to_string(), "HighComplexity".to_string()],
            optional_capabilities: vec![],
            inferred_permissions: vec![],
        };

        let security = SecurityAssessment {
            memory_patterns: vec![],
            control_flow_complexity: 150,
            suspicious_patterns: vec![],
            syscall_functions: vec![],
            resource_requirements: super::ResourceRequirements {
                estimated_memory: 1048576,
                estimated_cpu_cycles: 100000,
                max_stack_depth: 50,
                max_call_depth: 20,
            },
        };

        let constraints = inferrer.recommend_sandbox_constraints(&capabilities, &security);
        assert_eq!(constraints.max_memory_pages, 256);
        assert_eq!(constraints.max_cpu_time_ms, 10000);
    }

    #[test]
    fn test_high_risk_capabilities() {
        let inferrer = CapabilityInferrer::new();
        let module = WasmModule {
            types: vec![],
            functions: vec![],
            memory: None,
            exports: vec![
                Export {
                    name: "socket_connect".to_string(),
                    kind: ExportKind::Function,
                    index: 0,
                }
            ],
            code: vec![],
        };

        let security = SecurityAssessment {
            memory_patterns: vec![],
            control_flow_complexity: 5,
            suspicious_patterns: vec![],
            syscall_functions: vec![
                super::SyscallFunction {
                    name: "socket".to_string(),
                    import_index: 0,
                    usage_count: 1,
                    risk_level: RiskLevel::Severe,
                }
            ],
            resource_requirements: super::ResourceRequirements {
                estimated_memory: 65536,
                estimated_cpu_cycles: 1000,
                max_stack_depth: 10,
                max_call_depth: 5,
            },
        };

        let result = inferrer.infer(&module, &security);
        assert!(result.is_ok());

        let capabilities = result.unwrap();
        assert!(capabilities.required_capabilities.contains(&"NetworkAccess".to_string()));
        
        let constraints = inferrer.recommend_sandbox_constraints(&capabilities, &security);
        assert!(!constraints.network_allowed);
    }
}