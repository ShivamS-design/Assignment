use super::Breakpoint;
use std::collections::HashMap;

#[derive(Debug)]
pub struct BreakpointManager {
    breakpoints: HashMap<u32, Breakpoint>,
    next_id: u32,
    address_map: HashMap<u32, u32>, // address -> breakpoint_id
}

impl BreakpointManager {
    pub fn new() -> Self {
        Self {
            breakpoints: HashMap::new(),
            next_id: 1,
            address_map: HashMap::new(),
        }
    }

    pub fn set(&mut self, function_index: u32, instruction_offset: u32) -> u32 {
        let id = self.next_id;
        self.next_id += 1;

        let address = self.calculate_address(function_index, instruction_offset);
        
        let breakpoint = Breakpoint {
            id,
            function_index,
            instruction_offset,
            enabled: true,
            hit_count: 0,
        };

        self.breakpoints.insert(id, breakpoint);
        self.address_map.insert(address, id);
        
        id
    }

    pub fn clear(&mut self, id: u32) -> bool {
        if let Some(bp) = self.breakpoints.remove(&id) {
            let address = self.calculate_address(bp.function_index, bp.instruction_offset);
            self.address_map.remove(&address);
            true
        } else {
            false
        }
    }

    pub fn enable(&mut self, id: u32) -> bool {
        if let Some(bp) = self.breakpoints.get_mut(&id) {
            bp.enabled = true;
            true
        } else {
            false
        }
    }

    pub fn disable(&mut self, id: u32) -> bool {
        if let Some(bp) = self.breakpoints.get_mut(&id) {
            bp.enabled = false;
            true
        } else {
            false
        }
    }

    pub fn list(&self) -> Vec<&Breakpoint> {
        self.breakpoints.values().collect()
    }

    pub fn should_break(&mut self, address: u32) -> bool {
        if let Some(&bp_id) = self.address_map.get(&address) {
            if let Some(bp) = self.breakpoints.get_mut(&bp_id) {
                if bp.enabled {
                    bp.hit_count += 1;
                    return true;
                }
            }
        }
        false
    }

    pub fn get_hit_count(&self, id: u32) -> Option<u32> {
        self.breakpoints.get(&id).map(|bp| bp.hit_count)
    }

    pub fn clear_all(&mut self) {
        self.breakpoints.clear();
        self.address_map.clear();
    }

    fn calculate_address(&self, function_index: u32, instruction_offset: u32) -> u32 {
        // Simple address calculation - in real implementation would use function table
        (function_index << 16) | instruction_offset
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_breakpoint_management() {
        let mut manager = BreakpointManager::new();
        
        let id1 = manager.set(0, 10);
        let id2 = manager.set(1, 20);
        
        assert_eq!(manager.list().len(), 2);
        
        assert!(manager.should_break(10)); // function 0, offset 10
        assert!(!manager.should_break(15)); // no breakpoint
        
        assert!(manager.clear(id1));
        assert_eq!(manager.list().len(), 1);
        
        assert!(!manager.should_break(10)); // breakpoint cleared
    }

    #[test]
    fn test_breakpoint_enable_disable() {
        let mut manager = BreakpointManager::new();
        let id = manager.set(0, 10);
        
        assert!(manager.should_break(10));
        
        manager.disable(id);
        assert!(!manager.should_break(10));
        
        manager.enable(id);
        assert!(manager.should_break(10));
    }
}