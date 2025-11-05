package static_analysis

import (
	"encoding/json"
	"fmt"
	"time"
)

type AnalysisRequest struct {
	ModuleData []byte `json:"module_data"`
	ModuleName string `json:"module_name"`
	FastMode   bool   `json:"fast_mode"`
}

type AnalysisResponse struct {
	ModuleInfo            ModuleInfo            `json:"module_info"`
	SecurityAssessment    SecurityAssessment    `json:"security_assessment"`
	CapabilityRequirements CapabilityRequirements `json:"capability_requirements"`
	RiskScore             RiskScore             `json:"risk_score"`
	Recommendations       []Recommendation      `json:"recommendations"`
	AnalysisTime          time.Duration         `json:"analysis_time"`
}

type ModuleInfo struct {
	Size          int    `json:"size"`
	FunctionCount int    `json:"function_count"`
	ImportCount   int    `json:"import_count"`
	ExportCount   int    `json:"export_count"`
	MemoryPages   *int   `json:"memory_pages,omitempty"`
	TableSize     *int   `json:"table_size,omitempty"`
	GlobalCount   int    `json:"global_count"`
}

type SecurityAssessment struct {
	MemoryPatterns         []MemoryPattern      `json:"memory_patterns"`
	ControlFlowComplexity  int                  `json:"control_flow_complexity"`
	SuspiciousPatterns     []SuspiciousPattern  `json:"suspicious_patterns"`
	SyscallFunctions       []SyscallFunction    `json:"syscall_functions"`
	ResourceRequirements   ResourceRequirements `json:"resource_requirements"`
}

type CapabilityRequirements struct {
	RequiredCapabilities []string     `json:"required_capabilities"`
	OptionalCapabilities []string     `json:"optional_capabilities"`
	InferredPermissions  []Permission `json:"inferred_permissions"`
}

type RiskScore struct {
	Overall         string `json:"overall"`
	MemoryRisk      string `json:"memory_risk"`
	ExecutionRisk   string `json:"execution_risk"`
	SyscallRisk     string `json:"syscall_risk"`
	ComplexityRisk  string `json:"complexity_risk"`
	Score           int    `json:"score"`
}

type Recommendation struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Action   string `json:"action"`
}

type MemoryPattern struct {
	PatternType string `json:"pattern_type"`
	Locations   []int  `json:"locations"`
	RiskLevel   string `json:"risk_level"`
	Description string `json:"description"`
}

type SuspiciousPattern struct {
	PatternName       string `json:"pattern_name"`
	FunctionIndex     int    `json:"function_index"`
	InstructionOffset int    `json:"instruction_offset"`
	Description       string `json:"description"`
	RiskLevel         string `json:"risk_level"`
}

type SyscallFunction struct {
	Name        string `json:"name"`
	ImportIndex int    `json:"import_index"`
	UsageCount  int    `json:"usage_count"`
	RiskLevel   string `json:"risk_level"`
}

type ResourceRequirements struct {
	EstimatedMemory   int64 `json:"estimated_memory"`
	EstimatedCPUCycles int64 `json:"estimated_cpu_cycles"`
	MaxStackDepth     int   `json:"max_stack_depth"`
	MaxCallDepth      int   `json:"max_call_depth"`
}

type Permission struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
}

type Analyzer struct {
	rustEngine RustEngineInterface
}

type RustEngineInterface interface {
	AnalyzeModule(moduleData []byte, fastMode bool) (string, error)
}

func NewAnalyzer(rustEngine RustEngineInterface) *Analyzer {
	return &Analyzer{
		rustEngine: rustEngine,
	}
}

func (a *Analyzer) AnalyzeModule(req AnalysisRequest) (*AnalysisResponse, error) {
	start := time.Now()
	
	// Call Rust engine for analysis
	resultJSON, err := a.rustEngine.AnalyzeModule(req.ModuleData, req.FastMode)
	if err != nil {
		return nil, fmt.Errorf("rust engine analysis failed: %w", err)
	}
	
	// Parse result
	var response AnalysisResponse
	if err := json.Unmarshal([]byte(resultJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse analysis result: %w", err)
	}
	
	response.AnalysisTime = time.Since(start)
	return &response, nil
}

func (a *Analyzer) QuickScan(moduleData []byte) (*RiskScore, error) {
	req := AnalysisRequest{
		ModuleData: moduleData,
		FastMode:   true,
	}
	
	result, err := a.AnalyzeModule(req)
	if err != nil {
		return nil, err
	}
	
	return &result.RiskScore, nil
}

func (a *Analyzer) GenerateReport(response *AnalysisResponse, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(response, "", "  ")
		return string(data), err
	case "text":
		return a.generateTextReport(response), nil
	case "html":
		return a.generateHTMLReport(response), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (a *Analyzer) generateTextReport(response *AnalysisResponse) string {
	report := fmt.Sprintf("=== WASM Static Analysis Report ===\n")
	report += fmt.Sprintf("Analysis completed in %v\n\n", response.AnalysisTime)
	
	// Module Info
	report += fmt.Sprintf("--- Module Information ---\n")
	report += fmt.Sprintf("Functions: %d\n", response.ModuleInfo.FunctionCount)
	report += fmt.Sprintf("Exports: %d\n", response.ModuleInfo.ExportCount)
	report += fmt.Sprintf("Imports: %d\n", response.ModuleInfo.ImportCount)
	if response.ModuleInfo.MemoryPages != nil {
		report += fmt.Sprintf("Memory Pages: %d\n", *response.ModuleInfo.MemoryPages)
	}
	report += "\n"
	
	// Risk Assessment
	report += fmt.Sprintf("--- Risk Assessment ---\n")
	report += fmt.Sprintf("Overall Risk: %s (Score: %d/100)\n", response.RiskScore.Overall, response.RiskScore.Score)
	report += fmt.Sprintf("Memory Risk: %s\n", response.RiskScore.MemoryRisk)
	report += fmt.Sprintf("Execution Risk: %s\n", response.RiskScore.ExecutionRisk)
	report += fmt.Sprintf("Syscall Risk: %s\n", response.RiskScore.SyscallRisk)
	report += fmt.Sprintf("Complexity Risk: %s\n", response.RiskScore.ComplexityRisk)
	report += "\n"
	
	// Security Assessment
	report += fmt.Sprintf("--- Security Assessment ---\n")
	report += fmt.Sprintf("Control Flow Complexity: %d\n", response.SecurityAssessment.ControlFlowComplexity)
	report += fmt.Sprintf("Memory Patterns: %d\n", len(response.SecurityAssessment.MemoryPatterns))
	report += fmt.Sprintf("Suspicious Patterns: %d\n", len(response.SecurityAssessment.SuspiciousPatterns))
	report += fmt.Sprintf("Syscall Functions: %d\n", len(response.SecurityAssessment.SyscallFunctions))
	report += "\n"
	
	// Capabilities
	report += fmt.Sprintf("--- Required Capabilities ---\n")
	for _, cap := range response.CapabilityRequirements.RequiredCapabilities {
		report += fmt.Sprintf("  - %s\n", cap)
	}
	report += "\n"
	
	// Recommendations
	if len(response.Recommendations) > 0 {
		report += fmt.Sprintf("--- Recommendations ---\n")
		for _, rec := range response.Recommendations {
			report += fmt.Sprintf("%s [%s]: %s\n", rec.Category, rec.Severity, rec.Message)
			report += fmt.Sprintf("  Action: %s\n", rec.Action)
		}
	}
	
	return report
}

func (a *Analyzer) generateHTMLReport(response *AnalysisResponse) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>WASM Static Analysis Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .risk-ok { color: green; }
        .risk-warning { color: orange; }
        .risk-severe { color: red; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>`
	
	html += fmt.Sprintf(`<div class="header">
    <h1>WASM Static Analysis Report</h1>
    <p>Analysis completed in %v</p>
</div>`, response.AnalysisTime)
	
	// Risk Overview
	riskClass := getRiskClass(response.RiskScore.Overall)
	html += fmt.Sprintf(`<div class="section">
    <h2>Risk Overview</h2>
    <div class="risk-indicator %s">Overall Risk: %s (Score: %d/100)</div>
    <table>
        <tr><th>Category</th><th>Risk Level</th></tr>
        <tr><td>Memory</td><td class="%s">%s</td></tr>
        <tr><td>Execution</td><td class="%s">%s</td></tr>
        <tr><td>Syscalls</td><td class="%s">%s</td></tr>
        <tr><td>Complexity</td><td class="%s">%s</td></tr>
    </table>
</div>`,
		riskClass, response.RiskScore.Overall, response.RiskScore.Score,
		getRiskClass(response.RiskScore.MemoryRisk), response.RiskScore.MemoryRisk,
		getRiskClass(response.RiskScore.ExecutionRisk), response.RiskScore.ExecutionRisk,
		getRiskClass(response.RiskScore.SyscallRisk), response.RiskScore.SyscallRisk,
		getRiskClass(response.RiskScore.ComplexityRisk), response.RiskScore.ComplexityRisk)
	
	html += "</body></html>"
	return html
}

func getRiskClass(level string) string {
	switch level {
	case "OK":
		return "risk-ok"
	case "WARNING":
		return "risk-warning"
	case "SEVERE":
		return "risk-severe"
	default:
		return ""
	}
}

// Mock Rust engine for testing
type MockRustEngine struct{}

func (m *MockRustEngine) AnalyzeModule(moduleData []byte, fastMode bool) (string, error) {
	// Mock response
	response := AnalysisResponse{
		ModuleInfo: ModuleInfo{
			Size:          len(moduleData),
			FunctionCount: 5,
			ExportCount:   3,
			ImportCount:   2,
		},
		RiskScore: RiskScore{
			Overall: "OK",
			Score:   15,
		},
		SecurityAssessment: SecurityAssessment{
			ControlFlowComplexity: 10,
		},
		CapabilityRequirements: CapabilityRequirements{
			RequiredCapabilities: []string{"Log"},
		},
		Recommendations: []Recommendation{
			{
				Category: "Security",
				Message:  "Module appears safe",
				Severity: "OK",
				Action:   "Apply standard restrictions",
			},
		},
	}
	
	data, _ := json.Marshal(response)
	return string(data), nil
}