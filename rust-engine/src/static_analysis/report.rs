use super::{AnalysisResult, RiskLevel, RiskScore};
use std::fmt::Write;

pub struct ReportGenerator;

impl ReportGenerator {
    pub fn generate_text_report(analysis: &AnalysisResult) -> String {
        let mut report = String::new();
        
        writeln!(report, "=== WASM Static Analysis Report ===").unwrap();
        writeln!(report, "Analysis completed in {:?}", analysis.analysis_time).unwrap();
        writeln!(report).unwrap();

        // Module Information
        Self::write_module_info(&mut report, &analysis.module_info);
        
        // Risk Assessment
        Self::write_risk_assessment(&mut report, &analysis.risk_score);
        
        // Security Assessment
        Self::write_security_assessment(&mut report, &analysis.security_assessment);
        
        // Capability Requirements
        Self::write_capability_requirements(&mut report, &analysis.capability_requirements);
        
        // Recommendations
        Self::write_recommendations(&mut report, &analysis.recommendations);

        report
    }

    pub fn generate_json_report(analysis: &AnalysisResult) -> String {
        // Simplified JSON generation - in production would use serde
        format!(r#"{{
  "analysis_time_ms": {},
  "module_info": {{
    "size": {},
    "function_count": {},
    "export_count": {},
    "memory_pages": {}
  }},
  "risk_score": {{
    "overall": "{}",
    "score": {},
    "memory_risk": "{}",
    "execution_risk": "{}",
    "syscall_risk": "{}",
    "complexity_risk": "{}"
  }},
  "security_assessment": {{
    "control_flow_complexity": {},
    "memory_patterns_count": {},
    "suspicious_patterns_count": {},
    "syscall_functions_count": {}
  }},
  "capability_requirements": {{
    "required_capabilities": {},
    "optional_capabilities": {},
    "inferred_permissions": {}
  }},
  "recommendations_count": {}
}}"#,
            analysis.analysis_time.as_millis(),
            analysis.module_info.size,
            analysis.module_info.function_count,
            analysis.module_info.export_count,
            analysis.module_info.memory_pages.unwrap_or(0),
            Self::risk_level_to_string(&analysis.risk_score.overall),
            analysis.risk_score.score,
            Self::risk_level_to_string(&analysis.risk_score.memory_risk),
            Self::risk_level_to_string(&analysis.risk_score.execution_risk),
            Self::risk_level_to_string(&analysis.risk_score.syscall_risk),
            Self::risk_level_to_string(&analysis.risk_score.complexity_risk),
            analysis.security_assessment.control_flow_complexity,
            analysis.security_assessment.memory_patterns.len(),
            analysis.security_assessment.suspicious_patterns.len(),
            analysis.security_assessment.syscall_functions.len(),
            analysis.capability_requirements.required_capabilities.len(),
            analysis.capability_requirements.optional_capabilities.len(),
            analysis.capability_requirements.inferred_permissions.len(),
            analysis.recommendations.len()
        )
    }

    pub fn generate_html_report(analysis: &AnalysisResult) -> String {
        let mut html = String::new();
        
        writeln!(html, r#"<!DOCTYPE html>
<html>
<head>
    <title>WASM Static Analysis Report</title>
    <style>
        body {{ font-family: Arial, sans-serif; margin: 20px; }}
        .header {{ background: #f0f0f0; padding: 20px; border-radius: 5px; }}
        .section {{ margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }}
        .risk-ok {{ color: green; }}
        .risk-warning {{ color: orange; }}
        .risk-severe {{ color: red; }}
        .risk-indicator {{ font-weight: bold; font-size: 1.2em; }}
        table {{ border-collapse: collapse; width: 100%; }}
        th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
        th {{ background-color: #f2f2f2; }}
        .progress-bar {{ width: 100%; height: 20px; background: #f0f0f0; border-radius: 10px; }}
        .progress-fill {{ height: 100%; border-radius: 10px; }}
    </style>
</head>
<body>"#).unwrap();

        // Header
        writeln!(html, r#"<div class="header">
    <h1>WASM Static Analysis Report</h1>
    <p>Analysis completed in {:?}</p>
</div>"#, analysis.analysis_time).unwrap();

        // Risk Overview
        Self::write_html_risk_overview(&mut html, &analysis.risk_score);
        
        // Module Information
        Self::write_html_module_info(&mut html, &analysis.module_info);
        
        // Security Assessment
        Self::write_html_security_assessment(&mut html, &analysis.security_assessment);
        
        // Capabilities
        Self::write_html_capabilities(&mut html, &analysis.capability_requirements);
        
        // Recommendations
        Self::write_html_recommendations(&mut html, &analysis.recommendations);

        writeln!(html, "</body></html>").unwrap();
        html
    }

    fn write_module_info(report: &mut String, info: &super::ModuleInfo) {
        writeln!(report, "--- Module Information ---").unwrap();
        writeln!(report, "Functions: {}", info.function_count).unwrap();
        writeln!(report, "Exports: {}", info.export_count).unwrap();
        writeln!(report, "Imports: {}", info.import_count).unwrap();
        if let Some(pages) = info.memory_pages {
            writeln!(report, "Memory Pages: {} ({} KB)", pages, pages * 64).unwrap();
        }
        writeln!(report).unwrap();
    }

    fn write_risk_assessment(report: &mut String, risk: &RiskScore) {
        writeln!(report, "--- Risk Assessment ---").unwrap();
        writeln!(report, "Overall Risk: {} (Score: {}/100)", 
                Self::risk_level_to_string(&risk.overall), risk.score).unwrap();
        writeln!(report, "Memory Risk: {}", Self::risk_level_to_string(&risk.memory_risk)).unwrap();
        writeln!(report, "Execution Risk: {}", Self::risk_level_to_string(&risk.execution_risk)).unwrap();
        writeln!(report, "Syscall Risk: {}", Self::risk_level_to_string(&risk.syscall_risk)).unwrap();
        writeln!(report, "Complexity Risk: {}", Self::risk_level_to_string(&risk.complexity_risk)).unwrap();
        writeln!(report).unwrap();
    }

    fn write_security_assessment(report: &mut String, security: &super::SecurityAssessment) {
        writeln!(report, "--- Security Assessment ---").unwrap();
        writeln!(report, "Control Flow Complexity: {}", security.control_flow_complexity).unwrap();
        writeln!(report, "Memory Patterns Found: {}", security.memory_patterns.len()).unwrap();
        writeln!(report, "Suspicious Patterns: {}", security.suspicious_patterns.len()).unwrap();
        writeln!(report, "Syscall Functions: {}", security.syscall_functions.len()).unwrap();
        
        if !security.suspicious_patterns.is_empty() {
            writeln!(report, "\nSuspicious Patterns:").unwrap();
            for pattern in &security.suspicious_patterns {
                writeln!(report, "  - {} (Function {}, Offset {}): {}", 
                        pattern.pattern_name, pattern.function_index, 
                        pattern.instruction_offset, pattern.description).unwrap();
            }
        }
        
        if !security.syscall_functions.is_empty() {
            writeln!(report, "\nSyscall Functions:").unwrap();
            for syscall in &security.syscall_functions {
                writeln!(report, "  - {} (Risk: {}, Usage: {})", 
                        syscall.name, Self::risk_level_to_string(&syscall.risk_level), 
                        syscall.usage_count).unwrap();
            }
        }
        writeln!(report).unwrap();
    }

    fn write_capability_requirements(report: &mut String, capabilities: &super::CapabilityRequirements) {
        writeln!(report, "--- Capability Requirements ---").unwrap();
        
        if !capabilities.required_capabilities.is_empty() {
            writeln!(report, "Required Capabilities:").unwrap();
            for cap in &capabilities.required_capabilities {
                writeln!(report, "  - {}", cap).unwrap();
            }
        }
        
        if !capabilities.optional_capabilities.is_empty() {
            writeln!(report, "Optional Capabilities:").unwrap();
            for cap in &capabilities.optional_capabilities {
                writeln!(report, "  - {}", cap).unwrap();
            }
        }
        
        if !capabilities.inferred_permissions.is_empty() {
            writeln!(report, "Inferred Permissions:").unwrap();
            for perm in &capabilities.inferred_permissions {
                writeln!(report, "  - {} ({}): {}", 
                        perm.name, 
                        if perm.required { "Required" } else { "Optional" },
                        perm.reason).unwrap();
            }
        }
        writeln!(report).unwrap();
    }

    fn write_recommendations(report: &mut String, recommendations: &[super::Recommendation]) {
        if recommendations.is_empty() {
            return;
        }
        
        writeln!(report, "--- Recommendations ---").unwrap();
        for rec in recommendations {
            writeln!(report, "{} [{}]: {}", 
                    rec.category, Self::risk_level_to_string(&rec.severity), rec.message).unwrap();
            writeln!(report, "  Action: {}", rec.action).unwrap();
        }
        writeln!(report).unwrap();
    }

    fn write_html_risk_overview(html: &mut String, risk: &RiskScore) {
        let risk_class = match risk.overall {
            RiskLevel::OK => "risk-ok",
            RiskLevel::Warning => "risk-warning",
            RiskLevel::Severe => "risk-severe",
        };

        writeln!(html, r#"<div class="section">
    <h2>Risk Overview</h2>
    <div class="risk-indicator {}">Overall Risk: {} (Score: {}/100)</div>
    <div class="progress-bar">
        <div class="progress-fill" style="width: {}%; background: {};"></div>
    </div>
    <table>
        <tr><th>Category</th><th>Risk Level</th></tr>
        <tr><td>Memory</td><td class="{}">{}</td></tr>
        <tr><td>Execution</td><td class="{}">{}</td></tr>
        <tr><td>Syscalls</td><td class="{}">{}</td></tr>
        <tr><td>Complexity</td><td class="{}">{}</td></tr>
    </table>
</div>"#,
            risk_class,
            Self::risk_level_to_string(&risk.overall),
            risk.score,
            risk.score,
            Self::get_risk_color(&risk.overall),
            Self::get_risk_class(&risk.memory_risk),
            Self::risk_level_to_string(&risk.memory_risk),
            Self::get_risk_class(&risk.execution_risk),
            Self::risk_level_to_string(&risk.execution_risk),
            Self::get_risk_class(&risk.syscall_risk),
            Self::risk_level_to_string(&risk.syscall_risk),
            Self::get_risk_class(&risk.complexity_risk),
            Self::risk_level_to_string(&risk.complexity_risk)
        ).unwrap();
    }

    fn write_html_module_info(html: &mut String, info: &super::ModuleInfo) {
        writeln!(html, r#"<div class="section">
    <h2>Module Information</h2>
    <table>
        <tr><th>Property</th><th>Value</th></tr>
        <tr><td>Functions</td><td>{}</td></tr>
        <tr><td>Exports</td><td>{}</td></tr>
        <tr><td>Imports</td><td>{}</td></tr>
        <tr><td>Memory Pages</td><td>{}</td></tr>
    </table>
</div>"#,
            info.function_count,
            info.export_count,
            info.import_count,
            info.memory_pages.map(|p| p.to_string()).unwrap_or_else(|| "None".to_string())
        ).unwrap();
    }

    fn write_html_security_assessment(html: &mut String, security: &super::SecurityAssessment) {
        writeln!(html, r#"<div class="section">
    <h2>Security Assessment</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Control Flow Complexity</td><td>{}</td></tr>
        <tr><td>Memory Patterns</td><td>{}</td></tr>
        <tr><td>Suspicious Patterns</td><td>{}</td></tr>
        <tr><td>Syscall Functions</td><td>{}</td></tr>
    </table>
</div>"#,
            security.control_flow_complexity,
            security.memory_patterns.len(),
            security.suspicious_patterns.len(),
            security.syscall_functions.len()
        ).unwrap();
    }

    fn write_html_capabilities(html: &mut String, capabilities: &super::CapabilityRequirements) {
        writeln!(html, r#"<div class="section">
    <h2>Capability Requirements</h2>
    <h3>Required Capabilities</h3>
    <ul>"#).unwrap();
    
    for cap in &capabilities.required_capabilities {
        writeln!(html, "<li>{}</li>", cap).unwrap();
    }
    
    writeln!(html, "</ul></div>").unwrap();
    }

    fn write_html_recommendations(html: &mut String, recommendations: &[super::Recommendation]) {
        if recommendations.is_empty() {
            return;
        }
        
        writeln!(html, r#"<div class="section">
    <h2>Recommendations</h2>"#).unwrap();
        
        for rec in recommendations {
            let risk_class = Self::get_risk_class(&rec.severity);
            writeln!(html, r#"<div class="{}">
        <strong>{}:</strong> {}<br>
        <em>Action: {}</em>
    </div>"#, risk_class, rec.category, rec.message, rec.action).unwrap();
        }
        
        writeln!(html, "</div>").unwrap();
    }

    fn risk_level_to_string(level: &RiskLevel) -> &'static str {
        match level {
            RiskLevel::OK => "OK",
            RiskLevel::Warning => "WARNING",
            RiskLevel::Severe => "SEVERE",
        }
    }

    fn get_risk_class(level: &RiskLevel) -> &'static str {
        match level {
            RiskLevel::OK => "risk-ok",
            RiskLevel::Warning => "risk-warning",
            RiskLevel::Severe => "risk-severe",
        }
    }

    fn get_risk_color(level: &RiskLevel) -> &'static str {
        match level {
            RiskLevel::OK => "#4CAF50",
            RiskLevel::Warning => "#FF9800",
            RiskLevel::Severe => "#F44336",
        }
    }

    pub fn generate_summary(analysis: &AnalysisResult) -> String {
        format!("WASM Analysis: {} risk (score: {}/100) - {} functions, {} capabilities required",
                Self::risk_level_to_string(&analysis.risk_score.overall),
                analysis.risk_score.score,
                analysis.module_info.function_count,
                analysis.capability_requirements.required_capabilities.len())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_text_report_generation() {
        let analysis = create_test_analysis();
        let report = ReportGenerator::generate_text_report(&analysis);
        
        assert!(report.contains("WASM Static Analysis Report"));
        assert!(report.contains("Module Information"));
        assert!(report.contains("Risk Assessment"));
    }

    #[test]
    fn test_json_report_generation() {
        let analysis = create_test_analysis();
        let report = ReportGenerator::generate_json_report(&analysis);
        
        assert!(report.contains("analysis_time_ms"));
        assert!(report.contains("risk_score"));
        assert!(report.contains("module_info"));
    }

    #[test]
    fn test_html_report_generation() {
        let analysis = create_test_analysis();
        let report = ReportGenerator::generate_html_report(&analysis);
        
        assert!(report.contains("<!DOCTYPE html>"));
        assert!(report.contains("WASM Static Analysis Report"));
        assert!(report.contains("Risk Overview"));
    }

    fn create_test_analysis() -> AnalysisResult {
        AnalysisResult {
            module_info: super::ModuleInfo {
                size: 1024,
                function_count: 5,
                import_count: 2,
                export_count: 3,
                memory_pages: Some(1),
                table_size: None,
                global_count: 0,
            },
            security_assessment: super::SecurityAssessment {
                memory_patterns: vec![],
                control_flow_complexity: 10,
                suspicious_patterns: vec![],
                syscall_functions: vec![],
                resource_requirements: super::ResourceRequirements {
                    estimated_memory: 65536,
                    estimated_cpu_cycles: 1000,
                    max_stack_depth: 10,
                    max_call_depth: 5,
                },
            },
            capability_requirements: super::CapabilityRequirements {
                required_capabilities: vec!["Log".to_string()],
                optional_capabilities: vec![],
                inferred_permissions: vec![],
            },
            risk_score: RiskScore {
                overall: RiskLevel::OK,
                memory_risk: RiskLevel::OK,
                execution_risk: RiskLevel::OK,
                syscall_risk: RiskLevel::OK,
                complexity_risk: RiskLevel::OK,
                score: 15,
            },
            recommendations: vec![],
            analysis_time: Duration::from_millis(50),
        }
    }
}