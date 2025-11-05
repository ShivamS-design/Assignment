use super::capabilities::{Capability, CapabilitySet, SensorType, AlertLevel};
use super::limits::ResourceLimits;
use std::collections::HashMap;
use std::time::Duration;

#[derive(Debug, Clone)]
pub struct SecurityPolicy {
    pub name: String,
    pub description: String,
    pub resource_limits: ResourceLimits,
    pub allowed_capabilities: CapabilitySet,
    pub syscall_whitelist: Vec<String>,
    pub syscall_blacklist: Vec<String>,
    pub network_policy: NetworkPolicy,
    pub file_policy: FilePolicy,
}

#[derive(Debug, Clone)]
pub struct NetworkPolicy {
    pub allow_outbound: bool,
    pub allow_inbound: bool,
    pub allowed_hosts: Vec<String>,
    pub allowed_ports: Vec<u16>,
}

#[derive(Debug, Clone)]
pub struct FilePolicy {
    pub allow_read: bool,
    pub allow_write: bool,
    pub allowed_paths: Vec<String>,
    pub denied_paths: Vec<String>,
}

impl Default for NetworkPolicy {
    fn default() -> Self {
        Self {
            allow_outbound: false,
            allow_inbound: false,
            allowed_hosts: Vec::new(),
            allowed_ports: Vec::new(),
        }
    }
}

impl Default for FilePolicy {
    fn default() -> Self {
        Self {
            allow_read: false,
            allow_write: false,
            allowed_paths: Vec::new(),
            denied_paths: Vec::new(),
        }
    }
}

impl SecurityPolicy {
    pub fn strict() -> Self {
        let mut capabilities = CapabilitySet::new();
        capabilities.grant(Capability::Log);
        
        Self {
            name: "Strict".to_string(),
            description: "Minimal permissions for untrusted code".to_string(),
            resource_limits: ResourceLimits::strict(),
            allowed_capabilities: capabilities,
            syscall_whitelist: vec!["wasm_log".to_string()],
            syscall_blacklist: vec![
                "open".to_string(),
                "read".to_string(),
                "write".to_string(),
                "socket".to_string(),
                "connect".to_string(),
                "exec".to_string(),
                "fork".to_string(),
            ],
            network_policy: NetworkPolicy::default(),
            file_policy: FilePolicy::default(),
        }
    }
    
    pub fn sensor_access() -> Self {
        let mut capabilities = CapabilitySet::with_basic();
        capabilities.grant(Capability::ReadSensor(SensorType::Any));
        
        Self {
            name: "Sensor Access".to_string(),
            description: "Basic permissions with sensor access".to_string(),
            resource_limits: ResourceLimits::default(),
            allowed_capabilities: capabilities,
            syscall_whitelist: vec![
                "wasm_log".to_string(),
                "wasm_get_time".to_string(),
                "wasm_random".to_string(),
                "wasm_read_sensor".to_string(),
            ],
            syscall_blacklist: vec![
                "open".to_string(),
                "socket".to_string(),
                "exec".to_string(),
                "fork".to_string(),
            ],
            network_policy: NetworkPolicy::default(),
            file_policy: FilePolicy::default(),
        }
    }
    
    pub fn alert_system() -> Self {
        let mut capabilities = CapabilitySet::with_basic();
        capabilities.grant(Capability::ReadSensor(SensorType::Any));
        capabilities.grant(Capability::SendAlert(AlertLevel::Any));
        
        Self {
            name: "Alert System".to_string(),
            description: "Sensor access with alerting capabilities".to_string(),
            resource_limits: ResourceLimits::default(),
            allowed_capabilities: capabilities,
            syscall_whitelist: vec![
                "wasm_log".to_string(),
                "wasm_get_time".to_string(),
                "wasm_random".to_string(),
                "wasm_read_sensor".to_string(),
                "wasm_send_alert".to_string(),
            ],
            syscall_blacklist: vec![
                "open".to_string(),
                "socket".to_string(),
                "exec".to_string(),
                "fork".to_string(),
            ],
            network_policy: NetworkPolicy::default(),
            file_policy: FilePolicy::default(),
        }
    }
    
    pub fn development() -> Self {
        let mut capabilities = CapabilitySet::with_basic();
        capabilities.grant(Capability::ReadSensor(SensorType::Any));
        capabilities.grant(Capability::SendAlert(AlertLevel::Any));
        capabilities.grant(Capability::MemoryGrow);
        
        Self {
            name: "Development".to_string(),
            description: "Permissive policy for development and testing".to_string(),
            resource_limits: ResourceLimits::permissive(),
            allowed_capabilities: capabilities,
            syscall_whitelist: vec![
                "wasm_log".to_string(),
                "wasm_get_time".to_string(),
                "wasm_random".to_string(),
                "wasm_read_sensor".to_string(),
                "wasm_send_alert".to_string(),
            ],
            syscall_blacklist: vec![
                "exec".to_string(),
                "fork".to_string(),
            ],
            network_policy: NetworkPolicy::default(),
            file_policy: FilePolicy::default(),
        }
    }
    
    pub fn custom() -> PolicyBuilder {
        PolicyBuilder::new()
    }
    
    pub fn is_syscall_allowed(&self, syscall: &str) -> bool {
        if self.syscall_blacklist.contains(&syscall.to_string()) {
            return false;
        }
        
        if !self.syscall_whitelist.is_empty() {
            return self.syscall_whitelist.contains(&syscall.to_string());
        }
        
        true
    }
    
    pub fn validate(&self) -> Result<(), String> {
        // Check for conflicting syscall policies
        for syscall in &self.syscall_whitelist {
            if self.syscall_blacklist.contains(syscall) {
                return Err(format!("Syscall '{}' is both whitelisted and blacklisted", syscall));
            }
        }
        
        // Validate resource limits
        if self.resource_limits.max_memory_pages == 0 {
            return Err("Memory limit cannot be zero".to_string());
        }
        
        if self.resource_limits.max_cpu_time == Duration::ZERO {
            return Err("CPU time limit cannot be zero".to_string());
        }
        
        Ok(())
    }
}

#[derive(Debug)]
pub struct PolicyBuilder {
    policy: SecurityPolicy,
}

impl PolicyBuilder {
    pub fn new() -> Self {
        Self {
            policy: SecurityPolicy {
                name: "Custom".to_string(),
                description: "Custom security policy".to_string(),
                resource_limits: ResourceLimits::default(),
                allowed_capabilities: CapabilitySet::new(),
                syscall_whitelist: Vec::new(),
                syscall_blacklist: Vec::new(),
                network_policy: NetworkPolicy::default(),
                file_policy: FilePolicy::default(),
            },
        }
    }
    
    pub fn name(mut self, name: &str) -> Self {
        self.policy.name = name.to_string();
        self
    }
    
    pub fn description(mut self, description: &str) -> Self {
        self.policy.description = description.to_string();
        self
    }
    
    pub fn resource_limits(mut self, limits: ResourceLimits) -> Self {
        self.policy.resource_limits = limits;
        self
    }
    
    pub fn capability(mut self, capability: Capability) -> Self {
        self.policy.allowed_capabilities.grant(capability);
        self
    }
    
    pub fn allow_syscall(mut self, syscall: &str) -> Self {
        self.policy.syscall_whitelist.push(syscall.to_string());
        self
    }
    
    pub fn deny_syscall(mut self, syscall: &str) -> Self {
        self.policy.syscall_blacklist.push(syscall.to_string());
        self
    }
    
    pub fn network_policy(mut self, policy: NetworkPolicy) -> Self {
        self.policy.network_policy = policy;
        self
    }
    
    pub fn file_policy(mut self, policy: FilePolicy) -> Self {
        self.policy.file_policy = policy;
        self
    }
    
    pub fn build(self) -> Result<SecurityPolicy, String> {
        self.policy.validate()?;
        Ok(self.policy)
    }
}

#[derive(Debug)]
pub struct PolicyManager {
    policies: HashMap<String, SecurityPolicy>,
    default_policy: String,
}

impl PolicyManager {
    pub fn new() -> Self {
        let mut manager = Self {
            policies: HashMap::new(),
            default_policy: "strict".to_string(),
        };
        
        // Register built-in policies
        manager.register_policy(SecurityPolicy::strict());
        manager.register_policy(SecurityPolicy::sensor_access());
        manager.register_policy(SecurityPolicy::alert_system());
        manager.register_policy(SecurityPolicy::development());
        
        manager
    }
    
    pub fn register_policy(&mut self, policy: SecurityPolicy) {
        let name = policy.name.to_lowercase();
        self.policies.insert(name, policy);
    }
    
    pub fn get_policy(&self, name: &str) -> Option<&SecurityPolicy> {
        self.policies.get(&name.to_lowercase())
    }
    
    pub fn get_default_policy(&self) -> &SecurityPolicy {
        self.policies.get(&self.default_policy).unwrap()
    }
    
    pub fn set_default_policy(&mut self, name: &str) -> Result<(), String> {
        if !self.policies.contains_key(&name.to_lowercase()) {
            return Err(format!("Policy '{}' not found", name));
        }
        
        self.default_policy = name.to_lowercase();
        Ok(())
    }
    
    pub fn list_policies(&self) -> Vec<&str> {
        self.policies.keys().map(|s| s.as_str()).collect()
    }
    
    pub fn remove_policy(&mut self, name: &str) -> bool {
        self.policies.remove(&name.to_lowercase()).is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_security_policy_creation() {
        let policy = SecurityPolicy::strict();
        assert_eq!(policy.name, "Strict");
        assert!(policy.validate().is_ok());
    }

    #[test]
    fn test_policy_builder() {
        let policy = SecurityPolicy::custom()
            .name("Test Policy")
            .capability(Capability::Log)
            .allow_syscall("wasm_log")
            .deny_syscall("open")
            .build();
        
        assert!(policy.is_ok());
        let policy = policy.unwrap();
        assert_eq!(policy.name, "Test Policy");
        assert!(policy.is_syscall_allowed("wasm_log"));
        assert!(!policy.is_syscall_allowed("open"));
    }

    #[test]
    fn test_policy_manager() {
        let mut manager = PolicyManager::new();
        
        assert!(manager.get_policy("strict").is_some());
        assert!(manager.get_policy("nonexistent").is_none());
        
        let policies = manager.list_policies();
        assert!(policies.contains(&"strict"));
        assert!(policies.contains(&"development"));
    }

    #[test]
    fn test_conflicting_syscalls() {
        let policy = SecurityPolicy::custom()
            .allow_syscall("test")
            .deny_syscall("test")
            .build();
        
        assert!(policy.is_err());
    }
}