package output

import (
	"fmt"

	"github.com/meruem/promptly/internal/llm"
)

// StdoutStreamRenderer renders streaming output to stdout with decorators.
type StdoutStreamRenderer struct{}

// NewStdoutStreamRenderer creates a new stdout streaming renderer.
func NewStdoutStreamRenderer() *StdoutStreamRenderer {
	return &StdoutStreamRenderer{}
}

// RenderStream implements StreamRenderer interface.
func (r *StdoutStreamRenderer) RenderStream(ch <-chan llm.Chunk) error {
	first := true
	for chunk := range ch {
		if chunk.Err != nil {
			fmt.Printf("❌ Error: %v\n", chunk.Err)
			return chunk.Err
		}
		if chunk.Done {
			fmt.Println("\n✓ Done")
			return nil
		}
		if chunk.Text != "" {
			if first {
				fmt.Print("🤖 AI: ")
				first = false
			}
			fmt.Print(chunk.Text)
		}
	}
	return nil
}

// RawStdoutStreamRenderer outputs only the AI text — no prefix, no status lines.
// Used when --raw is set (e.g. Tauri sidecar mode).
type RawStdoutStreamRenderer struct{}

func NewRawStdoutStreamRenderer() *RawStdoutStreamRenderer {
	return &RawStdoutStreamRenderer{}
}

func (r *RawStdoutStreamRenderer) RenderStream(ch <-chan llm.Chunk) error {
	for chunk := range ch {
		if chunk.Err != nil {
			return chunk.Err
		}
		if chunk.Done {
			return nil
		}
		if chunk.Text != "" {
			fmt.Print(chunk.Text)
		}
	}
	return nil
}
