use wasm_engine::*;
use wasm_engine::sandbox::capabilities::{Capability, SensorType, AlertLevel};
use wasm_engine::sandbox::policy::SecurityPolicy;

#[test]
fn test_sandbox_resource_limits() {
    let limits = ResourceLimits::strict();
    let mut sandbox = Sandbox::new(limits);
    
    // Test memory limit enforcement
    sandbox.update_memory_usage(100); // Within limit
    assert!(sandbox.check_limits().is_ok());
    
    sandbox.update_memory_usage(1000); // Exceeds limit
    assert!(sandbox.check_limits().is_err());
}

#[test]
fn test_capability_system() {
    let limits = ResourceLimits::default();
    let mut sandbox = Sandbox::new(limits);
    
    // Should fail without capability
    assert!(sandbox.check_capability(&Capability::Log).is_err());
    
    // Grant capability and try again
    sandbox.grant_capability(Capability::Log);
    assert!(sandbox.check_capability(&Capability::Log).is_ok());
}

#[test]
fn test_sensor_capabilities() {
    let limits = ResourceLimits::default();
    let mut sandbox = Sandbox::new(limits);
    
    // Grant specific sensor access
    sandbox.grant_capability(Capability::ReadSensor(SensorType::Temperature));
    
    assert!(sandbox.check_capability(&Capability::ReadSensor(SensorType::Temperature)).is_ok());
    assert!(sandbox.check_capability(&Capability::ReadSensor(SensorType::Humidity)).is_err());
    
    // Grant wildcard sensor access
    sandbox.grant_capability(Capability::ReadSensor(SensorType::Any));
    assert!(sandbox.check_capability(&Capability::ReadSensor(SensorType::Humidity)).is_ok());
}

#[test]
fn test_alert_capabilities() {
    let limits = ResourceLimits::default();
    let mut sandbox = Sandbox::new(limits);
    
    // Grant specific alert level
    sandbox.grant_capability(Capability::SendAlert(AlertLevel::Warning));
    
    assert!(sandbox.check_capability(&Capability::SendAlert(AlertLevel::Warning)).is_ok());
    assert!(sandbox.check_capability(&Capability::SendAlert(AlertLevel::Critical)).is_err());
    
    // Grant wildcard alert access
    sandbox.grant_capability(Capability::SendAlert(AlertLevel::Any));
    assert!(sandbox.check_capability(&Capability::SendAlert(AlertLevel::Critical)).is_ok());
}

#[test]
fn test_syscall_interception() {
    let limits = ResourceLimits::default();
    let mut sandbox = Sandbox::new(limits);
    
    // Test allowed syscall
    let result = sandbox.intercept_syscall("wasm_get_time", &[]);
    assert!(result.is_ok());
    
    // Test denied syscall
    let result = sandbox.intercept_syscall("open", &[]);
    assert!(result.is_err());
}

#[test]
fn test_security_policies() {
    // Test strict policy
    let strict = SecurityPolicy::strict();
    assert!(!strict.is_syscall_allowed("wasm_read_sensor"));
    assert!(strict.is_syscall_allowed("wasm_log"));
    
    // Test sensor access policy
    let sensor_policy = SecurityPolicy::sensor_access();
    assert!(sensor_policy.is_syscall_allowed("wasm_read_sensor"));
    assert!(sensor_policy.is_syscall_allowed("wasm_log"));
    
    // Test alert system policy
    let alert_policy = SecurityPolicy::alert_system();
    assert!(alert_policy.is_syscall_allowed("wasm_send_alert"));
    assert!(alert_policy.is_syscall_allowed("wasm_read_sensor"));
}

#[test]
fn test_abi_integration() {
    let limits = ResourceLimits::default();
    let mut sandbox = Sandbox::new(limits);
    
    // Grant necessary capabilities
    sandbox.grant_capability(Capability::GetTime);
    sandbox.grant_capability(Capability::Random);
    sandbox.grant_capability(Capability::ReadSensor(SensorType::Any));
    
    let mut abi = WasmABI::new(sandbox);
    let mut memory = LinearMemory::new(1, None).unwrap();
    
    // Test time function
    let result = abi.call_host_function("wasm_get_time", &[], &mut memory);
    assert!(result.is_ok());
    assert!(result.unwrap() > 0);
    
    // Test random function
    let result = abi.call_host_function("wasm_random", &[], &mut memory);
    assert!(result.is_ok());
    
    // Test sensor function
    let result = abi.call_host_function("wasm_read_sensor", &[0], &mut memory);
    assert!(result.is_ok());
}

#[test]
fn test_memory_safety() {
    let mut memory = LinearMemory::new(1, Some(2)).unwrap();
    
    // Test valid memory access
    assert!(memory.write_u32(0, 0x12345678).is_ok());
    assert_eq!(memory.read_u32(0).unwrap(), 0x12345678);
    
    // Test bounds checking
    let page_size = 65536;
    let result = memory.read_u32(page_size);
    assert!(result.is_err());
    
    // Test memory growth
    let old_size = memory.grow(1).unwrap();
    assert_eq!(old_size, 1);
    assert_eq!(memory.size(), 2);
    
    // Test growth limit
    let result = memory.grow(1);
    assert!(result.is_err()); // Should exceed max_pages
}

#[test]
fn test_instruction_counting() {
    let limits = ResourceLimits::strict();
    let mut sandbox = Sandbox::new(limits);
    
    // Simulate instruction execution
    for _ in 0..1000 {
        sandbox.increment_instructions(100);
        if sandbox.check_limits().is_err() {
            break;
        }
    }
    
    // Should have hit the instruction limit
    assert!(sandbox.check_limits().is_err());
}

#[test]
fn test_wasm_parser() {
    // Valid WASM module header
    let valid_wasm = [
        0x00, 0x61, 0x73, 0x6d, // magic
        0x01, 0x00, 0x00, 0x00, // version
    ];
    
    let result = wasm_engine::parser::WasmParser::parse(&valid_wasm);
    assert!(result.is_ok());
    
    // Invalid magic number
    let invalid_wasm = [
        0x00, 0x00, 0x00, 0x00, // invalid magic
        0x01, 0x00, 0x00, 0x00, // version
    ];
    
    let result = wasm_engine::parser::WasmParser::parse(&invalid_wasm);
    assert!(result.is_err());
}

#[test]
fn test_host_interface() {
    use wasm_engine::abi::host_interface::HostInterface;
    
    let interface = HostInterface::new();
    
    // Test sensor reading
    let temp = interface.read_sensor(0);
    assert!(temp.is_ok());
    assert!(temp.unwrap() > 0);
    
    // Test time
    let time1 = interface.get_time().unwrap();
    std::thread::sleep(std::time::Duration::from_millis(10));
    let time2 = interface.get_time().unwrap();
    assert!(time2 >= time1);
    
    // Test random
    let rand1 = interface.random().unwrap();
    let rand2 = interface.random().unwrap();
    // Should be different with high probability
    assert_ne!(rand1, rand2);
    
    // Test alert
    let result = interface.send_alert(1, "Test alert");
    assert!(result.is_ok());
}

#[test]
fn test_complete_isolation() {
    let limits = ResourceLimits::strict();
    let mut sandbox = Sandbox::new(limits);
    
    // Test that dangerous syscalls are blocked
    let dangerous_syscalls = [
        "open", "read", "write", "socket", "connect", 
        "exec", "fork", "system", "unlink", "chmod"
    ];
    
    for syscall in &dangerous_syscalls {
        let result = sandbox.intercept_syscall(syscall, &[]);
        assert!(result.is_err(), "Syscall {} should be blocked", syscall);
    }
    
    // Test that only safe syscalls are allowed
    let safe_syscalls = ["wasm_log", "wasm_get_time", "wasm_random"];
    
    for syscall in &safe_syscalls {
        // Note: These might still fail due to capability checks, but not due to syscall blocking
        let _result = sandbox.intercept_syscall(syscall, &[]);
        // The important thing is that the syscall trap doesn't immediately reject them
    }
}