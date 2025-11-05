package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"wasm-as-os/internal/metrics"
)

func TestMetricsCollector(t *testing.T) {
	collector := metrics.NewCollector()
	
	// Test operation counting
	collector.IncrementOperations("test-module")
	collector.IncrementOperations("test-module")
	
	stats := collector.GetModuleStats("test-module")
	assert.Equal(t, uint64(2), stats.Operations)
	
	// Test memory tracking
	collector.UpdateMemoryUsage("test-module", 1024)
	stats = collector.GetModuleStats("test-module")
	assert.Equal(t, uint64(1024), stats.MemoryUsage)
}

func TestMetricsMonitor(t *testing.T) {
	monitor := metrics.NewMonitor()
	
	// Test alert configuration
	alert := &metrics.Alert{
		Name: "high-memory",
		Metric: "memory_usage",
		Threshold: 1000,
		Operator: "gt",
	}
	
	monitor.AddAlert(alert)
	
	// Simulate metric update that triggers alert
	triggered := monitor.CheckAlerts("test-module", map[string]float64{
		"memory_usage": 1500,
	})
	
	assert.True(t, triggered)
}

func TestMetricsReporter(t *testing.T) {
	reporter := metrics.NewReporter()
	
	// Add sample data
	reporter.AddDataPoint("test-module", "operations", 100, time.Now())
	reporter.AddDataPoint("test-module", "operations", 150, time.Now().Add(time.Minute))
	
	report := reporter.GenerateReport("test-module", time.Now().Add(-time.Hour), time.Now())
	
	assert.Equal(t, "test-module", report.ModuleID)
	assert.Len(t, report.Metrics, 1)
	assert.Equal(t, 2, len(report.Metrics["operations"]))
}

func BenchmarkMetricsCollection(b *testing.B) {
	collector := metrics.NewCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncrementOperations("bench-module")
	}
}