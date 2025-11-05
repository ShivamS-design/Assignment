use wasm_engine::static_analysis::*;
use wasm_engine::parser::{WasmModule, WasmParser, FunctionType, Function, Export, ExportKind, CodeSection, MemoryType};

#[test]
fn test_static_analyzer_basic() {
    let analyzer = StaticAnalyzer::new();
    let module = create_basic_module();
    
    let result = analyzer.analyze(&module);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert_eq!(analysis.module_info.function_count, 1);
    assert_eq!(analysis.module_info.export_count, 1);
}

#[test]
fn test_pattern_detection() {
    let analyzer = StaticAnalyzer::new();
    let module = create_suspicious_module();
    
    let result = analyzer.analyze(&module);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert!(!analysis.security_assessment.suspicious_patterns.is_empty());
    assert!(analysis.risk_score.overall != RiskLevel::OK);
}

#[test]
fn test_capability_inference() {
    let analyzer = StaticAnalyzer::new();
    let module = create_module_with_syscalls();
    
    let result = analyzer.analyze(&module);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert!(analysis.capability_requirements.required_capabilities.contains(&"Log".to_string()));
}

#[test]
fn test_memory_pattern_detection() {
    let analyzer = StaticAnalyzer::new();
    let module = create_memory_intensive_module();
    
    let result = analyzer.analyze(&module);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert!(!analysis.security_assessment.memory_patterns.is_empty());
    assert!(analysis.capability_requirements.required_capabilities.contains(&"MemoryGrow".to_string()));
}

#[test]
fn test_risk_scoring() {
    let analyzer = StaticAnalyzer::new();
    
    // Test low-risk module
    let safe_module = create_basic_module();
    let result = analyzer.analyze(&safe_module).unwrap();
    assert_eq!(result.risk_score.overall, RiskLevel::OK);
    assert!(result.risk_score.score < 30);
    
    // Test high-risk module
    let risky_module = create_risky_module();
    let result = analyzer.analyze(&risky_module).unwrap();
    assert!(result.risk_score.overall != RiskLevel::OK);
    assert!(result.risk_score.score > 30);
}

#[test]
fn test_fast_analyzer() {
    let mut fast_analyzer = analyzer::FastAnalyzer::new();
    let module = create_basic_module();
    let hash = analyzer::calculate_module_hash(&module);
    
    let result = fast_analyzer.analyze_fast(&module, &hash);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert!(analysis.analysis_time.as_millis() < 100); // Should be very fast
}

#[test]
fn test_analysis_caching() {
    let mut fast_analyzer = analyzer::FastAnalyzer::new();
    let module = create_basic_module();
    let hash = analyzer::calculate_module_hash(&module);
    
    // First analysis
    let start1 = std::time::Instant::now();
    let result1 = fast_analyzer.analyze_fast(&module, &hash).unwrap();
    let time1 = start1.elapsed();
    
    // Second analysis (should use cache)
    let start2 = std::time::Instant::now();
    let result2 = fast_analyzer.analyze_fast(&module, &hash).unwrap();
    let time2 = start2.elapsed();
    
    // Second analysis should be faster (cached)
    assert!(time2 <= time1);
    
    let stats = fast_analyzer.get_cache_stats();
    assert_eq!(stats.entries, 1);
}

#[test]
fn test_report_generation() {
    let analyzer = StaticAnalyzer::new();
    let module = create_basic_module();
    let analysis = analyzer.analyze(&module).unwrap();
    
    // Test text report
    let text_report = report::ReportGenerator::generate_text_report(&analysis);
    assert!(text_report.contains("WASM Static Analysis Report"));
    assert!(text_report.contains("Module Information"));
    assert!(text_report.contains("Risk Assessment"));
    
    // Test JSON report
    let json_report = report::ReportGenerator::generate_json_report(&analysis);
    assert!(json_report.contains("analysis_time_ms"));
    assert!(json_report.contains("risk_score"));
    
    // Test HTML report
    let html_report = report::ReportGenerator::generate_html_report(&analysis);
    assert!(html_report.contains("<!DOCTYPE html>"));
    assert!(html_report.contains("Risk Overview"));
}

#[test]
fn test_security_patterns() {
    use patterns::PatternMatcher;
    
    let matcher = PatternMatcher::new();
    let module = create_module_with_patterns();
    
    let patterns = matcher.find_patterns(&module);
    assert!(!patterns.is_empty());
    
    let complexity = matcher.analyze_control_flow(&module);
    assert!(complexity > 1);
    
    let data_flow = matcher.analyze_data_flow(&module);
    assert!(data_flow.memory_reads > 0 || data_flow.memory_writes > 0);
}

#[test]
fn test_capability_rules() {
    use capabilities::CapabilityInferrer;
    
    let inferrer = CapabilityInferrer::new();
    let module = create_module_with_syscalls();
    let security = create_mock_security_assessment();
    
    let result = inferrer.infer(&module, &security);
    assert!(result.is_ok());
    
    let capabilities = result.unwrap();
    assert!(!capabilities.required_capabilities.is_empty());
    
    let constraints = inferrer.recommend_sandbox_constraints(&capabilities, &security);
    assert!(constraints.max_memory_pages > 0);
    assert!(constraints.max_cpu_time_ms > 0);
}

#[test]
fn test_performance_benchmarks() {
    let analyzer = StaticAnalyzer::new();
    let modules = create_test_modules(10);
    
    let start = std::time::Instant::now();
    for module in &modules {
        let _ = analyzer.analyze(module);
    }
    let total_time = start.elapsed();
    
    let avg_time = total_time / modules.len() as u32;
    assert!(avg_time.as_millis() < 1000); // Should be sub-second per module
}

#[test]
fn test_malicious_patterns() {
    let analyzer = StaticAnalyzer::new();
    
    // Test infinite loop detection
    let infinite_loop_module = create_infinite_loop_module();
    let result = analyzer.analyze(&infinite_loop_module).unwrap();
    assert!(result.security_assessment.suspicious_patterns.iter()
        .any(|p| p.pattern_name == "InfiniteLoop"));
    
    // Test memory bomb detection
    let memory_bomb_module = create_memory_bomb_module();
    let result = analyzer.analyze(&memory_bomb_module).unwrap();
    assert!(result.security_assessment.memory_patterns.iter()
        .any(|p| p.pattern_type == "MemoryGrowth"));
}

#[test]
fn test_real_world_module() {
    // Test with a more realistic WASM module
    let wasm_bytes = create_realistic_wasm_binary();
    let module = WasmParser::parse(&wasm_bytes).unwrap();
    
    let analyzer = StaticAnalyzer::new();
    let result = analyzer.analyze(&module);
    assert!(result.is_ok());
    
    let analysis = result.unwrap();
    assert!(analysis.analysis_time.as_millis() < 500); // Should be fast
    assert!(!analysis.recommendations.is_empty());
}

// Helper functions to create test modules

fn create_basic_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: None,
        exports: vec![Export {
            name: "main".to_string(),
            kind: ExportKind::Function,
            index: 0,
        }],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![0x41, 0x01, 0x0B], // i32.const 1, end
        }],
    }
}

fn create_suspicious_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: None,
        exports: vec![],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![0x03, 0x40, 0x0C, 0x00, 0x0B], // loop, br 0, end (infinite loop)
        }],
    }
}

fn create_module_with_syscalls() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: None,
        exports: vec![
            Export {
                name: "wasm_log".to_string(),
                kind: ExportKind::Function,
                index: 0,
            },
            Export {
                name: "wasm_get_time".to_string(),
                kind: ExportKind::Function,
                index: 1,
            }
        ],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![0x41, 0x01, 0x0B],
        }],
    }
}

fn create_memory_intensive_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: Some(MemoryType { min: 100, max: Some(1000) }),
        exports: vec![],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![0x40, 0x00, 0x0B], // memory.grow, end
        }],
    }
}

fn create_risky_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: Some(MemoryType { min: 100, max: None }), // Unlimited memory
        exports: vec![Export {
            name: "wasm_exec".to_string(), // High-risk syscall
            kind: ExportKind::Function,
            index: 0,
        }],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![
                0x03, 0x40, 0x0C, 0x00, 0x0B, // Infinite loop
                0x40, 0x00, // Memory grow
                0x11, 0x00, // Indirect call
            ],
        }],
    }
}

fn create_module_with_patterns() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: Some(MemoryType { min: 1, max: Some(10) }),
        exports: vec![],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![
                0x41, 0x00, // i32.const 0
                0x28, 0x00, 0x00, // i32.load
                0x41, 0x01, // i32.const 1
                0x36, 0x00, 0x00, // i32.store
                0x23, 0x00, // global.get
                0x24, 0x00, // global.set
            ],
        }],
    }
}

fn create_infinite_loop_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: None,
        exports: vec![],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![0x03, 0x40, 0x0C, 0x00, 0x0B], // loop, br 0, end
        }],
    }
}

fn create_memory_bomb_module() -> WasmModule {
    WasmModule {
        types: vec![FunctionType { params: vec![], results: vec![] }],
        functions: vec![Function { type_idx: 0 }],
        memory: Some(MemoryType { min: 1, max: None }),
        exports: vec![],
        code: vec![CodeSection {
            locals: vec![],
            body: vec![
                0x03, 0x40, // loop
                0x41, 0x01, // i32.const 1
                0x40, 0x00, // memory.grow
                0x1A, // drop
                0x0C, 0x00, // br 0
                0x0B, // end
            ],
        }],
    }
}

fn create_test_modules(count: usize) -> Vec<WasmModule> {
    (0..count).map(|_| create_basic_module()).collect()
}

fn create_realistic_wasm_binary() -> Vec<u8> {
    vec![
        0x00, 0x61, 0x73, 0x6d, // magic
        0x01, 0x00, 0x00, 0x00, // version
        0x01, 0x05, 0x01, 0x60, 0x00, 0x00, // type section
        0x03, 0x02, 0x01, 0x00, // function section
        0x07, 0x09, 0x01, 0x05, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x00, // export section
        0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x01, 0x0b, // code section
    ]
}

fn create_mock_security_assessment() -> SecurityAssessment {
    SecurityAssessment {
        memory_patterns: vec![],
        control_flow_complexity: 10,
        suspicious_patterns: vec![],
        syscall_functions: vec![],
        resource_requirements: ResourceRequirements {
            estimated_memory: 65536,
            estimated_cpu_cycles: 1000,
            max_stack_depth: 10,
            max_call_depth: 5,
        },
    }
}