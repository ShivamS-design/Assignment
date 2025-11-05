use super::*;
use crate::parser::WasmModule;
use crate::error::{WasmError, Result};
use std::time::Instant;

pub struct FastAnalyzer {
    static_analyzer: StaticAnalyzer,
    cache: AnalysisCache,
}

struct AnalysisCache {
    results: std::collections::HashMap<String, AnalysisResult>,
    max_entries: usize,
}

impl FastAnalyzer {
    pub fn new() -> Self {
        Self {
            static_analyzer: StaticAnalyzer::new(),
            cache: AnalysisCache {
                results: std::collections::HashMap::new(),
                max_entries: 100,
            },
        }
    }

    pub fn analyze_fast(&mut self, module: &WasmModule, module_hash: &str) -> Result<AnalysisResult> {
        // Check cache first
        if let Some(cached) = self.cache.results.get(module_hash) {
            return Ok(cached.clone());
        }

        let start_time = Instant::now();
        
        // Perform fast analysis
        let result = self.fast_analysis_pipeline(module)?;
        
        // Cache result if analysis was fast enough
        if start_time.elapsed().as_millis() < 1000 {
            self.cache_result(module_hash.to_string(), result.clone());
        }

        Ok(result)
    }

    fn fast_analysis_pipeline(&self, module: &WasmModule) -> Result<AnalysisResult> {
        let start_time = Instant::now();

        // Quick module info extraction
        let module_info = self.extract_module_info_fast(module);
        
        // Fast security assessment
        let security_assessment = self.fast_security_assessment(module)?;
        
        // Quick capability inference
        let capability_requirements = self.fast_capability_inference(module, &security_assessment)?;
        
        // Calculate risk score
        let risk_score = self.calculate_risk_score_fast(&security_assessment, &capability_requirements);
        
        // Generate basic recommendations
        let recommendations = self.generate_fast_recommendations(&risk_score);

        Ok(AnalysisResult {
            module_info,
            security_assessment,
            capability_requirements,
            risk_score,
            recommendations,
            analysis_time: start_time.elapsed(),
        })
    }

    fn extract_module_info_fast(&self, module: &WasmModule) -> ModuleInfo {
        ModuleInfo {
            size: 0, // Would calculate from binary size
            function_count: module.functions.len(),
            import_count: 0, // Quick count from imports section
            export_count: module.exports.len(),
            memory_pages: module.memory.as_ref().map(|m| m.min),
            table_size: None, // Quick extraction
            global_count: 0, // Quick count from globals section
        }
    }

    fn fast_security_assessment(&self, module: &WasmModule) -> Result<SecurityAssessment> {
        let mut memory_patterns = Vec::new();
        let mut suspicious_patterns = Vec::new();
        let mut syscall_functions = Vec::new();
        let mut control_flow_complexity = 0;

        // Fast pattern matching - only check critical patterns
        for (func_idx, code_section) in module.code.iter().enumerate() {
            let bytecode = &code_section.body;
            
            // Quick complexity calculation
            control_flow_complexity += self.quick_complexity_calc(bytecode);
            
            // Check for critical patterns only
            if self.contains_critical_pattern(bytecode) {
                suspicious_patterns.push(SuspiciousPattern {
                    pattern_name: "CriticalPattern".to_string(),
                    function_index: func_idx as u32,
                    instruction_offset: 0,
                    description: "Critical security pattern detected".to_string(),
                    risk_level: RiskLevel::Severe,
                });
            }

            // Quick memory pattern check
            if bytecode.contains(&0x40) { // memory.grow
                memory_patterns.push(MemoryPattern {
                    pattern_type: "MemoryGrowth".to_string(),
                    locations: vec![func_idx as u32],
                    risk_level: RiskLevel::Warning,
                    description: "Memory growth detected".to_string(),
                });
            }
        }

        // Quick syscall detection
        for export in &module.exports {
            if self.is_syscall_name(&export.name) {
                syscall_functions.push(SyscallFunction {
                    name: export.name.clone(),
                    import_index: 0,
                    usage_count: 1, // Simplified count
                    risk_level: self.quick_syscall_risk(&export.name),
                });
            }
        }

        let resource_requirements = self.quick_resource_estimation(module);

        Ok(SecurityAssessment {
            memory_patterns,
            control_flow_complexity,
            suspicious_patterns,
            syscall_functions,
            resource_requirements,
        })
    }

    fn quick_complexity_calc(&self, bytecode: &[u8]) -> u32 {
        let mut complexity = 1;
        for &opcode in bytecode {
            match opcode {
                0x04 | 0x05 | 0x03 | 0x02 => complexity += 1, // if, else, loop, block
                0x0C | 0x0D => complexity += 1, // br, br_if
                _ => {}
            }
        }
        complexity
    }

    fn contains_critical_pattern(&self, bytecode: &[u8]) -> bool {
        // Check for infinite loop pattern
        bytecode.windows(5).any(|window| {
            matches!(window, [0x03, 0x40, 0x0C, 0x00, 0x0B]) // loop, br 0, end
        })
    }

    fn is_syscall_name(&self, name: &str) -> bool {
        name.starts_with("wasm_") || name.contains("syscall") || name.contains("host_")
    }

    fn quick_syscall_risk(&self, name: &str) -> RiskLevel {
        if name.contains("exec") || name.contains("system") || name.contains("socket") {
            RiskLevel::Severe
        } else if name.contains("read") || name.contains("write") || name.contains("open") {
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        }
    }

    fn quick_resource_estimation(&self, module: &WasmModule) -> ResourceRequirements {
        let estimated_memory = module.memory
            .as_ref()
            .map(|m| (m.min as u64) * 65536)
            .unwrap_or(0);

        let estimated_cpu_cycles = (module.functions.len() as u64) * 1000; // Rough estimate
        
        ResourceRequirements {
            estimated_memory,
            estimated_cpu_cycles,
            max_stack_depth: 32, // Conservative estimate
            max_call_depth: 16,  // Conservative estimate
        }
    }

    fn fast_capability_inference(&self, module: &WasmModule, security: &SecurityAssessment) -> Result<CapabilityRequirements> {
        let mut required_capabilities = Vec::new();
        let mut inferred_permissions = Vec::new();

        // Quick capability inference based on exports
        for export in &module.exports {
            match export.name.as_str() {
                name if name.contains("log") => {
                    required_capabilities.push("Log".to_string());
                    inferred_permissions.push(Permission {
                        name: "Log".to_string(),
                        required: true,
                        reason: "Logging function exported".to_string(),
                    });
                }
                name if name.contains("time") => {
                    required_capabilities.push("GetTime".to_string());
                }
                name if name.contains("random") => {
                    required_capabilities.push("Random".to_string());
                }
                name if name.contains("sensor") => {
                    required_capabilities.push("ReadSensor".to_string());
                }
                name if name.contains("alert") => {
                    required_capabilities.push("SendAlert".to_string());
                }
                _ => {}
            }
        }

        // Memory-based capabilities
        if let Some(memory) = &module.memory {
            if memory.min > 10 {
                required_capabilities.push("MemoryGrow".to_string());
            }
        }

        // Security-based capabilities
        if security.control_flow_complexity > 50 {
            required_capabilities.push("HighComplexity".to_string());
        }

        Ok(CapabilityRequirements {
            required_capabilities,
            optional_capabilities: Vec::new(),
            inferred_permissions,
        })
    }

    fn calculate_risk_score_fast(&self, security: &SecurityAssessment, _capabilities: &CapabilityRequirements) -> RiskScore {
        let mut score = 0u32;

        // Memory risk
        let memory_risk = if !security.memory_patterns.is_empty() {
            score += 20;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        // Execution risk
        let execution_risk = if security.control_flow_complexity > 50 {
            score += 25;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        // Syscall risk
        let syscall_risk = if security.syscall_functions.iter().any(|s| s.risk_level == RiskLevel::Severe) {
            score += 30;
            RiskLevel::Severe
        } else if !security.syscall_functions.is_empty() {
            score += 10;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        // Complexity risk
        let complexity_risk = if !security.suspicious_patterns.is_empty() {
            score += 25;
            RiskLevel::Severe
        } else {
            RiskLevel::OK
        };

        let overall = if score >= 60 {
            RiskLevel::Severe
        } else if score >= 25 {
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        RiskScore {
            overall,
            memory_risk,
            execution_risk,
            syscall_risk,
            complexity_risk,
            score,
        }
    }

    fn generate_fast_recommendations(&self, risk: &RiskScore) -> Vec<Recommendation> {
        let mut recommendations = Vec::new();

        match risk.overall {
            RiskLevel::Severe => {
                recommendations.push(Recommendation {
                    category: "Security".to_string(),
                    message: "Module poses significant security risks".to_string(),
                    severity: RiskLevel::Severe,
                    action: "Apply maximum sandbox restrictions or reject module".to_string(),
                });
            }
            RiskLevel::Warning => {
                recommendations.push(Recommendation {
                    category: "Security".to_string(),
                    message: "Module requires careful monitoring".to_string(),
                    severity: RiskLevel::Warning,
                    action: "Apply moderate sandbox restrictions".to_string(),
                });
            }
            RiskLevel::OK => {
                recommendations.push(Recommendation {
                    category: "Security".to_string(),
                    message: "Module appears safe for execution".to_string(),
                    severity: RiskLevel::OK,
                    action: "Apply standard sandbox restrictions".to_string(),
                });
            }
        }

        recommendations
    }

    fn cache_result(&mut self, hash: String, result: AnalysisResult) {
        if self.cache.results.len() >= self.cache.max_entries {
            // Simple LRU eviction - remove first entry
            if let Some(first_key) = self.cache.results.keys().next().cloned() {
                self.cache.results.remove(&first_key);
            }
        }
        self.cache.results.insert(hash, result);
    }

    pub fn clear_cache(&mut self) {
        self.cache.results.clear();
    }

    pub fn get_cache_stats(&self) -> CacheStats {
        CacheStats {
            entries: self.cache.results.len(),
            max_entries: self.cache.max_entries,
            hit_rate: 0.0, // Would track in real implementation
        }
    }
}

#[derive(Debug)]
pub struct CacheStats {
    pub entries: usize,
    pub max_entries: usize,
    pub hit_rate: f64,
}

pub fn calculate_module_hash(module: &WasmModule) -> String {
    // Simple hash calculation - in production would use proper hashing
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    module.functions.len().hash(&mut hasher);
    module.exports.len().hash(&mut hasher);
    if let Some(memory) = &module.memory {
        memory.min.hash(&mut hasher);
    }
    
    format!("{:x}", hasher.finish())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser::{WasmModule, FunctionType, Function, Export, ExportKind, CodeSection};

    #[test]
    fn test_fast_analyzer() {
        let mut analyzer = FastAnalyzer::new();
        let module = create_test_module();
        let hash = calculate_module_hash(&module);
        
        let result = analyzer.analyze_fast(&module, &hash);
        assert!(result.is_ok());
        
        let analysis = result.unwrap();
        assert!(analysis.analysis_time.as_millis() < 100); // Should be fast
    }

    #[test]
    fn test_caching() {
        let mut analyzer = FastAnalyzer::new();
        let module = create_test_module();
        let hash = calculate_module_hash(&module);
        
        // First analysis
        let result1 = analyzer.analyze_fast(&module, &hash);
        assert!(result1.is_ok());
        
        // Second analysis should use cache
        let result2 = analyzer.analyze_fast(&module, &hash);
        assert!(result2.is_ok());
        
        let stats = analyzer.get_cache_stats();
        assert_eq!(stats.entries, 1);
    }

    #[test]
    fn test_critical_pattern_detection() {
        let analyzer = FastAnalyzer::new();
        let bytecode = vec![0x03, 0x40, 0x0C, 0x00, 0x0B]; // infinite loop pattern
        
        assert!(analyzer.contains_critical_pattern(&bytecode));
    }

    #[test]
    fn test_syscall_risk_assessment() {
        let analyzer = FastAnalyzer::new();
        
        assert_eq!(analyzer.quick_syscall_risk("wasm_exec"), RiskLevel::Severe);
        assert_eq!(analyzer.quick_syscall_risk("wasm_read"), RiskLevel::Warning);
        assert_eq!(analyzer.quick_syscall_risk("wasm_log"), RiskLevel::OK);
    }

    fn create_test_module() -> WasmModule {
        WasmModule {
            types: vec![FunctionType { params: vec![], results: vec![] }],
            functions: vec![Function { type_idx: 0 }],
            memory: None,
            exports: vec![Export {
                name: "wasm_log".to_string(),
                kind: ExportKind::Function,
                index: 0,
            }],
            code: vec![CodeSection {
                locals: vec![],
                body: vec![0x41, 0x01], // i32.const 1
            }],
        }
    }
}