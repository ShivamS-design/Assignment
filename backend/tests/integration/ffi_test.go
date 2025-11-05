package integration

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
#cgo LDFLAGS: -L../../rust-engine/target/release -lwasm_engine
#include <stdint.h>

extern int32_t wasm_engine_init();
extern int32_t wasm_engine_load_module(const char* module_id, const uint8_t* wasm_bytes, uint32_t length);
extern int32_t wasm_engine_execute_function(const char* module_id, const char* function_name, const int32_t* args, uint32_t arg_count, int32_t* result);
extern int32_t wasm_engine_get_memory(const char* module_id, uint32_t offset, uint32_t length, uint8_t* buffer);
extern void wasm_engine_cleanup();
*/
import "C"

func TestFFIInitialization(t *testing.T) {
	result := C.wasm_engine_init()
	assert.Equal(t, int32(0), int32(result))
	
	defer C.wasm_engine_cleanup()
}

func TestFFIModuleLoading(t *testing.T) {
	C.wasm_engine_init()
	defer C.wasm_engine_cleanup()
	
	// Simple WASM module with add function
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // WASM header
		0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f, // Type section
		0x03, 0x02, 0x01, 0x00, // Function section
		0x07, 0x07, 0x01, 0x03, 0x61, 0x64, 0x64, 0x00, 0x00, // Export section
		0x0a, 0x09, 0x01, 0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a, 0x0b, // Code section
	}
	
	moduleID := C.CString("test_module")
	defer C.free(unsafe.Pointer(moduleID))
	
	result := C.wasm_engine_load_module(
		moduleID,
		(*C.uint8_t)(unsafe.Pointer(&wasmBytes[0])),
		C.uint32_t(len(wasmBytes)),
	)
	
	assert.Equal(t, int32(0), int32(result))
}

func TestFFIFunctionExecution(t *testing.T) {
	C.wasm_engine_init()
	defer C.wasm_engine_cleanup()
	
	// Load test module first
	wasmBytes := getTestAddModule()
	moduleID := C.CString("add_module")
	defer C.free(unsafe.Pointer(moduleID))
	
	loadResult := C.wasm_engine_load_module(
		moduleID,
		(*C.uint8_t)(unsafe.Pointer(&wasmBytes[0])),
		C.uint32_t(len(wasmBytes)),
	)
	require.Equal(t, int32(0), int32(loadResult))
	
	// Execute add function
	functionName := C.CString("add")
	defer C.free(unsafe.Pointer(functionName))
	
	args := []C.int32_t{5, 3}
	var result C.int32_t
	
	execResult := C.wasm_engine_execute_function(
		moduleID,
		functionName,
		&args[0],
		2,
		&result,
	)
	
	assert.Equal(t, int32(0), int32(execResult))
	assert.Equal(t, int32(8), int32(result))
}

func TestFFIMemoryAccess(t *testing.T) {
	C.wasm_engine_init()
	defer C.wasm_engine_cleanup()
	
	// Load module with memory
	wasmBytes := getTestMemoryModule()
	moduleID := C.CString("memory_module")
	defer C.free(unsafe.Pointer(moduleID))
	
	loadResult := C.wasm_engine_load_module(
		moduleID,
		(*C.uint8_t)(unsafe.Pointer(&wasmBytes[0])),
		C.uint32_t(len(wasmBytes)),
	)
	require.Equal(t, int32(0), int32(loadResult))
	
	// Read memory
	buffer := make([]byte, 4)
	result := C.wasm_engine_get_memory(
		moduleID,
		0,
		4,
		(*C.uint8_t)(unsafe.Pointer(&buffer[0])),
	)
	
	assert.Equal(t, int32(0), int32(result))
}

func TestFFIErrorHandling(t *testing.T) {
	C.wasm_engine_init()
	defer C.wasm_engine_cleanup()
	
	// Try to load invalid WASM
	invalidWasm := []byte{0x00, 0x00, 0x00, 0x00}
	moduleID := C.CString("invalid_module")
	defer C.free(unsafe.Pointer(moduleID))
	
	result := C.wasm_engine_load_module(
		moduleID,
		(*C.uint8_t)(unsafe.Pointer(&invalidWasm[0])),
		C.uint32_t(len(invalidWasm)),
	)
	
	assert.NotEqual(t, int32(0), int32(result))
}

func BenchmarkFFIFunctionCall(b *testing.B) {
	C.wasm_engine_init()
	defer C.wasm_engine_cleanup()
	
	wasmBytes := getTestAddModule()
	moduleID := C.CString("bench_module")
	defer C.free(unsafe.Pointer(moduleID))
	
	C.wasm_engine_load_module(
		moduleID,
		(*C.uint8_t)(unsafe.Pointer(&wasmBytes[0])),
		C.uint32_t(len(wasmBytes)),
	)
	
	functionName := C.CString("add")
	defer C.free(unsafe.Pointer(functionName))
	
	args := []C.int32_t{5, 3}
	var result C.int32_t
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		C.wasm_engine_execute_function(moduleID, functionName, &args[0], 2, &result)
	}
}

func getTestAddModule() []byte {
	return []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f,
		0x03, 0x02, 0x01, 0x00,
		0x07, 0x07, 0x01, 0x03, 0x61, 0x64, 0x64, 0x00, 0x00,
		0x0a, 0x09, 0x01, 0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a, 0x0b,
	}
}

func getTestMemoryModule() []byte {
	return []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		0x05, 0x03, 0x01, 0x00, 0x01, // Memory section: 1 page
	}
}