use std::time::{Duration, Instant};

#[derive(Debug, Clone)]
pub struct ResourceLimits {
    pub max_memory_pages: u32,
    pub max_cpu_time: Duration,
    pub max_syscalls: u32,
    pub max_instructions: u64,
    pub max_stack_depth: u32,
    pub max_globals: u32,
    pub max_table_size: u32,
}

impl Default for ResourceLimits {
    fn default() -> Self {
        Self {
            max_memory_pages: 256,        // 16MB
            max_cpu_time: Duration::from_secs(30),
            max_syscalls: 1000,
            max_instructions: 1_000_000,
            max_stack_depth: 1024,
            max_globals: 100,
            max_table_size: 1000,
        }
    }
}

impl ResourceLimits {
    pub fn strict() -> Self {
        Self {
            max_memory_pages: 64,         // 4MB
            max_cpu_time: Duration::from_secs(5),
            max_syscalls: 100,
            max_instructions: 100_000,
            max_stack_depth: 256,
            max_globals: 10,
            max_table_size: 100,
        }
    }
    
    pub fn permissive() -> Self {
        Self {
            max_memory_pages: 1024,       // 64MB
            max_cpu_time: Duration::from_secs(300),
            max_syscalls: 10000,
            max_instructions: 10_000_000,
            max_stack_depth: 4096,
            max_globals: 1000,
            max_table_size: 10000,
        }
    }
    
    pub fn with_memory_limit(mut self, pages: u32) -> Self {
        self.max_memory_pages = pages;
        self
    }
    
    pub fn with_cpu_time_limit(mut self, duration: Duration) -> Self {
        self.max_cpu_time = duration;
        self
    }
    
    pub fn with_syscall_limit(mut self, count: u32) -> Self {
        self.max_syscalls = count;
        self
    }
    
    pub fn with_instruction_limit(mut self, count: u64) -> Self {
        self.max_instructions = count;
        self
    }
}

#[derive(Debug)]
pub struct ResourceMonitor {
    limits: ResourceLimits,
    start_time: Instant,
    last_check: Instant,
    check_interval: Duration,
}

impl ResourceMonitor {
    pub fn new(limits: ResourceLimits) -> Self {
        let now = Instant::now();
        Self {
            limits,
            start_time: now,
            last_check: now,
            check_interval: Duration::from_millis(100), // Check every 100ms
        }
    }
    
    pub fn should_check(&mut self) -> bool {
        let now = Instant::now();
        if now.duration_since(self.last_check) >= self.check_interval {
            self.last_check = now;
            true
        } else {
            false
        }
    }
    
    pub fn elapsed_time(&self) -> Duration {
        Instant::now().duration_since(self.start_time)
    }
    
    pub fn is_cpu_time_exceeded(&self) -> bool {
        self.elapsed_time() > self.limits.max_cpu_time
    }
    
    pub fn remaining_cpu_time(&self) -> Duration {
        self.limits.max_cpu_time.saturating_sub(self.elapsed_time())
    }
    
    pub fn get_limits(&self) -> &ResourceLimits {
        &self.limits
    }
    
    pub fn reset(&mut self) {
        let now = Instant::now();
        self.start_time = now;
        self.last_check = now;
    }
}

#[derive(Debug)]
pub struct MemoryLimiter {
    max_pages: u32,
    current_pages: u32,
    peak_pages: u32,
    allocations: u32,
}

impl MemoryLimiter {
    pub fn new(max_pages: u32) -> Self {
        Self {
            max_pages,
            current_pages: 0,
            peak_pages: 0,
            allocations: 0,
        }
    }
    
    pub fn can_allocate(&self, pages: u32) -> bool {
        self.current_pages + pages <= self.max_pages
    }
    
    pub fn allocate(&mut self, pages: u32) -> Result<(), String> {
        if !self.can_allocate(pages) {
            return Err(format!(
                "Memory allocation would exceed limit: {} + {} > {}",
                self.current_pages, pages, self.max_pages
            ));
        }
        
        self.current_pages += pages;
        self.peak_pages = self.peak_pages.max(self.current_pages);
        self.allocations += 1;
        Ok(())
    }
    
    pub fn deallocate(&mut self, pages: u32) {
        self.current_pages = self.current_pages.saturating_sub(pages);
    }
    
    pub fn current_usage(&self) -> u32 {
        self.current_pages
    }
    
    pub fn peak_usage(&self) -> u32 {
        self.peak_pages
    }
    
    pub fn utilization(&self) -> f64 {
        if self.max_pages == 0 {
            0.0
        } else {
            self.current_pages as f64 / self.max_pages as f64
        }
    }
    
    pub fn reset(&mut self) {
        self.current_pages = 0;
        self.peak_pages = 0;
        self.allocations = 0;
    }
}

#[derive(Debug)]
pub struct InstructionCounter {
    max_instructions: u64,
    current_count: u64,
    last_reset: Instant,
}

impl InstructionCounter {
    pub fn new(max_instructions: u64) -> Self {
        Self {
            max_instructions,
            current_count: 0,
            last_reset: Instant::now(),
        }
    }
    
    pub fn increment(&mut self, count: u64) -> Result<(), String> {
        self.current_count += count;
        
        if self.current_count > self.max_instructions {
            return Err(format!(
                "Instruction limit exceeded: {} > {}",
                self.current_count, self.max_instructions
            ));
        }
        
        Ok(())
    }
    
    pub fn remaining(&self) -> u64 {
        self.max_instructions.saturating_sub(self.current_count)
    }
    
    pub fn current_count(&self) -> u64 {
        self.current_count
    }
    
    pub fn utilization(&self) -> f64 {
        if self.max_instructions == 0 {
            0.0
        } else {
            self.current_count as f64 / self.max_instructions as f64
        }
    }
    
    pub fn reset(&mut self) {
        self.current_count = 0;
        self.last_reset = Instant::now();
    }
    
    pub fn time_since_reset(&self) -> Duration {
        Instant::now().duration_since(self.last_reset)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_resource_limits_creation() {
        let limits = ResourceLimits::default();
        assert_eq!(limits.max_memory_pages, 256);
        
        let strict = ResourceLimits::strict();
        assert_eq!(strict.max_memory_pages, 64);
    }

    #[test]
    fn test_memory_limiter() {
        let mut limiter = MemoryLimiter::new(10);
        
        assert!(limiter.allocate(5).is_ok());
        assert_eq!(limiter.current_usage(), 5);
        
        assert!(limiter.allocate(6).is_err()); // Would exceed limit
        assert!(limiter.allocate(5).is_ok()); // Exactly at limit
        
        limiter.deallocate(3);
        assert_eq!(limiter.current_usage(), 7);
    }

    #[test]
    fn test_instruction_counter() {
        let mut counter = InstructionCounter::new(100);
        
        assert!(counter.increment(50).is_ok());
        assert_eq!(counter.remaining(), 50);
        
        assert!(counter.increment(51).is_err()); // Would exceed limit
        assert!(counter.increment(50).is_ok()); // Exactly at limit
    }

    #[test]
    fn test_resource_monitor() {
        let limits = ResourceLimits::default();
        let monitor = ResourceMonitor::new(limits);
        
        assert!(!monitor.is_cpu_time_exceeded());
        assert!(monitor.remaining_cpu_time() > Duration::ZERO);
    }
}