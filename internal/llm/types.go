// Package llm provides interfaces for LLM streaming communication.
package llm

import "context"

// Request represents a request to send to an LLM provider.
type Request struct {
	Prompt string // The user prompt/message to send to the LLM
	Model  string // The model name/ID to use (e.g., "gpt-4-mini")
}

// Chunk represents a single chunk of streaming response data from an LLM.
// Either Text contains data OR Err contains an error (but not both).
type Chunk struct {
	Text string // The text content of this chunk (empty if Err is set)
	Err  error  // Error encountered during streaming (nil if Text is set)
	Done bool   // True when the response stream is complete
}

// Client is the interface that all LLM providers must implement.
// Implementations handle provider-specific API communication and streaming.
type Client interface {
	// Name returns the name/identifier of this LLM provider (e.g., "openai", "gemini").
	Name() string

	// Stream sends a request to the LLM and returns a channel that streams response chunks.
	// The returned channel will be closed when streaming completes or an error occurs.
	// The provided context can be used to cancel the request.
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
