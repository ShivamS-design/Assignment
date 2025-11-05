pub mod parser;
pub mod memory;
pub mod interpreter;
pub mod vm;
pub mod error;
pub mod sandbox;
pub mod abi;

pub use error::{WasmError, Result};
pub use vm::{WasmModule, WasmInstance, WasmEngine};
pub use memory::LinearMemory;
pub use sandbox::{Sandbox, ResourceLimits};
pub use abi::WasmABI;

/// Initialize the WASM engine with logging
pub fn init() {
    env_logger::init();
    log::info!("WASM-as-OS engine initialized");
}

/// Create a new WASM engine with default settings
pub fn create_engine() -> Result<WasmEngine> {
    WasmEngine::new()
}

/// Create a sandboxed WASM instance
pub fn create_sandboxed_instance(
    module_bytes: &[u8],
    limits: ResourceLimits,
) -> Result<WasmInstance> {
    let engine = WasmEngine::new()?;
    let module = engine.parse_module(module_bytes)?;
    let sandbox = Sandbox::new(limits);
    engine.instantiate_with_sandbox(module, sandbox)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_engine_creation() {
        let engine = create_engine();
        assert!(engine.is_ok());
    }

    #[test]
    fn test_init() {
        init(); // Should not panic
    }
}