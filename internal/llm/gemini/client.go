// Package gemini provides a Google Gemini API client with streaming support.
package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"google.golang.org/api/iterator"
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

// stream handles the Gemini API call with real streaming via GenerateContentStream.
func (c *Client) stream(ctx context.Context, req llm.Request, ch chan<- llm.Chunk) {
	gc, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to create Gemini client: %w", err), Done: true}
		return
	}
	defer gc.Close()

	model := gc.GenerativeModel(c.model)

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	iter := model.GenerateContentStream(timeoutCtx, genai.Text(req.Prompt))
	for {
		resp, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			if timeoutCtx.Err() != nil {
				break
			}
			ch <- llm.Chunk{Err: fmt.Errorf("Gemini stream error: %w", err), Done: true}
			return
		}
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					if text, ok := part.(genai.Text); ok && string(text) != "" {
						ch <- llm.Chunk{Text: string(text), Done: false}
					}
				}
			}
		}
	}

	ch <- llm.Chunk{Done: true}
}
