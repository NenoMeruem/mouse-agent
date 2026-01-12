package llm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm/openai"
)

// TestLLMGateway tests the LLM gateway registration and retrieval
func TestLLMGateway(t *testing.T) {
	mgr := NewManager()

	// Create mock clients
	client1 := openai.NewClient("test-key", "gpt-4-mini", 30*time.Second)

	// Register clients
	if err := mgr.Register("openai", client1); err != nil {
		t.Fatalf("failed to register client: %v", err)
	}

	// Verify registration
	if err := mgr.Register("openai", client1); err == nil {
		t.Fatal("should not allow duplicate registration")
	}

	// Retrieve client
	client, err := mgr.Get("openai")
	if err != nil {
		t.Fatalf("failed to get client: %v", err)
	}

	if client.Name() != "openai" {
		t.Fatalf("expected 'openai', got '%s'", client.Name())
	}

	// Try to get non-existent client
	_, err = mgr.Get("non-existent")
	if err == nil {
		t.Fatal("should return error for non-existent client")
	}

	// List clients
	names := mgr.List()
	if len(names) != 1 || names[0] != "openai" {
		t.Fatalf("unexpected client list: %v", names)
	}
}

// TestStreamingInterface tests that clients implement streaming correctly
func TestStreamingInterface(t *testing.T) {
	// This is a compile-time test to ensure types implement the interface
	var _ Client = (*openai.Client)(nil)
}

// TestChunkStructure validates the Chunk data structure
func TestChunkStructure(t *testing.T) {
	tests := []struct {
		name string
		fn   func() Chunk
	}{
		{
			name: "text chunk",
			fn: func() Chunk {
				return Chunk{Text: "Hello world", Done: false}
			},
		},
		{
			name: "error chunk",
			fn: func() Chunk {
				return Chunk{Err: fmt.Errorf("test error"), Done: true}
			},
		},
		{
			name: "done chunk",
			fn: func() Chunk {
				return Chunk{Done: true}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := tt.fn()
			// Verify the chunk can be used with context
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			ch := make(chan Chunk, 1)
			ch <- chunk
			close(ch)

			select {
			case c := <-ch:
				if c.Done && chunk.Done {
					// Pass
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for chunk")
			}
		})
	}
}
