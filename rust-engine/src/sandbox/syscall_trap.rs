use crate::error::{WasmError, Result};
use std::collections::{HashMap, HashSet};

#[derive(Debug)]
pub struct SyscallTrap {
    allowed_syscalls: HashSet<String>,
    syscall_handlers: HashMap<String, Box<dyn SyscallHandler>>,
    syscall_log: Vec<SyscallEntry>,
}

#[derive(Debug, Clone)]
pub struct SyscallEntry {
    pub name: String,
    pub args: Vec<u32>,
    pub result: Result<u32>,
    pub timestamp: std::time::Instant,
}

pub trait SyscallHandler: std::fmt::Debug {
    fn handle(&self, args: &[u32]) -> Result<u32>;
}

#[derive(Debug)]
struct DeniedSyscallHandler;

impl SyscallHandler for DeniedSyscallHandler {
    fn handle(&self, _args: &[u32]) -> Result<u32> {
        Err(WasmError::Runtime("Syscall denied by policy".to_string()))
    }
}

#[derive(Debug)]
struct LogSyscallHandler;

impl SyscallHandler for LogSyscallHandler {
    fn handle(&self, args: &[u32]) -> Result<u32> {
        if args.len() < 2 {
            return Err(WasmError::Runtime("Invalid log syscall arguments".to_string()));
        }
        
        let _message_ptr = args[0];
        let _message_len = args[1];
        
        // In a real implementation, we would read the message from WASM memory
        log::info!("WASM log: [message at {}:{}]", _message_ptr, _message_len);
        Ok(0)
    }
}

#[derive(Debug)]
struct TimeSyscallHandler;

impl SyscallHandler for TimeSyscallHandler {
    fn handle(&self, _args: &[u32]) -> Result<u32> {
        use std::time::{SystemTime, UNIX_EPOCH};
        
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|_| WasmError::Runtime("Time error".to_string()))?
            .as_secs() as u32;
        
        Ok(timestamp)
    }
}

#[derive(Debug)]
struct RandomSyscallHandler;

impl SyscallHandler for RandomSyscallHandler {
    fn handle(&self, _args: &[u32]) -> Result<u32> {
        // Simple pseudo-random number generator
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        use std::time::SystemTime;
        
        let mut hasher = DefaultHasher::new();
        SystemTime::now().hash(&mut hasher);
        Ok(hasher.finish() as u32)
    }
}

#[derive(Debug)]
struct SensorSyscallHandler;

impl SyscallHandler for SensorSyscallHandler {
    fn handle(&self, args: &[u32]) -> Result<u32> {
        if args.is_empty() {
            return Err(WasmError::Runtime("Invalid sensor syscall arguments".to_string()));
        }
        
        let sensor_id = args[0];
        
        // Simulate sensor readings
        let value = match sensor_id {
            0 => 25, // Temperature: 25°C
            1 => 60, // Humidity: 60%
            2 => 1013, // Pressure: 1013 hPa
            3 => 0, // Motion: no motion
            4 => 500, // Light: 500 lux
            5 => 40, // Sound: 40 dB
            _ => return Err(WasmError::Runtime("Unknown sensor ID".to_string())),
        };
        
        log::info!("WASM sensor read: sensor_id={}, value={}", sensor_id, value);
        Ok(value)
    }
}

#[derive(Debug)]
struct AlertSyscallHandler;

impl SyscallHandler for AlertSyscallHandler {
    fn handle(&self, args: &[u32]) -> Result<u32> {
        if args.len() < 3 {
            return Err(WasmError::Runtime("Invalid alert syscall arguments".to_string()));
        }
        
        let level = args[0];
        let _message_ptr = args[1];
        let _message_len = args[2];
        
        let level_str = match level {
            0 => "INFO",
            1 => "WARNING", 
            2 => "ERROR",
            3 => "CRITICAL",
            _ => "UNKNOWN",
        };
        
        log::warn!("WASM alert [{}]: [message at {}:{}]", level_str, _message_ptr, _message_len);
        Ok(0)
    }
}

impl SyscallTrap {
    pub fn new() -> Self {
        let mut trap = Self {
            allowed_syscalls: HashSet::new(),
            syscall_handlers: HashMap::new(),
            syscall_log: Vec::new(),
        };
        
        trap.setup_default_handlers();
        trap
    }
    
    fn setup_default_handlers(&mut self) {
        // Register allowed syscalls and their handlers
        self.register_syscall("wasm_log", Box::new(LogSyscallHandler));
        self.register_syscall("wasm_get_time", Box::new(TimeSyscallHandler));
        self.register_syscall("wasm_random", Box::new(RandomSyscallHandler));
        self.register_syscall("wasm_read_sensor", Box::new(SensorSyscallHandler));
        self.register_syscall("wasm_send_alert", Box::new(AlertSyscallHandler));
        
        // Explicitly deny dangerous syscalls
        self.register_syscall("open", Box::new(DeniedSyscallHandler));
        self.register_syscall("read", Box::new(DeniedSyscallHandler));
        self.register_syscall("write", Box::new(DeniedSyscallHandler));
        self.register_syscall("socket", Box::new(DeniedSyscallHandler));
        self.register_syscall("connect", Box::new(DeniedSyscallHandler));
        self.register_syscall("exec", Box::new(DeniedSyscallHandler));
        self.register_syscall("fork", Box::new(DeniedSyscallHandler));
    }
    
    pub fn register_syscall(&mut self, name: &str, handler: Box<dyn SyscallHandler>) {
        self.allowed_syscalls.insert(name.to_string());
        self.syscall_handlers.insert(name.to_string(), handler);
    }
    
    pub fn is_allowed(&self, syscall: &str) -> bool {
        self.allowed_syscalls.contains(syscall)
    }
    
    pub fn handle(&mut self, syscall: &str, args: &[u32]) -> Result<u32> {
        let timestamp = std::time::Instant::now();
        
        let result = if let Some(handler) = self.syscall_handlers.get(syscall) {
            handler.handle(args)
        } else {
            log::warn!("Unknown syscall intercepted: {}", syscall);
            Err(WasmError::Runtime(format!("Unknown syscall: {}", syscall)))
        };
        
        // Log the syscall attempt
        self.syscall_log.push(SyscallEntry {
            name: syscall.to_string(),
            args: args.to_vec(),
            result: result.clone(),
            timestamp,
        });
        
        // Limit log size to prevent memory exhaustion
        if self.syscall_log.len() > 1000 {
            self.syscall_log.drain(0..500);
        }
        
        result
    }
    
    pub fn get_syscall_log(&self) -> &[SyscallEntry] {
        &self.syscall_log
    }
    
    pub fn clear_log(&mut self) {
        self.syscall_log.clear();
    }
    
    pub fn get_syscall_stats(&self) -> HashMap<String, u32> {
        let mut stats = HashMap::new();
        
        for entry in &self.syscall_log {
            *stats.entry(entry.name.clone()).or_insert(0) += 1;
        }
        
        stats
    }
    
    pub fn deny_syscall(&mut self, syscall: &str) {
        self.allowed_syscalls.remove(syscall);
        self.syscall_handlers.insert(
            syscall.to_string(), 
            Box::new(DeniedSyscallHandler)
        );
    }
    
    pub fn allow_syscall(&mut self, syscall: &str, handler: Box<dyn SyscallHandler>) {
        self.register_syscall(syscall, handler);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_syscall_trap_creation() {
        let trap = SyscallTrap::new();
        assert!(trap.is_allowed("wasm_log"));
        assert!(!trap.is_allowed("open"));
    }

    #[test]
    fn test_time_syscall() {
        let mut trap = SyscallTrap::new();
        let result = trap.handle("wasm_get_time", &[]);
        assert!(result.is_ok());
        assert!(result.unwrap() > 0);
    }

    #[test]
    fn test_denied_syscall() {
        let mut trap = SyscallTrap::new();
        let result = trap.handle("open", &[]);
        assert!(result.is_err());
    }

    #[test]
    fn test_sensor_syscall() {
        let mut trap = SyscallTrap::new();
        let result = trap.handle("wasm_read_sensor", &[0]); // Temperature sensor
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 25);
    }

    #[test]
    fn test_syscall_logging() {
        let mut trap = SyscallTrap::new();
        trap.handle("wasm_get_time", &[]).ok();
        
        let log = trap.get_syscall_log();
        assert_eq!(log.len(), 1);
        assert_eq!(log[0].name, "wasm_get_time");
    }
}