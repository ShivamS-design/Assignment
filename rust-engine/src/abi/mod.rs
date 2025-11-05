pub mod functions;
pub mod host_interface;

use crate::error::{WasmError, Result};
use crate::memory::LinearMemory;
use crate::sandbox::capabilities::{Capability, SensorType, AlertLevel};
use crate::sandbox::Sandbox;

#[derive(Debug)]
pub struct WasmABI {
    sandbox: Sandbox,
    host_interface: host_interface::HostInterface,
}

impl WasmABI {
    pub fn new(sandbox: Sandbox) -> Self {
        Self {
            sandbox,
            host_interface: host_interface::HostInterface::new(),
        }
    }
    
    pub fn call_host_function(
        &mut self,
        name: &str,
        args: &[u32],
        memory: &mut LinearMemory,
    ) -> Result<u32> {
        match name {
            "wasm_log" => self.wasm_log(args, memory),
            "wasm_read_sensor" => self.wasm_read_sensor(args),
            "wasm_send_alert" => self.wasm_send_alert(args, memory),
            "wasm_get_time" => self.wasm_get_time(),
            "wasm_random" => self.wasm_random(),
            _ => Err(WasmError::Runtime(format!("Unknown host function: {}", name))),
        }
    }
    
    fn wasm_log(&mut self, args: &[u32], memory: &LinearMemory) -> Result<u32> {
        self.sandbox.check_capability(&Capability::Log)?;
        
        if args.len() < 2 {
            return Err(WasmError::Runtime("wasm_log requires 2 arguments".to_string()));
        }
        
        let message_ptr = args[0];
        let message_len = args[1];
        
        let message_bytes = memory.read_bytes(message_ptr, message_len)?;
        let message = String::from_utf8_lossy(message_bytes);
        
        self.host_interface.log(&message);
        Ok(0)
    }
    
    fn wasm_read_sensor(&mut self, args: &[u32]) -> Result<u32> {
        if args.is_empty() {
            return Err(WasmError::Runtime("wasm_read_sensor requires 1 argument".to_string()));
        }
        
        let sensor_id = args[0];
        let sensor_type = match sensor_id {
            0 => SensorType::Temperature,
            1 => SensorType::Humidity,
            2 => SensorType::Pressure,
            3 => SensorType::Motion,
            4 => SensorType::Light,
            5 => SensorType::Sound,
            _ => return Err(WasmError::Runtime("Invalid sensor ID".to_string())),
        };
        
        self.sandbox.check_capability(&Capability::ReadSensor(sensor_type.clone()))?;
        
        let value = self.host_interface.read_sensor(sensor_id)?;
        Ok(value)
    }
    
    fn wasm_send_alert(&mut self, args: &[u32], memory: &LinearMemory) -> Result<u32> {
        if args.len() < 3 {
            return Err(WasmError::Runtime("wasm_send_alert requires 3 arguments".to_string()));
        }
        
        let level = args[0];
        let message_ptr = args[1];
        let message_len = args[2];
        
        let alert_level = match level {
            0 => AlertLevel::Info,
            1 => AlertLevel::Warning,
            2 => AlertLevel::Error,
            3 => AlertLevel::Critical,
            _ => return Err(WasmError::Runtime("Invalid alert level".to_string())),
        };
        
        self.sandbox.check_capability(&Capability::SendAlert(alert_level.clone()))?;
        
        let message_bytes = memory.read_bytes(message_ptr, message_len)?;
        let message = String::from_utf8_lossy(message_bytes);
        
        self.host_interface.send_alert(level, &message)?;
        Ok(0)
    }
    
    fn wasm_get_time(&mut self) -> Result<u32> {
        self.sandbox.check_capability(&Capability::GetTime)?;
        
        let timestamp = self.host_interface.get_time()?;
        Ok(timestamp)
    }
    
    fn wasm_random(&mut self) -> Result<u32> {
        self.sandbox.check_capability(&Capability::Random)?;
        
        let value = self.host_interface.random()?;
        Ok(value)
    }
    
    pub fn get_sandbox_mut(&mut self) -> &mut Sandbox {
        &mut self.sandbox
    }
    
    pub fn get_host_interface(&self) -> &host_interface::HostInterface {
        &self.host_interface
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sandbox::{ResourceLimits, Sandbox};
    use crate::sandbox::capabilities::CapabilitySet;

    #[test]
    fn test_abi_creation() {
        let sandbox = Sandbox::new(ResourceLimits::default());
        let abi = WasmABI::new(sandbox);
        assert!(abi.get_host_interface().is_initialized());
    }

    #[test]
    fn test_wasm_get_time() {
        let mut sandbox = Sandbox::new(ResourceLimits::default());
        sandbox.grant_capability(Capability::GetTime);
        let mut abi = WasmABI::new(sandbox);
        
        let result = abi.wasm_get_time();
        assert!(result.is_ok());
        assert!(result.unwrap() > 0);
    }

    #[test]
    fn test_capability_check() {
        let sandbox = Sandbox::new(ResourceLimits::default());
        let mut abi = WasmABI::new(sandbox);
        
        // Should fail without capability
        let result = abi.wasm_get_time();
        assert!(result.is_err());
    }
}