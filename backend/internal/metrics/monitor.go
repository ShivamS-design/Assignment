package metrics

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Monitor struct {
	collector   *MetricsCollector
	thresholds  map[string]Threshold
	alerts      []Alert
	subscribers map[*websocket.Conn]bool
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

type Threshold struct {
	MetricName string  `json:"metric_name"`
	MaxValue   float64 `json:"max_value"`
	MinValue   float64 `json:"min_value"`
	Enabled    bool    `json:"enabled"`
}

type Alert struct {
	ID        string    `json:"id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Active    bool      `json:"active"`
}

type MetricsUpdate struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"timestamp"`
}

func NewMonitor(collector *MetricsCollector) *Monitor {
	m := &Monitor{
		collector:   collector,
		thresholds:  make(map[string]Threshold),
		alerts:      make([]Alert, 0),
		subscribers: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	
	m.setDefaultThresholds()
	go m.monitorLoop()
	
	return m
}

func (m *Monitor) setDefaultThresholds() {
	m.thresholds["memory"] = Threshold{
		MetricName: "memory",
		MaxValue:   1024 * 1024 * 100, // 100MB
		Enabled:    true,
	}
	
	m.thresholds["operations"] = Threshold{
		MetricName: "operations",
		MaxValue:   10000, // 10k ops/sec
		Enabled:    true,
	}
	
	m.thresholds["cpu_time"] = Threshold{
		MetricName: "cpu_time",
		MaxValue:   float64(30 * time.Second), // 30 seconds
		Enabled:    true,
	}
}

func (m *Monitor) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	
	m.mu.Lock()
	m.subscribers[conn] = true
	m.mu.Unlock()
	
	defer func() {
		m.mu.Lock()
		delete(m.subscribers, conn)
		m.mu.Unlock()
	}()
	
	// Send initial metrics
	metrics := m.collector.GetRuntimeMetrics()
	m.sendToConnection(conn, MetricsUpdate{
		Type: "initial",
		Data: metrics,
		Time: time.Now(),
	})
	
	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		metrics := m.collector.GetRuntimeMetrics()
		m.checkThresholds(metrics)
		m.broadcastMetrics(metrics)
	}
}

func (m *Monitor) checkThresholds(metrics RuntimeMetrics) {
	now := time.Now()
	
	// Check memory threshold
	if threshold, exists := m.thresholds["memory"]; exists && threshold.Enabled {
		if float64(metrics.CurrentMemory) > threshold.MaxValue {
			m.createAlert("memory", float64(metrics.CurrentMemory), threshold.MaxValue, "warning", now)
		}
	}
	
	// Check operations threshold
	if threshold, exists := m.thresholds["operations"]; exists && threshold.Enabled {
		if float64(metrics.OperationsPerSecond) > threshold.MaxValue {
			m.createAlert("operations", float64(metrics.OperationsPerSecond), threshold.MaxValue, "info", now)
		}
	}
	
	// Check CPU time threshold
	if threshold, exists := m.thresholds["cpu_time"]; exists && threshold.Enabled {
		if float64(metrics.CPUTime) > threshold.MaxValue {
			m.createAlert("cpu_time", float64(metrics.CPUTime), threshold.MaxValue, "warning", now)
		}
	}
}

func (m *Monitor) createAlert(metric string, value, threshold float64, severity string, timestamp time.Time) {
	alert := Alert{
		ID:        generateAlertID(),
		Metric:    metric,
		Value:     value,
		Threshold: threshold,
		Severity:  severity,
		Message:   formatAlertMessage(metric, value, threshold),
		Timestamp: timestamp,
		Active:    true,
	}
	
	m.mu.Lock()
	m.alerts = append(m.alerts, alert)
	if len(m.alerts) > 100 {
		m.alerts = m.alerts[1:]
	}
	m.mu.Unlock()
	
	m.broadcastAlert(alert)
}

func (m *Monitor) broadcastMetrics(metrics RuntimeMetrics) {
	update := MetricsUpdate{
		Type: "metrics",
		Data: metrics,
		Time: time.Now(),
	}
	
	m.mu.RLock()
	for conn := range m.subscribers {
		go m.sendToConnection(conn, update)
	}
	m.mu.RUnlock()
}

func (m *Monitor) broadcastAlert(alert Alert) {
	update := MetricsUpdate{
		Type: "alert",
		Data: alert,
		Time: time.Now(),
	}
	
	m.mu.RLock()
	for conn := range m.subscribers {
		go m.sendToConnection(conn, update)
	}
	m.mu.RUnlock()
}

func (m *Monitor) sendToConnection(conn *websocket.Conn, update MetricsUpdate) {
	if err := conn.WriteJSON(update); err != nil {
		m.mu.Lock()
		delete(m.subscribers, conn)
		m.mu.Unlock()
		conn.Close()
	}
}

func (m *Monitor) GetAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]Alert, len(m.alerts))
	copy(result, m.alerts)
	return result
}

func (m *Monitor) SetThreshold(name string, threshold Threshold) {
	m.mu.Lock()
	m.thresholds[name] = threshold
	m.mu.Unlock()
}

func (m *Monitor) GetThresholds() map[string]Threshold {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]Threshold)
	for k, v := range m.thresholds {
		result[k] = v
	}
	return result
}

func (m *Monitor) DetectPerformanceDegradation() *PerformanceDegradation {
	timeSeries := m.collector.GetTimeSeries()
	if len(timeSeries) < 10 {
		return nil
	}
	
	recent := timeSeries[len(timeSeries)-5:]
	baseline := timeSeries[len(timeSeries)-10 : len(timeSeries)-5]
	
	recentAvg := calculateAverage(recent)
	baselineAvg := calculateAverage(baseline)
	
	degradation := &PerformanceDegradation{
		Detected:    false,
		Timestamp:   time.Now(),
		Metrics:     make(map[string]DegradationMetric),
	}
	
	// Check operations degradation
	if recentAvg.Operations < baselineAvg.Operations*0.8 {
		degradation.Detected = true
		degradation.Metrics["operations"] = DegradationMetric{
			Current:  recentAvg.Operations,
			Baseline: baselineAvg.Operations,
			Change:   (recentAvg.Operations - baselineAvg.Operations) / baselineAvg.Operations,
		}
	}
	
	// Check memory increase
	if recentAvg.Memory > baselineAvg.Memory*1.2 {
		degradation.Detected = true
		degradation.Metrics["memory"] = DegradationMetric{
			Current:  float64(recentAvg.Memory),
			Baseline: float64(baselineAvg.Memory),
			Change:   float64(recentAvg.Memory-baselineAvg.Memory) / float64(baselineAvg.Memory),
		}
	}
	
	if degradation.Detected {
		return degradation
	}
	return nil
}

type PerformanceDegradation struct {
	Detected  bool                         `json:"detected"`
	Timestamp time.Time                    `json:"timestamp"`
	Metrics   map[string]DegradationMetric `json:"metrics"`
}

type DegradationMetric struct {
	Current  float64 `json:"current"`
	Baseline float64 `json:"baseline"`
	Change   float64 `json:"change_percent"`
}

func calculateAverage(points []TimeSeriesPoint) TimeSeriesPoint {
	if len(points) == 0 {
		return TimeSeriesPoint{}
	}
	
	var totalOps, totalMem, totalCPU, totalSyscalls int64
	
	for _, point := range points {
		totalOps += point.Operations
		totalMem += point.Memory
		totalCPU += point.CPUTime
		totalSyscalls += point.Syscalls
	}
	
	count := int64(len(points))
	return TimeSeriesPoint{
		Operations: totalOps / count,
		Memory:     totalMem / count,
		CPUTime:    totalCPU / count,
		Syscalls:   totalSyscalls / count,
	}
}

func generateAlertID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func formatAlertMessage(metric string, value, threshold float64) string {
	switch metric {
	case "memory":
		return fmt.Sprintf("Memory usage %.2f MB exceeds threshold %.2f MB", value/1024/1024, threshold/1024/1024)
	case "operations":
		return fmt.Sprintf("Operations per second %.0f exceeds threshold %.0f", value, threshold)
	case "cpu_time":
		return fmt.Sprintf("CPU time %.2fs exceeds threshold %.2fs", value/float64(time.Second), threshold/float64(time.Second))
	default:
		return fmt.Sprintf("Metric %s value %.2f exceeds threshold %.2f", metric, value, threshold)
	}
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}