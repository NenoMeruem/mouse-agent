package gemini

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm"
)

// TestGeminiRealAPI tests with real Gemini API (requires GEMINI_API_KEY env var)
// Run with: GEMINI_API_KEY=... go test -run TestGeminiRealAPI -v
func TestGeminiRealAPI(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping real API test")
	}

	client := NewClient(apiKey, "gemini-2.5-flash-lite", 30*time.Second)

	ctx := context.Background()
	req := llm.Request{
		Prompt: "Explain what is machine learning in 2 sentences.",
		Model:  "gemini-2.5-flash-lite",
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunks := []llm.Chunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
		if chunk.Err != nil {
			t.Logf("Error chunk: %v", chunk.Err)
		}
		if chunk.Text != "" {
			t.Logf("Text: %s", chunk.Text)
		}
	}

	// Verify we got chunks
	if len(chunks) == 0 {
		t.Errorf("expected chunks, got none")
	}

	// Verify we got at least some text
	hasText := false
	for _, chunk := range chunks {
		if chunk.Text != "" {
			hasText = true
			break
		}
	}

	if !hasText {
		t.Errorf("expected text response, got empty")
	}

	// Verify last chunk is done marker
	if len(chunks) > 0 && !chunks[len(chunks)-1].Done {
		t.Errorf("expected last chunk to be done marker")
	}

	t.Logf("✅ Real API test passed with %d chunks", len(chunks))
}

// TestGeminiRealAPIMultipleRequests tests multiple sequential requests
// Run with: GEMINI_API_KEY=... go test -run TestGeminiRealAPIMultipleRequests -v
func TestGeminiRealAPIMultipleRequests(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping real API test")
	}

	client := NewClient(apiKey, "gemini-2.5-flash-lite", 30*time.Second)

	prompts := []string{
		"What is Python?",
		"What is Go?",
		"What is Rust?",
	}

	for i, prompt := range prompts {
		t.Run(prompt, func(t *testing.T) {
			ctx := context.Background()
			req := llm.Request{
				Prompt: prompt,
				Model:  "gemini-2.5-flash-lite",
			}

			ch, err := client.Stream(ctx, req)
			if err != nil {
				t.Fatalf("Stream() failed: %v", err)
			}

			textCount := 0
			for chunk := range ch {
				if chunk.Text != "" {
					textCount++
				}
				if chunk.Err != nil {
					t.Errorf("unexpected error: %v", chunk.Err)
					return
				}
			}

			if textCount == 0 {
				t.Errorf("expected text response")
			}

			t.Logf("✅ Request %d passed", i+1)
		})
	}
}

// TestGeminiRealAPITimeout tests timeout with real API
// Run with: GEMINI_API_KEY=... go test -run TestGeminiRealAPITimeout -v
func TestGeminiRealAPITimeout(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping real API test")
	}

	// Very short timeout
	client := NewClient(apiKey, "gemini-2.5-flash-lite", 1*time.Millisecond)

	ctx := context.Background()
	req := llm.Request{
		Prompt: "Hello",
		Model:  "gemini-2.5-flash-lite",
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	hasError := false
	for chunk := range ch {
		if chunk.Err != nil {
			hasError = true
			t.Logf("Got expected timeout error: %v", chunk.Err)
		}
	}

	if !hasError {
		t.Logf("⚠️ Expected timeout but got response (API might be very fast)")
	}
}

// TestGeminiRealAPIContextCancel tests context cancellation with real API
// Run with: GEMINI_API_KEY=... go test -run TestGeminiRealAPIContextCancel -v
func TestGeminiRealAPIContextCancel(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping real API test")
	}

	client := NewClient(apiKey, "gemini-2.5-flash-lite", 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req := llm.Request{
		Prompt: "Write a very long essay about the history of computing",
		Model:  "gemini-2.5-flash-lite",
	}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	totalChunks := 0
	for chunk := range ch {
		totalChunks++
		if chunk.Err != nil {
			t.Logf("✅ Got expected cancel error: %v", chunk.Err)
			return
		}
	}

	t.Logf("⚠️ Request completed with %d chunks before cancel took effect", totalChunks)
}
