use crate::error::{WasmError, Result};
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug)]
pub struct HostInterface {
    sensor_data: HashMap<u32, u32>,
    alert_handlers: Vec<Box<dyn AlertHandler>>,
    log_handlers: Vec<Box<dyn LogHandler>>,
    initialized: bool,
}

pub trait AlertHandler: std::fmt::Debug {
    fn handle_alert(&self, level: u32, message: &str) -> Result<()>;
}

pub trait LogHandler: std::fmt::Debug {
    fn handle_log(&self, message: &str);
}

#[derive(Debug)]
struct DefaultAlertHandler;

impl AlertHandler for DefaultAlertHandler {
    fn handle_alert(&self, level: u32, message: &str) -> Result<()> {
        let level_str = match level {
            0 => "INFO",
            1 => "WARNING",
            2 => "ERROR", 
            3 => "CRITICAL",
            _ => "UNKNOWN",
        };
        
        log::warn!("[WASM ALERT {}] {}", level_str, message);
        
        // In a real system, this might send notifications, write to databases, etc.
        if level >= 3 {
            // Critical alerts might trigger immediate actions
            log::error!("CRITICAL ALERT: {}", message);
        }
        
        Ok(())
    }
}

#[derive(Debug)]
struct DefaultLogHandler;

impl LogHandler for DefaultLogHandler {
    fn handle_log(&self, message: &str) {
        log::info!("[WASM LOG] {}", message);
    }
}

impl HostInterface {
    pub fn new() -> Self {
        let mut interface = Self {
            sensor_data: HashMap::new(),
            alert_handlers: Vec::new(),
            log_handlers: Vec::new(),
            initialized: false,
        };
        
        interface.initialize();
        interface
    }
    
    fn initialize(&mut self) {
        // Initialize sensor data with default values
        self.sensor_data.insert(0, 25);    // Temperature: 25°C
        self.sensor_data.insert(1, 60);    // Humidity: 60%
        self.sensor_data.insert(2, 1013);  // Pressure: 1013 hPa
        self.sensor_data.insert(3, 0);     // Motion: no motion
        self.sensor_data.insert(4, 500);   // Light: 500 lux
        self.sensor_data.insert(5, 40);    // Sound: 40 dB
        
        // Register default handlers
        self.alert_handlers.push(Box::new(DefaultAlertHandler));
        self.log_handlers.push(Box::new(DefaultLogHandler));
        
        self.initialized = true;
    }
    
    pub fn is_initialized(&self) -> bool {
        self.initialized
    }
    
    pub fn log(&self, message: &str) {
        for handler in &self.log_handlers {
            handler.handle_log(message);
        }
    }
    
    pub fn read_sensor(&self, sensor_id: u32) -> Result<u32> {
        match self.sensor_data.get(&sensor_id) {
            Some(&value) => {
                // Simulate some variation in sensor readings
                let variation = self.get_sensor_variation(sensor_id);
                Ok(value.saturating_add(variation))
            }
            None => Err(WasmError::Runtime(format!("Unknown sensor ID: {}", sensor_id))),
        }
    }
    
    pub fn send_alert(&self, level: u32, message: &str) -> Result<()> {
        if level > 3 {
            return Err(WasmError::Runtime("Invalid alert level".to_string()));
        }
        
        for handler in &self.alert_handlers {
            handler.handle_alert(level, message)?;
        }
        
        Ok(())
    }
    
    pub fn get_time(&self) -> Result<u32> {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs() as u32)
            .map_err(|_| WasmError::Runtime("Time error".to_string()))
    }
    
    pub fn random(&self) -> Result<u32> {
        // Simple pseudo-random number generator
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        
        let mut hasher = DefaultHasher::new();
        SystemTime::now().hash(&mut hasher);
        Ok(hasher.finish() as u32)
    }
    
    pub fn update_sensor(&mut self, sensor_id: u32, value: u32) {
        self.sensor_data.insert(sensor_id, value);
    }
    
    pub fn add_alert_handler(&mut self, handler: Box<dyn AlertHandler>) {
        self.alert_handlers.push(handler);
    }
    
    pub fn add_log_handler(&mut self, handler: Box<dyn LogHandler>) {
        self.log_handlers.push(handler);
    }
    
    pub fn get_sensor_list(&self) -> Vec<(u32, &str)> {
        vec![
            (0, "Temperature"),
            (1, "Humidity"),
            (2, "Pressure"),
            (3, "Motion"),
            (4, "Light"),
            (5, "Sound"),
        ]
    }
    
    fn get_sensor_variation(&self, sensor_id: u32) -> u32 {
        // Add some realistic variation to sensor readings
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        
        let mut hasher = DefaultHasher::new();
        (sensor_id, SystemTime::now()).hash(&mut hasher);
        let random = hasher.finish() as u32;
        
        match sensor_id {
            0 => random % 5,      // Temperature: ±2.5°C
            1 => random % 10,     // Humidity: ±5%
            2 => random % 20,     // Pressure: ±10 hPa
            3 => random % 2,      // Motion: 0 or 1
            4 => random % 100,    // Light: ±50 lux
            5 => random % 20,     // Sound: ±10 dB
            _ => 0,
        }
    }
}

// Custom alert handler for file logging
#[derive(Debug)]
pub struct FileAlertHandler {
    file_path: String,
}

impl FileAlertHandler {
    pub fn new(file_path: &str) -> Self {
        Self {
            file_path: file_path.to_string(),
        }
    }
}

impl AlertHandler for FileAlertHandler {
    fn handle_alert(&self, level: u32, message: &str) -> Result<()> {
        let level_str = match level {
            0 => "INFO",
            1 => "WARNING",
            2 => "ERROR",
            3 => "CRITICAL",
            _ => "UNKNOWN",
        };
        
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|_| WasmError::Runtime("Time error".to_string()))?
            .as_secs();
        
        let log_entry = format!("[{}] {} - {}\n", timestamp, level_str, message);
        
        // In a real implementation, this would write to a file
        log::info!("Would write to {}: {}", self.file_path, log_entry.trim());
        
        Ok(())
    }
}

// Network alert handler for remote notifications
#[derive(Debug)]
pub struct NetworkAlertHandler {
    endpoint: String,
}

impl NetworkAlertHandler {
    pub fn new(endpoint: &str) -> Self {
        Self {
            endpoint: endpoint.to_string(),
        }
    }
}

impl AlertHandler for NetworkAlertHandler {
    fn handle_alert(&self, level: u32, message: &str) -> Result<()> {
        // In a real implementation, this would send HTTP requests
        log::info!("Would send alert to {}: level={}, message={}", 
                  self.endpoint, level, message);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_host_interface_creation() {
        let interface = HostInterface::new();
        assert!(interface.is_initialized());
    }

    #[test]
    fn test_sensor_reading() {
        let interface = HostInterface::new();
        
        let temp = interface.read_sensor(0);
        assert!(temp.is_ok());
        
        let invalid = interface.read_sensor(999);
        assert!(invalid.is_err());
    }

    #[test]
    fn test_alert_sending() {
        let interface = HostInterface::new();
        
        let result = interface.send_alert(1, "Test alert");
        assert!(result.is_ok());
        
        let invalid = interface.send_alert(5, "Invalid level");
        assert!(invalid.is_err());
    }

    #[test]
    fn test_time_and_random() {
        let interface = HostInterface::new();
        
        let time1 = interface.get_time().unwrap();
        let time2 = interface.get_time().unwrap();
        assert!(time2 >= time1);
        
        let rand1 = interface.random().unwrap();
        let rand2 = interface.random().unwrap();
        // Random numbers should be different (with high probability)
        assert_ne!(rand1, rand2);
    }

    #[test]
    fn test_sensor_update() {
        let mut interface = HostInterface::new();
        
        interface.update_sensor(0, 30);
        let temp = interface.read_sensor(0).unwrap();
        // Should be around 30 (with variation)
        assert!(temp >= 25 && temp <= 35);
    }
}