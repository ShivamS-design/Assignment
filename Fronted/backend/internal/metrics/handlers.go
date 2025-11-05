package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type MetricsHandlers struct {
	collector *MetricsCollector
	monitor   *Monitor
	reporter  *Reporter
}

func NewMetricsHandlers(collector *MetricsCollector, monitor *Monitor, reporter *Reporter) *MetricsHandlers {
	return &MetricsHandlers{
		collector: collector,
		monitor:   monitor,
		reporter:  reporter,
	}
}

func (h *MetricsHandlers) GetRuntimeMetrics(c *gin.Context) {
	metrics := h.collector.GetRuntimeMetrics()
	systemMetrics := GetSystemMetrics()
	
	response := gin.H{
		"runtime": metrics,
		"system":  systemMetrics,
	}
	
	c.JSON(http.StatusOK, response)
}

func (h *MetricsHandlers) GetTimeSeries(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}
	
	timeSeries := h.collector.GetTimeSeries()
	if len(timeSeries) > limit {
		timeSeries = timeSeries[len(timeSeries)-limit:]
	}
	
	c.JSON(http.StatusOK, gin.H{"data": timeSeries})
}

func (h *MetricsHandlers) GetAlerts(c *gin.Context) {
	alerts := h.monitor.GetAlerts()
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *MetricsHandlers) SetThreshold(c *gin.Context) {
	var threshold Threshold
	if err := c.ShouldBindJSON(&threshold); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	h.monitor.SetThreshold(threshold.MetricName, threshold)
	c.JSON(http.StatusOK, gin.H{"message": "threshold updated"})
}

func (h *MetricsHandlers) GetThresholds(c *gin.Context) {
	thresholds := h.monitor.GetThresholds()
	c.JSON(http.StatusOK, gin.H{"thresholds": thresholds})
}

func (h *MetricsHandlers) GenerateReport(c *gin.Context) {
	templateName := c.DefaultQuery("template", "performance")
	
	startStr := c.Query("start")
	endStr := c.Query("end")
	
	var start, end time.Time
	var err error
	
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format"})
			return
		}
	} else {
		start = time.Now().Add(-1 * time.Hour)
	}
	
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format"})
			return
		}
	} else {
		end = time.Now()
	}
	
	period := ReportPeriod{Start: start, End: end}
	report, err := h.reporter.GenerateReport(templateName, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, report)
}

func (h *MetricsHandlers) ExportReport(c *gin.Context) {
	templateName := c.DefaultQuery("template", "performance")
	format := c.DefaultQuery("format", "json")
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	
	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}
	
	period := ReportPeriod{Start: start, End: end}
	report, err := h.reporter.GenerateReport(templateName, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	var data []byte
	var contentType string
	var filename string
	
	switch format {
	case "json":
		data, err = h.reporter.ExportJSON(report)
		contentType = "application/json"
		filename = "metrics-report.json"
	case "csv":
		data, err = h.reporter.ExportCSV(report)
		contentType = "text/csv"
		filename = "metrics-report.csv"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
		return
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

func (h *MetricsHandlers) GetTrendAnalysis(c *gin.Context) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	
	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	
	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}
	
	period := ReportPeriod{Start: start, End: end}
	trends := h.reporter.AnalyzeTrends(period)
	
	c.JSON(http.StatusOK, gin.H{"trends": trends})
}

func (h *MetricsHandlers) DetectPerformanceDegradation(c *gin.Context) {
	degradation := h.monitor.DetectPerformanceDegradation()
	
	if degradation == nil {
		c.JSON(http.StatusOK, gin.H{
			"detected": false,
			"message":  "No performance degradation detected",
		})
		return
	}
	
	c.JSON(http.StatusOK, degradation)
}

func (h *MetricsHandlers) HandleWebSocket(c *gin.Context) {
	h.monitor.HandleWebSocket(c.Writer, c.Request)
}