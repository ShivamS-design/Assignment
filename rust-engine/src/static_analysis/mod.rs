pub mod analyzer;
pub mod patterns;
pub mod security;
pub mod capabilities;
pub mod report;

use crate::error::{WasmError, Result};
use crate::parser::WasmModule;
use std::time::{Duration, Instant};

#[derive(Debug, Clone)]
pub struct AnalysisResult {
    pub module_info: ModuleInfo,
    pub security_assessment: SecurityAssessment,
    pub capability_requirements: CapabilityRequirements,
    pub risk_score: RiskScore,
    pub recommendations: Vec<Recommendation>,
    pub analysis_time: Duration,
}

#[derive(Debug, Clone)]
pub struct ModuleInfo {
    pub size: usize,
    pub function_count: usize,
    pub import_count: usize,
    pub export_count: usize,
    pub memory_pages: Option<u32>,
    pub table_size: Option<u32>,
    pub global_count: usize,
}

#[derive(Debug, Clone)]
pub struct SecurityAssessment {
    pub memory_patterns: Vec<MemoryPattern>,
    pub control_flow_complexity: u32,
    pub suspicious_patterns: Vec<SuspiciousPattern>,
    pub syscall_functions: Vec<SyscallFunction>,
    pub resource_requirements: ResourceRequirements,
}

#[derive(Debug, Clone)]
pub struct CapabilityRequirements {
    pub required_capabilities: Vec<String>,
    pub optional_capabilities: Vec<String>,
    pub inferred_permissions: Vec<Permission>,
}

#[derive(Debug, Clone)]
pub struct RiskScore {
    pub overall: RiskLevel,
    pub memory_risk: RiskLevel,
    pub execution_risk: RiskLevel,
    pub syscall_risk: RiskLevel,
    pub complexity_risk: RiskLevel,
    pub score: u32, // 0-100
}

#[derive(Debug, Clone, PartialEq)]
pub enum RiskLevel {
    OK,
    Warning,
    Severe,
}

#[derive(Debug, Clone)]
pub struct Recommendation {
    pub category: String,
    pub message: String,
    pub severity: RiskLevel,
    pub action: String,
}

#[derive(Debug, Clone)]
pub struct MemoryPattern {
    pub pattern_type: String,
    pub locations: Vec<u32>,
    pub risk_level: RiskLevel,
    pub description: String,
}

#[derive(Debug, Clone)]
pub struct SuspiciousPattern {
    pub pattern_name: String,
    pub function_index: u32,
    pub instruction_offset: u32,
    pub description: String,
    pub risk_level: RiskLevel,
}

#[derive(Debug, Clone)]
pub struct SyscallFunction {
    pub name: String,
    pub import_index: u32,
    pub usage_count: u32,
    pub risk_level: RiskLevel,
}

#[derive(Debug, Clone)]
pub struct ResourceRequirements {
    pub estimated_memory: u64,
    pub estimated_cpu_cycles: u64,
    pub max_stack_depth: u32,
    pub max_call_depth: u32,
}

#[derive(Debug, Clone)]
pub struct Permission {
    pub name: String,
    pub required: bool,
    pub reason: String,
}

pub struct StaticAnalyzer {
    patterns: patterns::PatternMatcher,
    security: security::SecurityAnalyzer,
    capabilities: capabilities::CapabilityInferrer,
}

impl StaticAnalyzer {
    pub fn new() -> Self {
        Self {
            patterns: patterns::PatternMatcher::new(),
            security: security::SecurityAnalyzer::new(),
            capabilities: capabilities::CapabilityInferrer::new(),
        }
    }

    pub fn analyze(&self, module: &WasmModule) -> Result<AnalysisResult> {
        let start_time = Instant::now();

        let module_info = self.analyze_module_info(module);
        let security_assessment = self.security.analyze(module)?;
        let capability_requirements = self.capabilities.infer(module, &security_assessment)?;
        let risk_score = self.calculate_risk_score(&security_assessment, &capability_requirements);
        let recommendations = self.generate_recommendations(&security_assessment, &risk_score);

        Ok(AnalysisResult {
            module_info,
            security_assessment,
            capability_requirements,
            risk_score,
            recommendations,
            analysis_time: start_time.elapsed(),
        })
    }

    fn analyze_module_info(&self, module: &WasmModule) -> ModuleInfo {
        ModuleInfo {
            size: 0, // Would calculate from binary
            function_count: module.functions.len(),
            import_count: 0, // Would count imports
            export_count: module.exports.len(),
            memory_pages: module.memory.as_ref().map(|m| m.min),
            table_size: None,
            global_count: 0,
        }
    }

    fn calculate_risk_score(&self, security: &SecurityAssessment, _capabilities: &CapabilityRequirements) -> RiskScore {
        let mut score = 0u32;

        let memory_risk = if security.memory_patterns.iter().any(|p| p.risk_level == RiskLevel::Severe) {
            score += 30;
            RiskLevel::Severe
        } else if security.memory_patterns.iter().any(|p| p.risk_level == RiskLevel::Warning) {
            score += 15;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        let execution_risk = if security.control_flow_complexity > 100 {
            score += 25;
            RiskLevel::Severe
        } else if security.control_flow_complexity > 50 {
            score += 10;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        let syscall_risk = if security.syscall_functions.iter().any(|s| s.risk_level == RiskLevel::Severe) {
            score += 25;
            RiskLevel::Severe
        } else if security.syscall_functions.iter().any(|s| s.risk_level == RiskLevel::Warning) {
            score += 10;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        let complexity_risk = if security.suspicious_patterns.len() > 5 {
            score += 20;
            RiskLevel::Severe
        } else if security.suspicious_patterns.len() > 2 {
            score += 8;
            RiskLevel::Warning
        } else {
            RiskLevel::OK
        };

        let overall = if score >= 70 {
            RiskLevel::Severe
        } else if score >= 30 {
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

    fn generate_recommendations(&self, _security: &SecurityAssessment, risk: &RiskScore) -> Vec<Recommendation> {
        let mut recommendations = Vec::new();

        if risk.memory_risk != RiskLevel::OK {
            recommendations.push(Recommendation {
                category: "Memory".to_string(),
                message: "Module shows suspicious memory access patterns".to_string(),
                severity: risk.memory_risk.clone(),
                action: "Apply strict memory limits".to_string(),
            });
        }

        if risk.execution_risk != RiskLevel::OK {
            recommendations.push(Recommendation {
                category: "Execution".to_string(),
                message: "High control flow complexity detected".to_string(),
                severity: risk.execution_risk.clone(),
                action: "Limit execution time and instruction count".to_string(),
            });
        }

        if risk.overall == RiskLevel::Severe {
            recommendations.push(Recommendation {
                category: "General".to_string(),
                message: "Module poses significant security risks".to_string(),
                severity: RiskLevel::Severe,
                action: "Consider rejecting or applying maximum restrictions".to_string(),
            });
        }

        recommendations
    }
}