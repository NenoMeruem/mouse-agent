package llm

import "context"

// Request represents an LLM request
type Request struct {
	Prompt string // The prompt to send
	Model  string // Model name/ID
}

// Chunk represents a streaming response chunk
type Chunk struct {
	Text string // The text content of this chunk
	Err  error  // Error if occurred (mutually exclusive with Text)
	Done bool   // True when stream is complete
}

// Client is the interface all LLM providers must implement
type Client interface {
	// Name returns the name of the LLM provider
	Name() string

	// Stream returns a channel that streams response chunks
	// The channel will be closed when streaming is complete
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
