use crate::error::{WasmError, Result};

/// WASM ABI function signatures and documentation
/// 
/// This module defines the interface between WASM modules and the host system.
/// All functions are designed to be safe and sandboxed.

#[derive(Debug, Clone)]
pub struct FunctionSignature {
    pub name: String,
    pub params: Vec<ValueType>,
    pub results: Vec<ValueType>,
    pub description: String,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ValueType {
    I32,
    I64,
    F32,
    F64,
}

impl ValueType {
    pub fn size(&self) -> u32 {
        match self {
            ValueType::I32 | ValueType::F32 => 4,
            ValueType::I64 | ValueType::F64 => 8,
        }
    }
}

/// Get all available ABI functions
pub fn get_abi_functions() -> Vec<FunctionSignature> {
    vec![
        FunctionSignature {
            name: "wasm_log".to_string(),
            params: vec![ValueType::I32, ValueType::I32], // message_ptr, message_len
            results: vec![ValueType::I32], // status code
            description: "Log a message to the host system. Args: (message_ptr, message_len)".to_string(),
        },
        FunctionSignature {
            name: "wasm_read_sensor".to_string(),
            params: vec![ValueType::I32], // sensor_id
            results: vec![ValueType::I32], // sensor_value
            description: "Read a sensor value. Args: (sensor_id). Sensor IDs: 0=temp, 1=humidity, 2=pressure, 3=motion, 4=light, 5=sound".to_string(),
        },
        FunctionSignature {
            name: "wasm_send_alert".to_string(),
            params: vec![ValueType::I32, ValueType::I32, ValueType::I32], // level, message_ptr, message_len
            results: vec![ValueType::I32], // status code
            description: "Send an alert. Args: (level, message_ptr, message_len). Levels: 0=info, 1=warning, 2=error, 3=critical".to_string(),
        },
        FunctionSignature {
            name: "wasm_get_time".to_string(),
            params: vec![], // no parameters
            results: vec![ValueType::I32], // timestamp
            description: "Get current Unix timestamp in seconds".to_string(),
        },
        FunctionSignature {
            name: "wasm_random".to_string(),
            params: vec![], // no parameters
            results: vec![ValueType::I32], // random_value
            description: "Get a pseudo-random 32-bit integer".to_string(),
        },
        FunctionSignature {
            name: "wasm_memory_size".to_string(),
            params: vec![], // no parameters
            results: vec![ValueType::I32], // page_count
            description: "Get current memory size in pages (64KB each)".to_string(),
        },
        FunctionSignature {
            name: "wasm_memory_grow".to_string(),
            params: vec![ValueType::I32], // delta_pages
            results: vec![ValueType::I32], // previous_size or -1 on failure
            description: "Grow memory by delta pages. Returns previous size or -1 on failure".to_string(),
        },
    ]
}

/// Validate function call arguments
pub fn validate_function_call(name: &str, args: &[ValueType]) -> Result<()> {
    let functions = get_abi_functions();
    
    let function = functions.iter()
        .find(|f| f.name == name)
        .ok_or_else(|| WasmError::Runtime(format!("Unknown function: {}", name)))?;
    
    if args.len() != function.params.len() {
        return Err(WasmError::Runtime(format!(
            "Function {} expects {} arguments, got {}",
            name, function.params.len(), args.len()
        )));
    }
    
    for (i, (expected, actual)) in function.params.iter().zip(args.iter()).enumerate() {
        if expected != actual {
            return Err(WasmError::Runtime(format!(
                "Function {} argument {} type mismatch: expected {:?}, got {:?}",
                name, i, expected, actual
            )));
        }
    }
    
    Ok(())
}

/// Get function documentation
pub fn get_function_docs(name: &str) -> Option<String> {
    get_abi_functions().iter()
        .find(|f| f.name == name)
        .map(|f| f.description.clone())
}

/// Generate C header for the ABI
pub fn generate_c_header() -> String {
    let mut header = String::new();
    header.push_str("/* WASM-as-OS ABI Functions */\n");
    header.push_str("/* This header defines the interface between WASM modules and the host */\n\n");
    header.push_str("#ifndef WASM_ABI_H\n");
    header.push_str("#define WASM_ABI_H\n\n");
    header.push_str("#include <stdint.h>\n\n");
    
    // Function declarations
    for func in get_abi_functions() {
        header.push_str(&format!("/* {} */\n", func.description));
        
        let return_type = match func.results.first() {
            Some(ValueType::I32) => "int32_t",
            Some(ValueType::I64) => "int64_t", 
            Some(ValueType::F32) => "float",
            Some(ValueType::F64) => "double",
            None => "void",
        };
        
        let params: Vec<String> = func.params.iter().enumerate().map(|(i, param_type)| {
            let type_str = match param_type {
                ValueType::I32 => "int32_t",
                ValueType::I64 => "int64_t",
                ValueType::F32 => "float", 
                ValueType::F64 => "double",
            };
            format!("{} arg{}", type_str, i)
        }).collect();
        
        let params_str = if params.is_empty() {
            "void".to_string()
        } else {
            params.join(", ")
        };
        
        header.push_str(&format!("extern {} {}({});\n\n", return_type, func.name, params_str));
    }
    
    // Constants
    header.push_str("/* Sensor IDs */\n");
    header.push_str("#define SENSOR_TEMPERATURE 0\n");
    header.push_str("#define SENSOR_HUMIDITY    1\n");
    header.push_str("#define SENSOR_PRESSURE    2\n");
    header.push_str("#define SENSOR_MOTION      3\n");
    header.push_str("#define SENSOR_LIGHT       4\n");
    header.push_str("#define SENSOR_SOUND       5\n\n");
    
    header.push_str("/* Alert Levels */\n");
    header.push_str("#define ALERT_INFO     0\n");
    header.push_str("#define ALERT_WARNING  1\n");
    header.push_str("#define ALERT_ERROR    2\n");
    header.push_str("#define ALERT_CRITICAL 3\n\n");
    
    header.push_str("#endif /* WASM_ABI_H */\n");
    
    header
}

/// Generate Rust bindings for the ABI
pub fn generate_rust_bindings() -> String {
    let mut bindings = String::new();
    bindings.push_str("//! WASM-as-OS ABI Bindings for Rust\n");
    bindings.push_str("//! This module provides safe Rust bindings for WASM modules\n\n");
    
    // External function declarations
    bindings.push_str("extern \"C\" {\n");
    for func in get_abi_functions() {
        bindings.push_str(&format!("    /// {}\n", func.description));
        
        let return_type = match func.results.first() {
            Some(ValueType::I32) => "i32",
            Some(ValueType::I64) => "i64",
            Some(ValueType::F32) => "f32", 
            Some(ValueType::F64) => "f64",
            None => "()",
        };
        
        let params: Vec<String> = func.params.iter().enumerate().map(|(i, param_type)| {
            let type_str = match param_type {
                ValueType::I32 => "i32",
                ValueType::I64 => "i64",
                ValueType::F32 => "f32",
                ValueType::F64 => "f64",
            };
            format!("arg{}: {}", i, type_str)
        }).collect();
        
        let params_str = params.join(", ");
        bindings.push_str(&format!("    pub fn {}({}) -> {};\n", func.name, params_str, return_type));
    }
    bindings.push_str("}\n\n");
    
    // Safe wrapper functions
    bindings.push_str("/// Safe wrapper functions\n");
    bindings.push_str("pub mod safe {\n");
    bindings.push_str("    use super::*;\n\n");
    
    bindings.push_str("    pub fn log(message: &str) -> i32 {\n");
    bindings.push_str("        unsafe {\n");
    bindings.push_str("            wasm_log(message.as_ptr() as i32, message.len() as i32)\n");
    bindings.push_str("        }\n");
    bindings.push_str("    }\n\n");
    
    bindings.push_str("    pub fn read_sensor(sensor_id: u32) -> i32 {\n");
    bindings.push_str("        unsafe { wasm_read_sensor(sensor_id as i32) }\n");
    bindings.push_str("    }\n\n");
    
    bindings.push_str("    pub fn send_alert(level: u32, message: &str) -> i32 {\n");
    bindings.push_str("        unsafe {\n");
    bindings.push_str("            wasm_send_alert(\n");
    bindings.push_str("                level as i32,\n");
    bindings.push_str("                message.as_ptr() as i32,\n");
    bindings.push_str("                message.len() as i32\n");
    bindings.push_str("            )\n");
    bindings.push_str("        }\n");
    bindings.push_str("    }\n\n");
    
    bindings.push_str("    pub fn get_time() -> i32 {\n");
    bindings.push_str("        unsafe { wasm_get_time() }\n");
    bindings.push_str("    }\n\n");
    
    bindings.push_str("    pub fn random() -> i32 {\n");
    bindings.push_str("        unsafe { wasm_random() }\n");
    bindings.push_str("    }\n");
    
    bindings.push_str("}\n\n");
    
    // Constants
    bindings.push_str("pub mod sensors {\n");
    bindings.push_str("    pub const TEMPERATURE: u32 = 0;\n");
    bindings.push_str("    pub const HUMIDITY: u32 = 1;\n");
    bindings.push_str("    pub const PRESSURE: u32 = 2;\n");
    bindings.push_str("    pub const MOTION: u32 = 3;\n");
    bindings.push_str("    pub const LIGHT: u32 = 4;\n");
    bindings.push_str("    pub const SOUND: u32 = 5;\n");
    bindings.push_str("}\n\n");
    
    bindings.push_str("pub mod alerts {\n");
    bindings.push_str("    pub const INFO: u32 = 0;\n");
    bindings.push_str("    pub const WARNING: u32 = 1;\n");
    bindings.push_str("    pub const ERROR: u32 = 2;\n");
    bindings.push_str("    pub const CRITICAL: u32 = 3;\n");
    bindings.push_str("}\n");
    
    bindings
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_function_validation() {
        let result = validate_function_call("wasm_log", &[ValueType::I32, ValueType::I32]);
        assert!(result.is_ok());
        
        let result = validate_function_call("wasm_log", &[ValueType::I32]);
        assert!(result.is_err());
        
        let result = validate_function_call("unknown_function", &[]);
        assert!(result.is_err());
    }

    #[test]
    fn test_function_docs() {
        let docs = get_function_docs("wasm_log");
        assert!(docs.is_some());
        assert!(docs.unwrap().contains("Log a message"));
        
        let docs = get_function_docs("unknown_function");
        assert!(docs.is_none());
    }

    #[test]
    fn test_header_generation() {
        let header = generate_c_header();
        assert!(header.contains("#ifndef WASM_ABI_H"));
        assert!(header.contains("wasm_log"));
        assert!(header.contains("SENSOR_TEMPERATURE"));
    }

    #[test]
    fn test_rust_bindings_generation() {
        let bindings = generate_rust_bindings();
        assert!(bindings.contains("extern \"C\""));
        assert!(bindings.contains("pub fn wasm_log"));
        assert!(bindings.contains("pub mod safe"));
    }
}