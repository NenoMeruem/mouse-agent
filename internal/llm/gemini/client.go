// Package gemini provides a Google Gemini API client with streaming support.
package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"google.golang.org/api/option"
)

// Client implements the llm.Client interface for Google Gemini's API.
// It uses the official github.com/google/generative-ai-go SDK for API communication.
type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
}

// NewClient creates a new Gemini client with the given API key and model.
func NewClient(apiKey string, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "gemini"
}

// Stream implements the Client interface
// Returns response as streaming chunks
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk)

	go func() {
		defer close(ch)
		c.stream(ctx, req, ch)
	}()

	return ch, nil
}

// stream handles the Gemini API call with streaming
func (c *Client) stream(ctx context.Context, req llm.Request, ch chan<- llm.Chunk) {
	// Create a new Gemini client with API key option
	gc, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to create Gemini client: %w", err), Done: true}
		return
	}
	defer gc.Close()

	// Get the model
	model := gc.GenerativeModel(c.model)

	// Set timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Generate content using the SDK
	resp, err := model.GenerateContent(timeoutCtx, genai.Text(req.Prompt))
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("Gemini API error: %w", err), Done: true}
		return
	}

	// Extract and stream text from response
	text := c.extractText(resp)
	if text != "" {
		// Split text into chunks for streaming effect
		// Send in 100-char chunks to simulate streaming
		chunkSize := 100
		for i := 0; i < len(text); i += chunkSize {
			end := i + chunkSize
			if end > len(text) {
				end = len(text)
			}
			ch <- llm.Chunk{Text: text[i:end], Done: false}
		}
	}

	// Send done marker
	ch <- llm.Chunk{Done: true}
}

// extractText extracts the text content from Gemini response
func (c *Client) extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}

	// Iterate through candidates and extract text
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if text, ok := part.(genai.Text); ok {
					return string(text)
				}
			}
		}
	}

	return ""
}
