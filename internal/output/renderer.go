package output

import (
	"github.com/sl/prompt-builder-agent/internal/llm"
)

// Renderer interface for streaming output
type StreamRenderer interface {
	// RenderStream renders chunks from a streaming channel
	// Blocks until the channel is closed or context is cancelled
	RenderStream(ch <-chan llm.Chunk) error
}

// Factory creates renderers based on configuration
type Factory struct {
	outputType string
}

// NewFactory creates a new renderer factory
func NewFactory(outputType string) *Factory {
	return &Factory{
		outputType: outputType,
	}
}

// CreateRenderer creates a renderer based on output type
func (f *Factory) CreateRenderer() StreamRenderer {
	switch f.outputType {
	case "tui":
		return NewTUIRenderer()
	case "stdout":
		fallthrough
	default:
		return NewStdoutStreamRenderer()
	}
}
