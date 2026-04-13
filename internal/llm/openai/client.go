// Package openai provides an OpenAI API client with streaming support.
package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meruem/prompt-builder-agent/internal/llm"
)

// Client implements the llm.Client interface for OpenAI's API.
// It supports streaming responses via Server-Sent Events (SSE).
type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	baseURL string
}

// NewClient creates a new OpenAI client with the given API key and model.
func NewClient(apiKey string, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		baseURL: "https://api.openai.com/v1",
	}
}

// Name returns the provider name
func (c *Client) Name() string {
	return "openai"
}

// Stream implements the Client interface with streaming support
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk)

	// Create HTTP request for streaming
	httpReq, err := c.createRequest(ctx, req)
	if err != nil {
		close(ch)
		return ch, err
	}

	// Start streaming in a goroutine
	go func() {
		defer close(ch)
		c.stream(httpReq, ch)
	}()

	return ch, nil
}

// createRequest creates an HTTP request for OpenAI API
func (c *Client) createRequest(ctx context.Context, req llm.Request) (*http.Request, error) {
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

	payload := map[string]interface{}{
		"model":       req.Model,
		"messages":    []map[string]string{{"role": "user", "content": req.Prompt}},
		"stream":      true,
		"temperature": 0.7,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	return httpReq, nil
}

// stream handles the SSE streaming response
func (c *Client) stream(req *http.Request, ch chan<- llm.Chunk) {
	client := &http.Client{
		Timeout: c.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("request failed: %w", err), Done: true}
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ch <- llm.Chunk{
			Err:  fmt.Errorf("API error %d: %s", resp.StatusCode, string(body)),
			Done: true,
		}
		return
	}

	// Stream Server-Sent Events
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for stream termination marker
		if line == "data: [DONE]" {
			ch <- llm.Chunk{Done: true}
			return
		}

		// Parse SSE line
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// Skip malformed lines
			continue
		}

		// Extract text from choices
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			ch <- llm.Chunk{
				Text: streamResp.Choices[0].Delta.Content,
				Done: false,
			}
		}
	}

	// Handle scanner error
	if err := scanner.Err(); err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("stream error: %w", err), Done: true}
	}
}

// StreamResponse represents OpenAI's streaming response format
type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
