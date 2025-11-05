package snapshots

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SnapshotManager struct {
	storageDir string
	registry   map[string]*SnapshotMetadata
	mu         sync.RWMutex
}

type Snapshot struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	Tags        []string          `json:"tags"`
	Version     string            `json:"version"`
	State       *ExecutionState   `json:"state"`
	Metadata    map[string]string `json:"metadata"`
	Checksum    string            `json:"checksum"`
	Compressed  bool              `json:"compressed"`
	Size        int64             `json:"size"`
}

type SnapshotMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	Tags        []string          `json:"tags"`
	Version     string            `json:"version"`
	Checksum    string            `json:"checksum"`
	Compressed  bool              `json:"compressed"`
	Size        int64             `json:"size"`
	FilePath    string            `json:"file_path"`
	Metadata    map[string]string `json:"metadata"`
}

type ExecutionState struct {
	InstructionPointer uint32            `json:"instruction_pointer"`
	StackPointer       uint32            `json:"stack_pointer"`
	Registers          map[string]uint32 `json:"registers"`
	Memory             []byte            `json:"memory"`
	CallStack          []CallFrame       `json:"call_stack"`
	Locals             []int32           `json:"locals"`
	Globals            []int32           `json:"globals"`
	ModuleID           string            `json:"module_id"`
	InstanceID         string            `json:"instance_id"`
}

type CallFrame struct {
	FunctionIndex      uint32 `json:"function_index"`
	InstructionPointer uint32 `json:"instruction_pointer"`
	LocalsStart        uint32 `json:"locals_start"`
}

func NewSnapshotManager(storageDir string) (*SnapshotManager, error) {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	sm := &SnapshotManager{
		storageDir: storageDir,
		registry:   make(map[string]*SnapshotMetadata),
	}

	if err := sm.loadRegistry(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return sm, nil
}

func (sm *SnapshotManager) CreateSnapshot(name, description string, state *ExecutionState, tags []string) (*Snapshot, error) {
	snapshot := &Snapshot{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Tags:        tags,
		Version:     "1.0",
		State:       state,
		Metadata:    make(map[string]string),
		Compressed:  true,
	}

	snapshot.Metadata["module_id"] = state.ModuleID
	snapshot.Metadata["instance_id"] = state.InstanceID
	snapshot.Metadata["memory_size"] = fmt.Sprintf("%d", len(state.Memory))

	data, err := sm.serializeSnapshot(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	hash := sha256.Sum256(data)
	snapshot.Checksum = fmt.Sprintf("%x", hash)
	snapshot.Size = int64(len(data))

	filePath := filepath.Join(sm.storageDir, snapshot.ID+".snap")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	metadata := &SnapshotMetadata{
		ID:          snapshot.ID,
		Name:        snapshot.Name,
		Description: snapshot.Description,
		CreatedAt:   snapshot.CreatedAt,
		Tags:        snapshot.Tags,
		Version:     snapshot.Version,
		Checksum:    snapshot.Checksum,
		Compressed:  snapshot.Compressed,
		Size:        snapshot.Size,
		FilePath:    filePath,
		Metadata:    snapshot.Metadata,
	}

	sm.mu.Lock()
	sm.registry[snapshot.ID] = metadata
	sm.mu.Unlock()

	if err := sm.saveRegistry(); err != nil {
		return nil, fmt.Errorf("failed to update registry: %w", err)
	}

	return snapshot, nil
}

func (sm *SnapshotManager) LoadSnapshot(id string) (*Snapshot, error) {
	sm.mu.RLock()
	metadata, exists := sm.registry[id]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	data, err := os.ReadFile(metadata.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	hash := sha256.Sum256(data)
	if fmt.Sprintf("%x", hash) != metadata.Checksum {
		return nil, fmt.Errorf("snapshot integrity check failed")
	}

	snapshot, err := sm.deserializeSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}

	return snapshot, nil
}

func (sm *SnapshotManager) ListSnapshots() []*SnapshotMetadata {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshots := make([]*SnapshotMetadata, 0, len(sm.registry))
	for _, metadata := range sm.registry {
		snapshots = append(snapshots, metadata)
	}

	return snapshots
}

func (sm *SnapshotManager) DeleteSnapshot(id string) error {
	sm.mu.Lock()
	metadata, exists := sm.registry[id]
	if exists {
		delete(sm.registry, id)
	}
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("snapshot not found: %s", id)
	}

	if err := os.Remove(metadata.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove snapshot file: %w", err)
	}

	return sm.saveRegistry()
}

func (sm *SnapshotManager) RestoreSnapshot(id string) (*ExecutionState, error) {
	snapshot, err := sm.LoadSnapshot(id)
	if err != nil {
		return nil, err
	}

	if snapshot.Metadata["incremental"] == "true" {
		baseID := snapshot.Metadata["base_snapshot"]
		baseState, err := sm.RestoreSnapshot(baseID)
		if err != nil {
			return nil, fmt.Errorf("failed to restore base snapshot: %w", err)
		}

		return sm.applyStateDiff(baseState, snapshot.State), nil
	}

	return snapshot.State, nil
}

func (sm *SnapshotManager) serializeSnapshot(snapshot *Snapshot) ([]byte, error) {
	jsonData, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}

	if !snapshot.Compressed {
		return jsonData, nil
	}

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(jsonData); err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (sm *SnapshotManager) deserializeSnapshot(data []byte) (*Snapshot, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err == nil {
		defer gzReader.Close()
		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			return nil, err
		}
		data = decompressed
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (sm *SnapshotManager) applyStateDiff(base, diff *ExecutionState) *ExecutionState {
	result := &ExecutionState{
		InstructionPointer: diff.InstructionPointer,
		StackPointer:       diff.StackPointer,
		Registers:          make(map[string]uint32),
		CallStack:          diff.CallStack,
		ModuleID:           diff.ModuleID,
		InstanceID:         diff.InstanceID,
	}

	for name, value := range base.Registers {
		result.Registers[name] = value
	}
	for name, value := range diff.Registers {
		result.Registers[name] = value
	}

	if len(diff.Memory) > 0 {
		result.Memory = make([]byte, len(diff.Memory))
		copy(result.Memory, diff.Memory)
	} else {
		result.Memory = make([]byte, len(base.Memory))
		copy(result.Memory, base.Memory)
	}

	if len(diff.Locals) > 0 {
		result.Locals = make([]int32, len(diff.Locals))
		copy(result.Locals, diff.Locals)
	} else {
		result.Locals = make([]int32, len(base.Locals))
		copy(result.Locals, base.Locals)
	}

	return result
}

func (sm *SnapshotManager) loadRegistry() error {
	registryPath := filepath.Join(sm.storageDir, "registry.json")
	data, err := os.ReadFile(registryPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var registry map[string]*SnapshotMetadata
	if err := json.Unmarshal(data, &registry); err != nil {
		return err
	}

	sm.registry = registry
	return nil
}

func (sm *SnapshotManager) saveRegistry() error {
	registryPath := filepath.Join(sm.storageDir, "registry.json")
	data, err := json.MarshalIndent(sm.registry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(registryPath, data, 0644)
}