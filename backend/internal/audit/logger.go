package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuditLogger struct {
	logFile    *os.File
	buffer     []AuditEntry
	bufferSize int
	mu         sync.RWMutex
	flushChan  chan struct{}
	stopChan   chan struct{}
}

type AuditEntry struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   Details   `json:"details"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type Details struct {
	ModuleID     *uuid.UUID `json:"module_id,omitempty"`
	InstanceID   *uuid.UUID `json:"instance_id,omitempty"`
	SessionID    string     `json:"session_id,omitempty"`
	Parameters   string     `json:"parameters,omitempty"`
	ResponseSize int        `json:"response_size,omitempty"`
	Duration     int64      `json:"duration_ms,omitempty"`
}

type AuditConfig struct {
	LogPath       string        `json:"log_path"`
	BufferSize    int           `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	MaxFileSize   int64         `json:"max_file_size"`
	MaxFiles      int           `json:"max_files"`
}

func NewAuditLogger(config AuditConfig) (*AuditLogger, error) {
	if err := os.MkdirAll(filepath.Dir(config.LogPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logFile, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	logger := &AuditLogger{
		logFile:    logFile,
		buffer:     make([]AuditEntry, 0, config.BufferSize),
		bufferSize: config.BufferSize,
		flushChan:  make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}

	go logger.flushWorker(config.FlushInterval)
	return logger, nil
}

func (al *AuditLogger) Log(entry AuditEntry) {
	entry.ID = uuid.New()
	entry.Timestamp = time.Now().UTC()

	al.mu.Lock()
	al.buffer = append(al.buffer, entry)
	shouldFlush := len(al.buffer) >= al.bufferSize
	al.mu.Unlock()

	if shouldFlush {
		select {
		case al.flushChan <- struct{}{}:
		default:
		}
	}
}

func (al *AuditLogger) LogAction(userID uuid.UUID, username, action, resource string, details Details, ipAddress, userAgent string, success bool, err error) {
	entry := AuditEntry{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	al.Log(entry)
}

func (al *AuditLogger) LogLogin(userID uuid.UUID, username, ipAddress, userAgent string, success bool, err error) {
	al.LogAction(userID, username, "login", "auth", Details{}, ipAddress, userAgent, success, err)
}

func (al *AuditLogger) LogModuleAction(userID uuid.UUID, username, action string, moduleID uuid.UUID, ipAddress string, success bool, err error) {
	details := Details{ModuleID: &moduleID}
	al.LogAction(userID, username, action, "module", details, ipAddress, "", success, err)
}

func (al *AuditLogger) flushWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			al.flush()
		case <-al.flushChan:
			al.flush()
		case <-al.stopChan:
			al.flush()
			return
		}
	}
}

func (al *AuditLogger) flush() {
	al.mu.Lock()
	if len(al.buffer) == 0 {
		al.mu.Unlock()
		return
	}

	entries := make([]AuditEntry, len(al.buffer))
	copy(entries, al.buffer)
	al.buffer = al.buffer[:0]
	al.mu.Unlock()

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}

		al.logFile.Write(data)
		al.logFile.Write([]byte("\n"))
	}

	al.logFile.Sync()
}

func (al *AuditLogger) Close() error {
	close(al.stopChan)
	al.flush()
	return al.logFile.Close()
}

type AuditFilter struct {
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Action    string     `json:"action,omitempty"`
	Resource  string     `json:"resource,omitempty"`
	StartTime time.Time  `json:"start_time,omitempty"`
	EndTime   time.Time  `json:"end_time,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

func (al *AuditLogger) Query(filter AuditFilter) ([]AuditEntry, error) {
	file, err := os.Open(al.logFile.Name())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []AuditEntry
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			continue
		}

		if al.matchesFilter(entry, filter) {
			entries = append(entries, entry)
		}

		if len(entries) >= filter.Limit {
			break
		}
	}

	return entries, nil
}

func (al *AuditLogger) matchesFilter(entry AuditEntry, filter AuditFilter) bool {
	if filter.UserID != nil && entry.UserID != *filter.UserID {
		return false
	}

	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}

	if filter.Resource != "" && entry.Resource != filter.Resource {
		return false
	}

	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	if filter.Success != nil && entry.Success != *filter.Success {
		return false
	}

	return true
}