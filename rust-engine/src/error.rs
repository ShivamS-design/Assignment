use thiserror::Error;

pub type Result<T> = std::result::Result<T, WasmError>;

#[derive(Error, Debug, Clone)]
pub enum WasmError {
    #[error("Invalid WASM magic number")]
    InvalidMagic,
    
    #[error("Unsupported WASM version: {0}")]
    UnsupportedVersion(u32),
    
    #[error("Invalid section type: {0}")]
    InvalidSection(u8),
    
    #[error("Memory access out of bounds: address {address}, size {size}")]
    MemoryOutOfBounds { address: u32, size: u32 },
    
    #[error("Stack overflow")]
    StackOverflow,
    
    #[error("Stack underflow")]
    StackUnderflow,
    
    #[error("Invalid instruction: {0}")]
    InvalidInstruction(u8),
    
    #[error("Function not found: {0}")]
    FunctionNotFound(u32),
    
    #[error("Type mismatch")]
    TypeMismatch,
    
    #[error("Invalid module")]
    InvalidModule,
    
    #[error("Runtime error: {0}")]
    Runtime(String),
    
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
}