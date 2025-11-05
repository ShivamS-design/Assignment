use crate::error::{WasmError, Result};
use byteorder::{LittleEndian, ReadBytesExt};
use std::io::{Cursor, Read};

const WASM_MAGIC: u32 = 0x6d736100; // "\0asm"
const WASM_VERSION: u32 = 0x01;

#[derive(Debug, Clone)]
pub struct WasmModule {
    pub types: Vec<FunctionType>,
    pub functions: Vec<Function>,
    pub memory: Option<MemoryType>,
    pub exports: Vec<Export>,
    pub code: Vec<CodeSection>,
}

#[derive(Debug, Clone)]
pub struct FunctionType {
    pub params: Vec<ValueType>,
    pub results: Vec<ValueType>,
}

#[derive(Debug, Clone)]
pub struct Function {
    pub type_idx: u32,
}

#[derive(Debug, Clone)]
pub struct MemoryType {
    pub min: u32,
    pub max: Option<u32>,
}

#[derive(Debug, Clone)]
pub struct Export {
    pub name: String,
    pub kind: ExportKind,
    pub index: u32,
}

#[derive(Debug, Clone)]
pub enum ExportKind {
    Function,
    Memory,
    Global,
    Table,
}

#[derive(Debug, Clone)]
pub struct CodeSection {
    pub locals: Vec<LocalEntry>,
    pub body: Vec<u8>,
}

#[derive(Debug, Clone)]
pub struct LocalEntry {
    pub count: u32,
    pub value_type: ValueType,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ValueType {
    I32,
    I64,
    F32,
    F64,
}

pub struct WasmParser;

impl WasmParser {
    pub fn parse(bytes: &[u8]) -> Result<WasmModule> {
        let mut cursor = Cursor::new(bytes);
        
        // Validate magic number and version
        let magic = cursor.read_u32::<LittleEndian>()?;
        if magic != WASM_MAGIC {
            return Err(WasmError::InvalidMagic);
        }
        
        let version = cursor.read_u32::<LittleEndian>()?;
        if version != WASM_VERSION {
            return Err(WasmError::UnsupportedVersion(version));
        }
        
        let mut module = WasmModule {
            types: Vec::new(),
            functions: Vec::new(),
            memory: None,
            exports: Vec::new(),
            code: Vec::new(),
        };
        
        // Parse sections
        while cursor.position() < bytes.len() as u64 {
            let section_id = cursor.read_u8()?;
            let section_size = Self::read_leb128_u32(&mut cursor)?;
            
            match section_id {
                1 => module.types = Self::parse_type_section(&mut cursor, section_size)?,
                3 => module.functions = Self::parse_function_section(&mut cursor, section_size)?,
                5 => module.memory = Self::parse_memory_section(&mut cursor, section_size)?,
                7 => module.exports = Self::parse_export_section(&mut cursor, section_size)?,
                10 => module.code = Self::parse_code_section(&mut cursor, section_size)?,
                _ => {
                    // Skip unknown sections
                    cursor.set_position(cursor.position() + section_size as u64);
                }
            }
        }
        
        Ok(module)
    }
    
    fn parse_type_section(cursor: &mut Cursor<&[u8]>, _size: u32) -> Result<Vec<FunctionType>> {
        let count = Self::read_leb128_u32(cursor)?;
        let mut types = Vec::with_capacity(count as usize);
        
        for _ in 0..count {
            let form = cursor.read_u8()?;
            if form != 0x60 {
                return Err(WasmError::InvalidModule);
            }
            
            let param_count = Self::read_leb128_u32(cursor)?;
            let mut params = Vec::with_capacity(param_count as usize);
            for _ in 0..param_count {
                params.push(Self::read_value_type(cursor)?);
            }
            
            let result_count = Self::read_leb128_u32(cursor)?;
            let mut results = Vec::with_capacity(result_count as usize);
            for _ in 0..result_count {
                results.push(Self::read_value_type(cursor)?);
            }
            
            types.push(FunctionType { params, results });
        }
        
        Ok(types)
    }
    
    fn parse_function_section(cursor: &mut Cursor<&[u8]>, _size: u32) -> Result<Vec<Function>> {
        let count = Self::read_leb128_u32(cursor)?;
        let mut functions = Vec::with_capacity(count as usize);
        
        for _ in 0..count {
            let type_idx = Self::read_leb128_u32(cursor)?;
            functions.push(Function { type_idx });
        }
        
        Ok(functions)
    }
    
    fn parse_memory_section(cursor: &mut Cursor<&[u8]>, _size: u32) -> Result<Option<MemoryType>> {
        let count = Self::read_leb128_u32(cursor)?;
        if count == 0 {
            return Ok(None);
        }
        
        let flags = cursor.read_u8()?;
        let min = Self::read_leb128_u32(cursor)?;
        let max = if flags & 0x01 != 0 {
            Some(Self::read_leb128_u32(cursor)?)
        } else {
            None
        };
        
        Ok(Some(MemoryType { min, max }))
    }
    
    fn parse_export_section(cursor: &mut Cursor<&[u8]>, _size: u32) -> Result<Vec<Export>> {
        let count = Self::read_leb128_u32(cursor)?;
        let mut exports = Vec::with_capacity(count as usize);
        
        for _ in 0..count {
            let name_len = Self::read_leb128_u32(cursor)?;
            let mut name_bytes = vec![0u8; name_len as usize];
            cursor.read_exact(&mut name_bytes)?;
            let name = String::from_utf8_lossy(&name_bytes).to_string();
            
            let kind = match cursor.read_u8()? {
                0 => ExportKind::Function,
                1 => ExportKind::Table,
                2 => ExportKind::Memory,
                3 => ExportKind::Global,
                _ => return Err(WasmError::InvalidModule),
            };
            
            let index = Self::read_leb128_u32(cursor)?;
            exports.push(Export { name, kind, index });
        }
        
        Ok(exports)
    }
    
    fn parse_code_section(cursor: &mut Cursor<&[u8]>, _size: u32) -> Result<Vec<CodeSection>> {
        let count = Self::read_leb128_u32(cursor)?;
        let mut code_sections = Vec::with_capacity(count as usize);
        
        for _ in 0..count {
            let body_size = Self::read_leb128_u32(cursor)?;
            let local_count = Self::read_leb128_u32(cursor)?;
            
            let mut locals = Vec::with_capacity(local_count as usize);
            for _ in 0..local_count {
                let count = Self::read_leb128_u32(cursor)?;
                let value_type = Self::read_value_type(cursor)?;
                locals.push(LocalEntry { count, value_type });
            }
            
            let body_len = body_size - (cursor.position() as u32 - (body_size - body_size));
            let mut body = vec![0u8; body_len as usize];
            cursor.read_exact(&mut body)?;
            
            code_sections.push(CodeSection { locals, body });
        }
        
        Ok(code_sections)
    }
    
    fn read_value_type(cursor: &mut Cursor<&[u8]>) -> Result<ValueType> {
        match cursor.read_u8()? {
            0x7F => Ok(ValueType::I32),
            0x7E => Ok(ValueType::I64),
            0x7D => Ok(ValueType::F32),
            0x7C => Ok(ValueType::F64),
            _ => Err(WasmError::InvalidModule),
        }
    }
    
    fn read_leb128_u32(cursor: &mut Cursor<&[u8]>) -> Result<u32> {
        let mut result = 0u32;
        let mut shift = 0;
        
        loop {
            let byte = cursor.read_u8()?;
            result |= ((byte & 0x7F) as u32) << shift;
            
            if byte & 0x80 == 0 {
                break;
            }
            
            shift += 7;
            if shift >= 32 {
                return Err(WasmError::InvalidModule);
            }
        }
        
        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_empty_module() {
        let bytes = [
            0x00, 0x61, 0x73, 0x6d, // magic
            0x01, 0x00, 0x00, 0x00, // version
        ];
        
        let result = WasmParser::parse(&bytes);
        assert!(result.is_ok());
    }

    #[test]
    fn test_invalid_magic() {
        let bytes = [0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00];
        let result = WasmParser::parse(&bytes);
        assert!(matches!(result, Err(WasmError::InvalidMagic)));
    }
}