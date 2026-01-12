package output

import (
	"fmt"

	"github.com/sl/prompt-builder-agent/internal/llm"
)

// StdoutStreamRenderer renders streaming output to stdout
type StdoutStreamRenderer struct {
	formatted bool
}

// NewStdoutStreamRenderer creates a new stdout streaming renderer
func NewStdoutStreamRenderer() *StdoutStreamRenderer {
	return &StdoutStreamRenderer{
		formatted: true,
	}
}

// RenderStream implements StreamRenderer interface
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
