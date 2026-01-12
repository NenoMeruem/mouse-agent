package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm"
	"github.com/sl/prompt-builder-agent/internal/llm/gemini"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ GEMINI_API_KEY not set")
	}

	fmt.Println("🚀 Gemini API Demo")
	fmt.Println("=====================================")
	fmt.Println()

	// Create Gemini client
	client := gemini.NewClient(apiKey, "gemini-2.5-flash-lite", 30*time.Second)

	fmt.Printf("✅ Client created: %s\n", client.Name())
	fmt.Println()

	// Demo 1: Simple prompt
	runDemo("Demo 1: Explain AI", "What is Artificial Intelligence? Explain in 2 sentences.", client)

	fmt.Println()
	fmt.Println("-------------------------------------")
	fmt.Println()

	// Demo 2: Code generation
	runDemo("Demo 2: Code Generation", "Write a simple Hello World in Go", client)

	fmt.Println()
	fmt.Println("-------------------------------------")
	fmt.Println()

	// Demo 3: Problem solving
	runDemo("Demo 3: Problem Solving", "What are the top 3 best practices for API design?", client)

	fmt.Println()
	fmt.Println("=====================================")
	fmt.Println("✅ Demo complete!")
}

func runDemo(title string, prompt string, client llm.Client) {
	fmt.Printf("📝 %s\n", title)
	fmt.Printf("💬 Prompt: %s\n", prompt)
	fmt.Println()
	fmt.Println("📤 Response:")
	fmt.Println("─────────────────────────────────────")

	ctx := context.Background()
	req := llm.Request{
		Prompt: prompt,
		Model:  "gemini-2.5-flash-lite",
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	chunksReceived := 0
	totalText := ""

	for chunk := range ch {
		if chunk.Err != nil {
			fmt.Printf("❌ Error: %v\n", chunk.Err)
			return
		}

		if chunk.Text != "" {
			fmt.Print(chunk.Text)
			totalText += chunk.Text
		}

		if chunk.Done {
			chunksReceived++
		}
	}

	fmt.Println()
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("✅ Response complete (%d characters)\n", len(totalText))
}
