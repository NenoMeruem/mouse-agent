package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm"
)

// Client is a Google Gemini API client
type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	baseURL string
}

// NewClient creates a new Gemini client
func NewClient(apiKey string, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
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
	// Build request using c.model from config, not hardcoded
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	
	fmt.Fprintf(os.Stderr, "DEBUG: Gemini URL=%s, timeout=%v\n", url[:len(url)-len(c.apiKey)-4]+"KEY", c.timeout)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": req.Prompt},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to marshal request: %w", err), Done: true}
		return
	}

	// Don't use context here - let http.Client.Timeout handle it
	// Using context can conflict with signal.NotifyContext from run_handler
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to create request: %w", err), Done: true}
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Create HTTP client with timeout
	// Note: Don't use ResponseHeaderTimeout as it can be too strict
	transport := &http.Transport{
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 10,
	}

	client := &http.Client{
		Timeout:   c.timeout, // Use full timeout (120s from config)
		Transport: transport,
	}
	
	fmt.Fprintf(os.Stderr, "DEBUG: Starting request with timeout=%v\n", client.Timeout)

	resp, err := client.Do(httpReq)
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("request failed: %w", err), Done: true}
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		ch <- llm.Chunk{
			Err:  fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes)),
			Done: true,
		}
		return
	}

	// Read and parse response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to read response: %w", err), Done: true}
		return
	}

	var geminiResp GenerateContentResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		ch <- llm.Chunk{Err: fmt.Errorf("failed to parse response: %w", err), Done: true}
		return
	}

	// Extract text and stream it
	text := c.extractText(geminiResp)
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
func (c *Client) extractText(resp GenerateContentResponse) string {
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return part.Text
			}
		}
	}
	return ""
}

// GenerateContentResponse represents Gemini's API response format
type GenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}
