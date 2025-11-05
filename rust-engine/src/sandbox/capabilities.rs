use std::collections::HashSet;

#[derive(Debug, Clone, Hash, PartialEq, Eq)]
pub enum Capability {
    // Logging capabilities
    Log,
    
    // Sensor access
    ReadSensor(SensorType),
    
    // Alert system
    SendAlert(AlertLevel),
    
    // Time access
    GetTime,
    
    // Random number generation
    Random,
    
    // Memory operations
    MemoryGrow,
    
    // Network (if ever needed)
    NetworkAccess,
    
    // File system (if ever needed)
    FileSystemRead,
    FileSystemWrite,
}

#[derive(Debug, Clone, Hash, PartialEq, Eq)]
pub enum SensorType {
    Temperature,
    Humidity,
    Pressure,
    Motion,
    Light,
    Sound,
    Any,
}

#[derive(Debug, Clone, Hash, PartialEq, Eq)]
pub enum AlertLevel {
    Info,
    Warning,
    Error,
    Critical,
    Any,
}

#[derive(Debug)]
pub struct CapabilitySet {
    capabilities: HashSet<Capability>,
}

impl CapabilitySet {
    pub fn new() -> Self {
        Self {
            capabilities: HashSet::new(),
        }
    }
    
    pub fn with_basic() -> Self {
        let mut set = Self::new();
        set.grant(Capability::Log);
        set.grant(Capability::GetTime);
        set.grant(Capability::Random);
        set
    }
    
    pub fn with_sensor_access() -> Self {
        let mut set = Self::with_basic();
        set.grant(Capability::ReadSensor(SensorType::Any));
        set
    }
    
    pub fn with_alert_system() -> Self {
        let mut set = Self::with_basic();
        set.grant(Capability::SendAlert(AlertLevel::Any));
        set
    }
    
    pub fn grant(&mut self, capability: Capability) {
        self.capabilities.insert(capability);
    }
    
    pub fn revoke(&mut self, capability: &Capability) {
        self.capabilities.remove(capability);
    }
    
    pub fn has(&self, capability: &Capability) -> bool {
        // Check exact match first
        if self.capabilities.contains(capability) {
            return true;
        }
        
        // Check for wildcard permissions
        match capability {
            Capability::ReadSensor(_) => {
                self.capabilities.contains(&Capability::ReadSensor(SensorType::Any))
            }
            Capability::SendAlert(_) => {
                self.capabilities.contains(&Capability::SendAlert(AlertLevel::Any))
            }
            _ => false,
        }
    }
    
    pub fn can_read_sensor(&self, sensor_type: &SensorType) -> bool {
        self.has(&Capability::ReadSensor(sensor_type.clone())) ||
        self.has(&Capability::ReadSensor(SensorType::Any))
    }
    
    pub fn can_send_alert(&self, level: &AlertLevel) -> bool {
        self.has(&Capability::SendAlert(level.clone())) ||
        self.has(&Capability::SendAlert(AlertLevel::Any))
    }
    
    pub fn list_capabilities(&self) -> Vec<&Capability> {
        self.capabilities.iter().collect()
    }
    
    pub fn clear(&mut self) {
        self.capabilities.clear();
    }
}

#[derive(Debug)]
pub struct CapabilityPolicy {
    default_capabilities: CapabilitySet,
    restricted_capabilities: HashSet<Capability>,
}

impl CapabilityPolicy {
    pub fn new() -> Self {
        Self {
            default_capabilities: CapabilitySet::with_basic(),
            restricted_capabilities: HashSet::new(),
        }
    }
    
    pub fn strict() -> Self {
        let mut policy = Self::new();
        policy.default_capabilities.clear();
        policy.restrict(Capability::NetworkAccess);
        policy.restrict(Capability::FileSystemRead);
        policy.restrict(Capability::FileSystemWrite);
        policy
    }
    
    pub fn restrict(&mut self, capability: Capability) {
        self.restricted_capabilities.insert(capability);
    }
    
    pub fn is_restricted(&self, capability: &Capability) -> bool {
        self.restricted_capabilities.contains(capability)
    }
    
    pub fn apply_to(&self, capability_set: &mut CapabilitySet) {
        // Remove restricted capabilities
        for restricted in &self.restricted_capabilities {
            capability_set.revoke(restricted);
        }
    }
    
    pub fn create_capability_set(&self) -> CapabilitySet {
        let mut set = CapabilitySet::new();
        
        // Add default capabilities
        for capability in self.default_capabilities.list_capabilities() {
            if !self.is_restricted(capability) {
                set.grant(capability.clone());
            }
        }
        
        set
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_capability_set_basic() {
        let mut caps = CapabilitySet::new();
        caps.grant(Capability::Log);
        
        assert!(caps.has(&Capability::Log));
        assert!(!caps.has(&Capability::GetTime));
    }

    #[test]
    fn test_sensor_wildcard() {
        let mut caps = CapabilitySet::new();
        caps.grant(Capability::ReadSensor(SensorType::Any));
        
        assert!(caps.can_read_sensor(&SensorType::Temperature));
        assert!(caps.can_read_sensor(&SensorType::Humidity));
    }

    #[test]
    fn test_alert_wildcard() {
        let mut caps = CapabilitySet::new();
        caps.grant(Capability::SendAlert(AlertLevel::Any));
        
        assert!(caps.can_send_alert(&AlertLevel::Info));
        assert!(caps.can_send_alert(&AlertLevel::Critical));
    }

    #[test]
    fn test_capability_policy() {
        let mut policy = CapabilityPolicy::strict();
        policy.restrict(Capability::Log);
        
        let caps = policy.create_capability_set();
        assert!(!caps.has(&Capability::Log));
    }
}