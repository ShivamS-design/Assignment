use crate::error::{WasmError, Result};
use std::collections::HashMap;

const PAGE_SIZE: u32 = 65536; // 64KB
const MAX_PAGES: u32 = 65536; // 4GB max

#[derive(Debug)]
pub struct LinearMemory {
    data: Vec<u8>,
    min_pages: u32,
    max_pages: Option<u32>,
    current_pages: u32,
}

impl LinearMemory {
    pub fn new(min_pages: u32, max_pages: Option<u32>) -> Result<Self> {
        if let Some(max) = max_pages {
            if min_pages > max || max > MAX_PAGES {
                return Err(WasmError::InvalidModule);
            }
        }
        
        let initial_size = (min_pages * PAGE_SIZE) as usize;
        let mut data = Vec::with_capacity(initial_size);
        data.resize(initial_size, 0);
        
        Ok(LinearMemory {
            data,
            min_pages,
            max_pages,
            current_pages: min_pages,
        })
    }
    
    pub fn size(&self) -> u32 {
        self.current_pages
    }
    
    pub fn grow(&mut self, delta: u32) -> Result<u32> {
        let old_size = self.current_pages;
        let new_size = old_size + delta;
        
        if let Some(max) = self.max_pages {
            if new_size > max {
                return Err(WasmError::Runtime("Memory grow failed".to_string()));
            }
        }
        
        if new_size > MAX_PAGES {
            return Err(WasmError::Runtime("Memory grow failed".to_string()));
        }
        
        let new_byte_size = (new_size * PAGE_SIZE) as usize;
        self.data.resize(new_byte_size, 0);
        self.current_pages = new_size;
        
        Ok(old_size)
    }
    
    pub fn read_u8(&self, address: u32) -> Result<u8> {
        self.check_bounds(address, 1)?;
        Ok(self.data[address as usize])
    }
    
    pub fn read_u16(&self, address: u32) -> Result<u16> {
        self.check_bounds(address, 2)?;
        let bytes = &self.data[address as usize..address as usize + 2];
        Ok(u16::from_le_bytes([bytes[0], bytes[1]]))
    }
    
    pub fn read_u32(&self, address: u32) -> Result<u32> {
        self.check_bounds(address, 4)?;
        let bytes = &self.data[address as usize..address as usize + 4];
        Ok(u32::from_le_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]))
    }
    
    pub fn read_u64(&self, address: u32) -> Result<u64> {
        self.check_bounds(address, 8)?;
        let bytes = &self.data[address as usize..address as usize + 8];
        Ok(u64::from_le_bytes([
            bytes[0], bytes[1], bytes[2], bytes[3],
            bytes[4], bytes[5], bytes[6], bytes[7],
        ]))
    }
    
    pub fn read_f32(&self, address: u32) -> Result<f32> {
        Ok(f32::from_bits(self.read_u32(address)?))
    }
    
    pub fn read_f64(&self, address: u32) -> Result<f64> {
        Ok(f64::from_bits(self.read_u64(address)?))
    }
    
    pub fn write_u8(&mut self, address: u32, value: u8) -> Result<()> {
        self.check_bounds(address, 1)?;
        self.data[address as usize] = value;
        Ok(())
    }
    
    pub fn write_u16(&mut self, address: u32, value: u16) -> Result<()> {
        self.check_bounds(address, 2)?;
        let bytes = value.to_le_bytes();
        self.data[address as usize..address as usize + 2].copy_from_slice(&bytes);
        Ok(())
    }
    
    pub fn write_u32(&mut self, address: u32, value: u32) -> Result<()> {
        self.check_bounds(address, 4)?;
        let bytes = value.to_le_bytes();
        self.data[address as usize..address as usize + 4].copy_from_slice(&bytes);
        Ok(())
    }
    
    pub fn write_u64(&mut self, address: u32, value: u64) -> Result<()> {
        self.check_bounds(address, 8)?;
        let bytes = value.to_le_bytes();
        self.data[address as usize..address as usize + 8].copy_from_slice(&bytes);
        Ok(())
    }
    
    pub fn write_f32(&mut self, address: u32, value: f32) -> Result<()> {
        self.write_u32(address, value.to_bits())
    }
    
    pub fn write_f64(&mut self, address: u32, value: f64) -> Result<()> {
        self.write_u64(address, value.to_bits())
    }
    
    pub fn read_bytes(&self, address: u32, len: u32) -> Result<&[u8]> {
        self.check_bounds(address, len)?;
        Ok(&self.data[address as usize..(address + len) as usize])
    }
    
    pub fn write_bytes(&mut self, address: u32, data: &[u8]) -> Result<()> {
        self.check_bounds(address, data.len() as u32)?;
        let start = address as usize;
        let end = start + data.len();
        self.data[start..end].copy_from_slice(data);
        Ok(())
    }
    
    fn check_bounds(&self, address: u32, size: u32) -> Result<()> {
        let end_address = address.checked_add(size)
            .ok_or(WasmError::MemoryOutOfBounds { address, size })?;
        
        if end_address > self.data.len() as u32 {
            return Err(WasmError::MemoryOutOfBounds { address, size });
        }
        
        Ok(())
    }
}

#[derive(Debug)]
pub struct MemoryManager {
    memories: HashMap<u32, LinearMemory>,
    next_id: u32,
}

impl MemoryManager {
    pub fn new() -> Self {
        MemoryManager {
            memories: HashMap::new(),
            next_id: 0,
        }
    }
    
    pub fn create_memory(&mut self, min_pages: u32, max_pages: Option<u32>) -> Result<u32> {
        let memory = LinearMemory::new(min_pages, max_pages)?;
        let id = self.next_id;
        self.memories.insert(id, memory);
        self.next_id += 1;
        Ok(id)
    }
    
    pub fn get_memory(&self, id: u32) -> Option<&LinearMemory> {
        self.memories.get(&id)
    }
    
    pub fn get_memory_mut(&mut self, id: u32) -> Option<&mut LinearMemory> {
        self.memories.get_mut(&id)
    }
    
    pub fn destroy_memory(&mut self, id: u32) -> bool {
        self.memories.remove(&id).is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_memory_creation() {
        let memory = LinearMemory::new(1, Some(10));
        assert!(memory.is_ok());
        let memory = memory.unwrap();
        assert_eq!(memory.size(), 1);
    }

    #[test]
    fn test_memory_read_write() {
        let mut memory = LinearMemory::new(1, None).unwrap();
        
        memory.write_u32(0, 0x12345678).unwrap();
        assert_eq!(memory.read_u32(0).unwrap(), 0x12345678);
    }

    #[test]
    fn test_memory_bounds_check() {
        let memory = LinearMemory::new(1, None).unwrap();
        let result = memory.read_u32(PAGE_SIZE);
        assert!(matches!(result, Err(WasmError::MemoryOutOfBounds { .. })));
    }

    #[test]
    fn test_memory_grow() {
        let mut memory = LinearMemory::new(1, Some(3)).unwrap();
        let old_size = memory.grow(1).unwrap();
        assert_eq!(old_size, 1);
        assert_eq!(memory.size(), 2);
    }

    #[test]
    fn test_memory_manager() {
        let mut manager = MemoryManager::new();
        let id = manager.create_memory(1, None).unwrap();
        assert!(manager.get_memory(id).is_some());
        assert!(manager.destroy_memory(id));
        assert!(manager.get_memory(id).is_none());
    }
}