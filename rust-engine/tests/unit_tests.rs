use wasm_engine::*;
use std::collections::HashMap;

#[cfg(test)]
mod parser_tests {
    use super::*;
    use wasm_engine::parser::*;

    #[test]
    fn test_wasm_header_parsing() {
        let valid_wasm = vec![0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00];
        let parser = WasmParser::new();
        
        let result = parser.parse_header(&valid_wasm);
        assert!(result.is_ok());
        
        let header = result.unwrap();
        assert_eq!(header.magic, 0x6d736100);
        assert_eq!(header.version, 1);
    }

    #[test]
    fn test_invalid_magic_number() {
        let invalid_wasm = vec![0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00];
        let parser = WasmParser::new();
        
        let result = parser.parse_header(&invalid_wasm);
        assert!(result.is_err());
    }

    #[test]
    fn test_section_parsing() {
        let parser = WasmParser::new();
        let section_data = vec![0x01, 0x05, 0x01, 0x60, 0x00, 0x00]; // Type section
        
        let result = parser.parse_section(&section_data);
        assert!(result.is_ok());
        
        let section = result.unwrap();
        assert_eq!(section.section_type, SectionType::Type);
    }
}

#[cfg(test)]
mod memory_tests {
    use super::*;
    use wasm_engine::memory::*;

    #[test]
    fn test_linear_memory_allocation() {
        let mut memory = LinearMemory::new(1); // 1 page = 64KB
        
        assert_eq!(memory.size(), 65536);
        
        let result = memory.grow(1);
        assert!(result.is_ok());
        assert_eq!(memory.size(), 131072);
    }

    #[test]
    fn test_memory_bounds_checking() {
        let memory = LinearMemory::new(1);
        
        // Valid access
        let result = memory.read_u32(0);
        assert!(result.is_ok());
        
        // Out of bounds access
        let result = memory.read_u32(65536);
        assert!(result.is_err());
    }

    #[test]
    fn test_memory_write_read() {
        let mut memory = LinearMemory::new(1);
        
        let data = vec![0x01, 0x02, 0x03, 0x04];
        memory.write(0, &data).unwrap();
        
        let read_data = memory.read(0, 4).unwrap();
        assert_eq!(data, read_data);
    }
}

#[cfg(test)]
mod sandbox_tests {
    use super::*;
    use wasm_engine::sandbox::*;

    #[test]
    fn test_resource_limits() {
        let limits = ResourceLimits {
            max_memory: 1024 * 1024, // 1MB
            max_execution_time: std::time::Duration::from_secs(5),
            max_stack_depth: 1000,
        };
        
        let sandbox = Sandbox::new(limits);
        assert_eq!(sandbox.limits().max_memory, 1024 * 1024);
    }

    #[test]
    fn test_capability_enforcement() {
        let mut capabilities = Capabilities::new();
        capabilities.allow_file_access = false;
        capabilities.allow_network_access = false;
        
        let sandbox = Sandbox::with_capabilities(capabilities);
        
        // Should deny file access
        let result = sandbox.check_file_access("/etc/passwd");
        assert!(result.is_err());
        
        // Should deny network access
        let result = sandbox.check_network_access("127.0.0.1", 80);
        assert!(result.is_err());
    }
}

#[cfg(test)]
mod vm_tests {
    use super::*;
    use wasm_engine::vm::*;

    #[test]
    fn test_vm_initialization() {
        let vm = WasmVM::new();
        assert_eq!(vm.state(), VMState::Initialized);
    }

    #[test]
    fn test_module_loading() {
        let mut vm = WasmVM::new();
        let wasm_bytes = include_bytes!("../testdata/simple.wasm");
        
        let result = vm.load_module("test", wasm_bytes);
        assert!(result.is_ok());
        
        let module = vm.get_module("test");
        assert!(module.is_some());
    }

    #[test]
    fn test_function_execution() {
        let mut vm = WasmVM::new();
        let wasm_bytes = include_bytes!("../testdata/add.wasm");
        
        vm.load_module("math", wasm_bytes).unwrap();
        
        let result = vm.call_function("math", "add", &[Value::I32(5), Value::I32(3)]);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), vec![Value::I32(8)]);
    }
}

#[cfg(test)]
mod benchmarks {
    use super::*;
    use test::Bencher;

    #[bench]
    fn bench_wasm_parsing(b: &mut Bencher) {
        let wasm_bytes = include_bytes!("../testdata/complex.wasm");
        let parser = WasmParser::new();
        
        b.iter(|| {
            parser.parse_module(wasm_bytes).unwrap()
        });
    }

    #[bench]
    fn bench_memory_access(b: &mut Bencher) {
        let mut memory = LinearMemory::new(1);
        
        b.iter(|| {
            memory.write_u32(0, 0x12345678).unwrap();
            memory.read_u32(0).unwrap()
        });
    }

    #[bench]
    fn bench_function_call(b: &mut Bencher) {
        let mut vm = WasmVM::new();
        let wasm_bytes = include_bytes!("../testdata/fibonacci.wasm");
        vm.load_module("fib", wasm_bytes).unwrap();
        
        b.iter(|| {
            vm.call_function("fib", "fib", &[Value::I32(10)]).unwrap()
        });
    }
}