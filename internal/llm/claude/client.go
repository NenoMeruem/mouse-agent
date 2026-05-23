// Package claude provides an Anthropic Claude API client with streaming support.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meruem/promptly/internal/llm"
)

// Client implements the llm.Client interface for Anthropic's Claude API.
// It supports streaming responses via Server-Sent Events (SSE).
type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	baseURL string
}

// NewClient creates a new Claude client with the given API key and model.
func NewClient(apiKey string, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		baseURL: "https://api.anthropic.com/v1",
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "claude"
}

// Stream implements the Client interface with streaming support.
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk)

	httpReq, err := c.createRequest(ctx, req)
	if err != nil {
		close(ch)
		return ch, err
	}

	go func() {
		defer close(ch)
		c.stream(httpReq, ch)
	}()

	return ch, nil
}

// createRequest creates an HTTP request for the Anthropic Messages API.
func (c *Client) createRequest(ctx context.Context, req llm.Request) (*http.Request, error) {
	url := fmt.Sprintf("%s/messages", c.baseURL)

	var msgs []map[string]string
	if len(req.Messages) > 0 {
		for _, m := range req.Messages {
			msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
		}
	} else {
		msgs = []map[string]string{{"role": "user", "content": req.Prompt}}
	}

	payload := map[string]interface{}{
		"model":      req.Model,
		"messages":   msgs,
		"max_tokens": 4096,
		"stream":     true,
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
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	return httpReq, nil
}

// stream handles the SSE streaming response from the Anthropic API.
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ch <- llm.Chunk{
			Err:  fmt.Errorf("API error %d: %s", resp.StatusCode, string(body)),
			Done: true,
		}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event StreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				ch <- llm.Chunk{Text: event.Delta.Text, Done: false}
			}
		case "message_stop":
			ch <- llm.Chunk{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("stream error: %w", err), Done: true}
	}
}

// StreamEvent represents an Anthropic SSE event payload.
type StreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}
