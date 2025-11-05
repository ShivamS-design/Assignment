package metrics

import (
	"sync"
	"sync/atomic"
	"time"
	"runtime"
)

type MetricsCollector struct {
	operationsPerSecond int64
	currentMemory       int64
	peakMemory          int64
	cpuTime             int64
	syscallCounts       map[string]*int64
	
	timeSeries    []TimeSeriesPoint
	maxDataPoints int
	mu            sync.RWMutex
	
	startTime     time.Time
	lastUpdate    time.Time
	updateTicker  *time.Ticker
	stopChan      chan struct{}
}

type TimeSeriesPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Operations  int64     `json:"operations"`
	Memory      int64     `json:"memory"`
	CPUTime     int64     `json:"cpu_time"`
	Syscalls    int64     `json:"syscalls"`
}

type RuntimeMetrics struct {
	OperationsPerSecond int64             `json:"operations_per_second"`
	CurrentMemory       int64             `json:"current_memory"`
	PeakMemory          int64             `json:"peak_memory"`
	CPUTime             time.Duration     `json:"cpu_time"`
	SyscallCounts       map[string]int64  `json:"syscall_counts"`
	Uptime              time.Duration     `json:"uptime"`
	LastUpdate          time.Time         `json:"last_update"`
}

func NewMetricsCollector(maxDataPoints int) *MetricsCollector {
	mc := &MetricsCollector{
		syscallCounts: make(map[string]*int64),
		timeSeries:    make([]TimeSeriesPoint, 0, maxDataPoints),
		maxDataPoints: maxDataPoints,
		startTime:     time.Now(),
		stopChan:      make(chan struct{}),
	}
	
	mc.updateTicker = time.NewTicker(time.Second)
	go mc.collectLoop()
	
	return mc
}

func (mc *MetricsCollector) RecordOperation() {
	atomic.AddInt64(&mc.operationsPerSecond, 1)
}

func (mc *MetricsCollector) RecordMemoryUsage(bytes int64) {
	atomic.StoreInt64(&mc.currentMemory, bytes)
	
	for {
		peak := atomic.LoadInt64(&mc.peakMemory)
		if bytes <= peak || atomic.CompareAndSwapInt64(&mc.peakMemory, peak, bytes) {
			break
		}
	}
}

func (mc *MetricsCollector) RecordCPUTime(duration time.Duration) {
	atomic.AddInt64(&mc.cpuTime, int64(duration))
}

func (mc *MetricsCollector) RecordSyscall(name string) {
	mc.mu.RLock()
	counter, exists := mc.syscallCounts[name]
	mc.mu.RUnlock()
	
	if !exists {
		mc.mu.Lock()
		if counter, exists = mc.syscallCounts[name]; !exists {
			counter = new(int64)
			mc.syscallCounts[name] = counter
		}
		mc.mu.Unlock()
	}
	
	atomic.AddInt64(counter, 1)
}

func (mc *MetricsCollector) GetRuntimeMetrics() RuntimeMetrics {
	mc.mu.RLock()
	syscallCounts := make(map[string]int64, len(mc.syscallCounts))
	for name, counter := range mc.syscallCounts {
		syscallCounts[name] = atomic.LoadInt64(counter)
	}
	mc.mu.RUnlock()
	
	return RuntimeMetrics{
		OperationsPerSecond: atomic.LoadInt64(&mc.operationsPerSecond),
		CurrentMemory:       atomic.LoadInt64(&mc.currentMemory),
		PeakMemory:          atomic.LoadInt64(&mc.peakMemory),
		CPUTime:             time.Duration(atomic.LoadInt64(&mc.cpuTime)),
		SyscallCounts:       syscallCounts,
		Uptime:              time.Since(mc.startTime),
		LastUpdate:          mc.lastUpdate,
	}
}

func (mc *MetricsCollector) GetTimeSeries() []TimeSeriesPoint {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make([]TimeSeriesPoint, len(mc.timeSeries))
	copy(result, mc.timeSeries)
	return result
}

func (mc *MetricsCollector) collectLoop() {
	defer mc.updateTicker.Stop()
	
	for {
		select {
		case <-mc.updateTicker.C:
			mc.updateTimeSeries()
		case <-mc.stopChan:
			return
		}
	}
}

func (mc *MetricsCollector) updateTimeSeries() {
	now := time.Now()
	
	var totalSyscalls int64
	mc.mu.RLock()
	for _, counter := range mc.syscallCounts {
		totalSyscalls += atomic.LoadInt64(counter)
	}
	mc.mu.RUnlock()
	
	point := TimeSeriesPoint{
		Timestamp:  now,
		Operations: atomic.LoadInt64(&mc.operationsPerSecond),
		Memory:     atomic.LoadInt64(&mc.currentMemory),
		CPUTime:    atomic.LoadInt64(&mc.cpuTime),
		Syscalls:   totalSyscalls,
	}
	
	mc.mu.Lock()
	mc.timeSeries = append(mc.timeSeries, point)
	if len(mc.timeSeries) > mc.maxDataPoints {
		mc.timeSeries = mc.timeSeries[1:]
	}
	mc.lastUpdate = now
	mc.mu.Unlock()
	
	atomic.StoreInt64(&mc.operationsPerSecond, 0)
}

func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

type SystemMetrics struct {
	GoRoutines   int           `json:"goroutines"`
	HeapAlloc    uint64        `json:"heap_alloc"`
	HeapSys      uint64        `json:"heap_sys"`
	NumGC        uint32        `json:"num_gc"`
	GCPauseTotal time.Duration `json:"gc_pause_total"`
}

func GetSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemMetrics{
		GoRoutines:   runtime.NumGoroutine(),
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		NumGC:        m.NumGC,
		GCPauseTotal: time.Duration(m.PauseTotalNs),
	}
}