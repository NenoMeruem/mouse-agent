package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm"
)

// TestGeminiClientName tests the Name method
func TestGeminiClientName(t *testing.T) {
	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	if client.Name() != "gemini" {
		t.Errorf("expected 'gemini', got '%s'", client.Name())
	}
}

// TestGeminiStreamSuccess tests successful streaming
func TestGeminiStreamSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GenerateContentResponse{
			Candidates: []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			}{
				{
					Content: struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					}{
						Parts: []struct {
							Text string `json:"text"`
						}{
							{Text: "This is a test response"},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunks := []llm.Chunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Text != "This is a test response" {
		t.Errorf("expected text chunk, got: %s", chunks[0].Text)
	}
	if !chunks[1].Done {
		t.Errorf("expected done marker")
	}
}

// TestGeminiStreamUnauthorized tests 401 error
func TestGeminiStreamUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	client := NewClient("invalid-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected error for 401 Unauthorized")
	}
}

// TestGeminiStreamRateLimited tests 429 error
func TestGeminiStreamRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "Rate limit exceeded"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected error for 429 Too Many Requests")
	}
}

// TestGeminiStreamServerError tests 500 error
func TestGeminiStreamServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected error for 500 Internal Server Error")
	}
}

// TestGeminiStreamMalformedJSON tests malformed response
func TestGeminiStreamMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected error for malformed JSON")
	}
}

// TestGeminiStreamTimeout tests timeout handling
func TestGeminiStreamTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 100*time.Millisecond)
	client.baseURL = server.URL

	ctx := context.Background()
	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected timeout error")
	}
}

// TestGeminiStreamContextCancellation tests context cancellation
func TestGeminiStreamContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
	client.baseURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := llm.Request{Prompt: "Hello", Model: "gemini-2.5-flash-lite"}

	ch, err := client.Stream(ctx, req)
	if err != nil {
		return // Context error is acceptable
	}

	chunk := <-ch
	if chunk.Err == nil {
		t.Errorf("expected error for cancelled context")
	}
}

// TestGeminiExtractText tests text extraction
func TestGeminiExtractText(t *testing.T) {
	tests := []struct {
		name     string
		response GenerateContentResponse
		expected string
	}{
		{
			name: "single text",
			response: GenerateContentResponse{
				Candidates: []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				}{
					{
						Content: struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						}{
							Parts: []struct {
								Text string `json:"text"`
							}{
								{Text: "Hello, World!"},
							},
						},
					},
				},
			},
			expected: "Hello, World!",
		},
		{
			name:     "empty response",
			response: GenerateContentResponse{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("test-key", "gemini-2.5-flash-lite", 30*time.Second)
			result := client.extractText(tt.response)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGeminiInterfaceImplementation verifies interface implementation
func TestGeminiInterfaceImplementation(t *testing.T) {
	var _ llm.Client = (*Client)(nil)
}
