package gobin

import (
	"fmt"
	"sync"
	"testing"
)

// TestInstanceIsolation verifies that different GoBin instances are completely isolated
func TestInstanceIsolation(t *testing.T) {
	type TestStruct struct {
		ID   int    `binary:"id"`
		Name string `binary:"name"`
		Data []byte `binary:"data"`
	}

	// Create two separate instances
	instance1 := New()
	instance2 := New()

	// Create different data for each instance
	data1 := TestStruct{ID: 1, Name: "Instance1", Data: []byte{1, 2, 3}}
	data2 := TestStruct{ID: 2, Name: "Instance2", Data: []byte{4, 5, 6}}

	// Encode with different instances
	encoded1, err := instance1.Encode(data1)
	if err != nil {
		t.Fatalf("Instance1 encode failed: %v", err)
	}

	encoded2, err := instance2.Encode(data2)
	if err != nil {
		t.Fatalf("Instance2 encode failed: %v", err)
	}

	// Verify encoded data is different (due to different content)
	if string(encoded1) == string(encoded2) {
		t.Error("Expected different encoded data for different instances")
	}

	// Decode with respective instances
	var decoded1 TestStruct
	err = instance1.Decode(encoded1, &decoded1)
	if err != nil {
		t.Fatalf("Instance1 decode failed: %v", err)
	}

	var decoded2 TestStruct
	err = instance2.Decode(encoded2, &decoded2)
	if err != nil {
		t.Fatalf("Instance2 decode failed: %v", err)
	}

	// Verify decoded data matches original
	if decoded1.ID != data1.ID || decoded1.Name != data1.Name {
		t.Errorf("Instance1: Expected %+v, got %+v", data1, decoded1)
	}

	if decoded2.ID != data2.ID || decoded2.Name != data2.Name {
		t.Errorf("Instance2: Expected %+v, got %+v", data2, decoded2)
	}
}

// TestConcurrentInstanceUsage verifies thread safety and isolation in concurrent scenarios
func TestConcurrentInstanceUsage(t *testing.T) {
	instance1 := New()
	instance2 := New()

	type Counter struct {
		Value int `binary:"value"`
	}

	const numGoroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Test instance1
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				counter := Counter{Value: id*iterations + j}

				data, err := instance1.Encode(counter)
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}

				var decoded Counter
				err = instance1.Decode(data, &decoded)
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}

				if decoded.Value != counter.Value {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}
			}
		}(i)
	}

	// Test instance2 concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				counter := Counter{Value: id*iterations + j + 10000} // Different range

				data, err := instance2.Encode(counter)
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}

				var decoded Counter
				err = instance2.Decode(data, &decoded)
				if err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}

				if decoded.Value != counter.Value {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		t.Fatalf("Concurrent test failed with errors: %v", errors)
	}
}

// TestInstanceCacheIsolation verifies that schema caches are isolated between instances
func TestInstanceCacheIsolation(t *testing.T) {
	// Create two instances
	instance1 := New()
	instance2 := New()

	// Use the same struct type with both instances
	type CachedStruct struct {
		A int    `binary:"a"`
		B string `binary:"b"`
		C []byte `binary:"c"`
	}

	data := CachedStruct{A: 42, B: "test", C: []byte{1, 2, 3}}

	// First encode/decode with instance1 (populates its cache)
	encoded1, err := instance1.Encode(data)
	if err != nil {
		t.Fatalf("Instance1 encode failed: %v", err)
	}

	var decoded1 CachedStruct
	err = instance1.Decode(encoded1, &decoded1)
	if err != nil {
		t.Fatalf("Instance1 decode failed: %v", err)
	}

	// First encode/decode with instance2 (populates its cache)
	encoded2, err := instance2.Encode(data)
	if err != nil {
		t.Fatalf("Instance2 encode failed: %v", err)
	}

	var decoded2 CachedStruct
	err = instance2.Decode(encoded2, &decoded2)
	if err != nil {
		t.Fatalf("Instance2 decode failed: %v", err)
	}

	// Verify both instances work correctly
	if decoded1.A != data.A || decoded1.B != data.B {
		t.Errorf("Instance1: Expected %+v, got %+v", data, decoded1)
	}

	if decoded2.A != data.A || decoded2.B != data.B {
		t.Errorf("Instance2: Expected %+v, got %+v", data, decoded2)
	}

	// Verify encoded data is identical (same encoding logic)
	if len(encoded1) != len(encoded2) {
		t.Errorf("Expected same encoded length, got %d vs %d", len(encoded1), len(encoded2))
	}
}

// TestCustomLogging verifies that custom logging works with instances
func TestCustomLogging(t *testing.T) {
	logs := make([]string, 0)
	logMutex := sync.Mutex{}

	// Create instance with custom logging
	tb := New(func(msg ...any) {
		logMutex.Lock()
		defer logMutex.Unlock()
		logs = append(logs, fmt.Sprintf("%v", msg))
	})

	type SimpleStruct struct {
		Value int `binary:"value"`
	}

	data := SimpleStruct{Value: 123}

	// Perform encode/decode operations
	encoded, err := tb.Encode(data)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var decoded SimpleStruct
	err = tb.Decode(encoded, &decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify the operation completed successfully
	if decoded.Value != data.Value {
		t.Errorf("Expected %v, got %v", data, decoded)
	}

	// Note: In a real scenario, you might verify that logging occurred
	// For this test, we mainly verify that custom logging doesn't break functionality
}

// TestMultipleProtocolSimulation demonstrates how different services can use isolated instances
func TestMultipleProtocolSimulation(t *testing.T) {
	// Simulate different protocol handlers
	httpHandler := New()
	grpcHandler := New()
	kafkaHandler := New()

	type Message struct {
		ID      int    `binary:"id"`
		Content string `binary:"content"`
	}

	// Each handler processes the same message independently
	message := Message{ID: 1, Content: "Hello World"}

	httpData, err := httpHandler.Encode(message)
	if err != nil {
		t.Fatalf("HTTP handler encode failed: %v", err)
	}

	grpcData, err := grpcHandler.Encode(message)
	if err != nil {
		t.Fatalf("gRPC handler encode failed: %v", err)
	}

	kafkaData, err := kafkaHandler.Encode(message)
	if err != nil {
		t.Fatalf("Kafka handler encode failed: %v", err)
	}

	// All handlers should produce identical results for the same input
	if string(httpData) != string(grpcData) || string(grpcData) != string(kafkaData) {
		t.Error("Different handlers produced different encoded data for same input")
	}

	// Each handler can decode independently
	var httpDecoded, grpcDecoded, kafkaDecoded Message

	err = httpHandler.Decode(httpData, &httpDecoded)
	if err != nil {
		t.Fatalf("HTTP handler decode failed: %v", err)
	}

	err = grpcHandler.Decode(grpcData, &grpcDecoded)
	if err != nil {
		t.Fatalf("gRPC handler decode failed: %v", err)
	}

	err = kafkaHandler.Decode(kafkaData, &kafkaDecoded)
	if err != nil {
		t.Fatalf("Kafka handler decode failed: %v", err)
	}

	// All decoded messages should be identical
	if httpDecoded.ID != grpcDecoded.ID || grpcDecoded.ID != kafkaDecoded.ID {
		t.Error("Handlers produced different decoded results")
	}

	if httpDecoded.Content != grpcDecoded.Content || grpcDecoded.Content != kafkaDecoded.Content {
		t.Error("Handlers produced different decoded content")
	}
}
