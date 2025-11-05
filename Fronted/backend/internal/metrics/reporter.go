package metrics

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Reporter struct {
	collector *MetricsCollector
	templates map[string]ReportTemplate
}

type ReportTemplate struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Sections    []ReportSection   `json:"sections"`
	Parameters  map[string]string `json:"parameters"`
}

type ReportSection struct {
	Title   string      `json:"title"`
	Type    string      `json:"type"` // "table", "chart", "summary"
	Metrics []string    `json:"metrics"`
	Options interface{} `json:"options"`
}

type Report struct {
	Title       string                 `json:"title"`
	GeneratedAt time.Time              `json:"generated_at"`
	Period      ReportPeriod           `json:"period"`
	Summary     ReportSummary          `json:"summary"`
	Sections    []RenderedSection      `json:"sections"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ReportSummary struct {
	TotalOperations   int64         `json:"total_operations"`
	AverageMemory     int64         `json:"average_memory"`
	PeakMemory        int64         `json:"peak_memory"`
	TotalCPUTime      time.Duration `json:"total_cpu_time"`
	TopSyscalls       []SyscallStat `json:"top_syscalls"`
	PerformanceGrade  string        `json:"performance_grade"`
}

type RenderedSection struct {
	Title string      `json:"title"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
}

type SyscallStat struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type TrendAnalysis struct {
	Metric    string    `json:"metric"`
	Trend     string    `json:"trend"` // "increasing", "decreasing", "stable"
	Change    float64   `json:"change_percent"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func NewReporter(collector *MetricsCollector) *Reporter {
	r := &Reporter{
		collector: collector,
		templates: make(map[string]ReportTemplate),
	}
	r.loadDefaultTemplates()
	return r
}

func (r *Reporter) loadDefaultTemplates() {
	r.templates["performance"] = ReportTemplate{
		Name:        "Performance Report",
		Description: "Comprehensive performance analysis",
		Sections: []ReportSection{
			{
				Title:   "Performance Summary",
				Type:    "summary",
				Metrics: []string{"operations", "memory", "cpu_time"},
			},
			{
				Title:   "Time Series Analysis",
				Type:    "chart",
				Metrics: []string{"operations", "memory"},
			},
			{
				Title:   "Syscall Analysis",
				Type:    "table",
				Metrics: []string{"syscalls"},
			},
		},
	}
	
	r.templates["security"] = ReportTemplate{
		Name:        "Security Report",
		Description: "Security-focused metrics analysis",
		Sections: []ReportSection{
			{
				Title:   "Syscall Security Analysis",
				Type:    "table",
				Metrics: []string{"syscalls"},
			},
			{
				Title:   "Resource Usage Patterns",
				Type:    "chart",
				Metrics: []string{"memory", "cpu_time"},
			},
		},
	}
}

func (r *Reporter) GenerateReport(templateName string, period ReportPeriod) (*Report, error) {
	template, exists := r.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}
	
	timeSeries := r.getTimeSeriesForPeriod(period)
	currentMetrics := r.collector.GetRuntimeMetrics()
	
	report := &Report{
		Title:       template.Name,
		GeneratedAt: time.Now(),
		Period:      period,
		Summary:     r.generateSummary(timeSeries, currentMetrics),
		Sections:    make([]RenderedSection, 0),
		Metadata: map[string]interface{}{
			"template":    templateName,
			"data_points": len(timeSeries),
		},
	}
	
	for _, section := range template.Sections {
		rendered := r.renderSection(section, timeSeries, currentMetrics)
		report.Sections = append(report.Sections, rendered)
	}
	
	return report, nil
}

func (r *Reporter) generateSummary(timeSeries []TimeSeriesPoint, current RuntimeMetrics) ReportSummary {
	if len(timeSeries) == 0 {
		return ReportSummary{
			PeakMemory:       current.PeakMemory,
			TotalCPUTime:     current.CPUTime,
			PerformanceGrade: "N/A",
		}
	}
	
	var totalOps, totalMem int64
	var peakMem int64
	
	for _, point := range timeSeries {
		totalOps += point.Operations
		totalMem += point.Memory
		if point.Memory > peakMem {
			peakMem = point.Memory
		}
	}
	
	avgMem := totalMem / int64(len(timeSeries))
	
	// Generate top syscalls
	topSyscalls := make([]SyscallStat, 0)
	for name, count := range current.SyscallCounts {
		topSyscalls = append(topSyscalls, SyscallStat{
			Name:  name,
			Count: count,
		})
	}
	
	// Sort by count (simplified)
	for i := 0; i < len(topSyscalls)-1; i++ {
		for j := i + 1; j < len(topSyscalls); j++ {
			if topSyscalls[j].Count > topSyscalls[i].Count {
				topSyscalls[i], topSyscalls[j] = topSyscalls[j], topSyscalls[i]
			}
		}
	}
	
	if len(topSyscalls) > 5 {
		topSyscalls = topSyscalls[:5]
	}
	
	return ReportSummary{
		TotalOperations:  totalOps,
		AverageMemory:    avgMem,
		PeakMemory:       peakMem,
		TotalCPUTime:     current.CPUTime,
		TopSyscalls:      topSyscalls,
		PerformanceGrade: r.calculatePerformanceGrade(avgMem, totalOps, current.CPUTime),
	}
}

func (r *Reporter) renderSection(section ReportSection, timeSeries []TimeSeriesPoint, current RuntimeMetrics) RenderedSection {
	switch section.Type {
	case "summary":
		return RenderedSection{
			Title: section.Title,
			Type:  section.Type,
			Data:  r.generateSummary(timeSeries, current),
		}
	case "chart":
		return RenderedSection{
			Title: section.Title,
			Type:  section.Type,
			Data:  timeSeries,
		}
	case "table":
		return RenderedSection{
			Title: section.Title,
			Type:  section.Type,
			Data:  current.SyscallCounts,
		}
	default:
		return RenderedSection{
			Title: section.Title,
			Type:  section.Type,
			Data:  "Unknown section type",
		}
	}
}

func (r *Reporter) getTimeSeriesForPeriod(period ReportPeriod) []TimeSeriesPoint {
	allSeries := r.collector.GetTimeSeries()
	filtered := make([]TimeSeriesPoint, 0)
	
	for _, point := range allSeries {
		if point.Timestamp.After(period.Start) && point.Timestamp.Before(period.End) {
			filtered = append(filtered, point)
		}
	}
	
	return filtered
}

func (r *Reporter) calculatePerformanceGrade(avgMem, totalOps int64, cpuTime time.Duration) string {
	score := 100
	
	// Memory usage penalty
	if avgMem > 50*1024*1024 { // 50MB
		score -= 20
	} else if avgMem > 20*1024*1024 { // 20MB
		score -= 10
	}
	
	// Operations efficiency bonus
	if totalOps > 10000 {
		score += 10
	}
	
	// CPU time penalty
	if cpuTime > 10*time.Second {
		score -= 15
	} else if cpuTime > 5*time.Second {
		score -= 5
	}
	
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func (r *Reporter) AnalyzeTrends(period ReportPeriod) []TrendAnalysis {
	timeSeries := r.getTimeSeriesForPeriod(period)
	if len(timeSeries) < 2 {
		return nil
	}
	
	trends := make([]TrendAnalysis, 0)
	
	// Analyze operations trend
	opsTrend := r.calculateTrend(timeSeries, "operations")
	trends = append(trends, opsTrend)
	
	// Analyze memory trend
	memTrend := r.calculateTrend(timeSeries, "memory")
	trends = append(trends, memTrend)
	
	// Analyze CPU trend
	cpuTrend := r.calculateTrend(timeSeries, "cpu_time")
	trends = append(trends, cpuTrend)
	
	return trends
}

func (r *Reporter) calculateTrend(timeSeries []TimeSeriesPoint, metric string) TrendAnalysis {
	if len(timeSeries) < 2 {
		return TrendAnalysis{Metric: metric, Trend: "stable", Change: 0}
	}
	
	first := timeSeries[0]
	last := timeSeries[len(timeSeries)-1]
	
	var firstVal, lastVal float64
	
	switch metric {
	case "operations":
		firstVal = float64(first.Operations)
		lastVal = float64(last.Operations)
	case "memory":
		firstVal = float64(first.Memory)
		lastVal = float64(last.Memory)
	case "cpu_time":
		firstVal = float64(first.CPUTime)
		lastVal = float64(last.CPUTime)
	}
	
	change := (lastVal - firstVal) / firstVal * 100
	
	var trend string
	if change > 5 {
		trend = "increasing"
	} else if change < -5 {
		trend = "decreasing"
	} else {
		trend = "stable"
	}
	
	return TrendAnalysis{
		Metric:    metric,
		Trend:     trend,
		Change:    change,
		StartTime: first.Timestamp,
		EndTime:   last.Timestamp,
	}
}

func (r *Reporter) ExportJSON(report *Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func (r *Reporter) ExportCSV(report *Report) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	
	// Write header
	header := []string{"Timestamp", "Operations", "Memory", "CPU Time", "Syscalls"}
	writer.Write(header)
	
	// Write data from first chart section
	for _, section := range report.Sections {
		if section.Type == "chart" {
			if timeSeries, ok := section.Data.([]TimeSeriesPoint); ok {
				for _, point := range timeSeries {
					record := []string{
						point.Timestamp.Format(time.RFC3339),
						strconv.FormatInt(point.Operations, 10),
						strconv.FormatInt(point.Memory, 10),
						strconv.FormatInt(point.CPUTime, 10),
						strconv.FormatInt(point.Syscalls, 10),
					}
					writer.Write(record)
				}
			}
			break
		}
	}
	
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func (r *Reporter) GenerateComparativeReport(reports []*Report) *ComparativeReport {
	if len(reports) < 2 {
		return nil
	}
	
	comparative := &ComparativeReport{
		Title:       "Comparative Analysis",
		GeneratedAt: time.Now(),
		Reports:     make([]ReportComparison, len(reports)),
	}
	
	for i, report := range reports {
		comparative.Reports[i] = ReportComparison{
			Name:             report.Title,
			Period:           report.Period,
			PerformanceGrade: report.Summary.PerformanceGrade,
			TotalOperations:  report.Summary.TotalOperations,
			PeakMemory:       report.Summary.PeakMemory,
			TotalCPUTime:     report.Summary.TotalCPUTime,
		}
	}
	
	// Calculate improvements/degradations
	baseline := comparative.Reports[0]
	for i := 1; i < len(comparative.Reports); i++ {
		current := &comparative.Reports[i]
		current.OperationsChange = float64(current.TotalOperations-baseline.TotalOperations) / float64(baseline.TotalOperations) * 100
		current.MemoryChange = float64(current.PeakMemory-baseline.PeakMemory) / float64(baseline.PeakMemory) * 100
		current.CPUTimeChange = float64(current.TotalCPUTime-baseline.TotalCPUTime) / float64(baseline.TotalCPUTime) * 100
	}
	
	return comparative
}

type ComparativeReport struct {
	Title       string             `json:"title"`
	GeneratedAt time.Time          `json:"generated_at"`
	Reports     []ReportComparison `json:"reports"`
}

type ReportComparison struct {
	Name             string        `json:"name"`
	Period           ReportPeriod  `json:"period"`
	PerformanceGrade string        `json:"performance_grade"`
	TotalOperations  int64         `json:"total_operations"`
	PeakMemory       int64         `json:"peak_memory"`
	TotalCPUTime     time.Duration `json:"total_cpu_time"`
	OperationsChange float64       `json:"operations_change"`
	MemoryChange     float64       `json:"memory_change"`
	CPUTimeChange    float64       `json:"cpu_time_change"`
}