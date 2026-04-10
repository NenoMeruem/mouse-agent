package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sl/prompt-builder-agent/internal/config"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"github.com/sl/prompt-builder-agent/internal/llm/claude"
	"github.com/sl/prompt-builder-agent/internal/llm/gemini"
	"github.com/sl/prompt-builder-agent/internal/llm/openai"
	"github.com/spf13/cobra"
)

// PingResult is the JSON result returned by ping-engine.
type PingResult struct {
	Status  string `json:"status"`  // ok | rate_limited | auth_error | no_key | error
	Message string `json:"message"`
}

func init() {
	configCmd.AddCommand(pingEngineCmd)
}

var pingEngineCmd = &cobra.Command{
	Use:   "ping-engine <engine>",
	Short: "Test if an engine API key is valid and within quota",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result := pingEngine(args[0])
		b, _ := json.Marshal(result)
		fmt.Println(string(b))
		return nil
	},
}

func pingEngine(engine string) PingResult {
	// Resolve API key: env var takes priority over config file.
	var key string
	switch engine {
	case "openai":
		key = os.Getenv("OPENAI_API_KEY")
	case "gemini":
		key = os.Getenv("GEMINI_API_KEY")
	case "claude":
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		key = config.GetEngineAPIKey(engine)
	}
	if key == "" {
		return PingResult{Status: "no_key", Message: "No API key configured"}
	}

	model := config.GetEngineModel(engine)
	timeout := 12 * time.Second

	var client llm.Client
	switch engine {
	case "gemini":
		client = gemini.NewClient(key, model, timeout)
	case "openai":
		client = openai.NewClient(key, model, timeout)
	case "claude":
		client = claude.NewClient(key, model, timeout)
	default:
		return PingResult{Status: "error", Message: "Unknown engine: " + engine}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch, err := client.Stream(ctx, llm.Request{Prompt: "hi", Model: model})
	if err != nil {
		return classifyLLMError(err.Error())
	}

	// Wait for first chunk — cancel streaming right after.
	select {
	case chunk, ok := <-ch:
		cancel() // stop streaming immediately, we only needed the first signal
		if !ok || chunk.Done {
			return PingResult{Status: "ok", Message: ""}
		}
		if chunk.Err != nil {
			return classifyLLMError(chunk.Err.Error())
		}
		return PingResult{Status: "ok", Message: ""}
	case <-ctx.Done():
		return PingResult{Status: "error", Message: "Request timed out"}
	}
}

func classifyLLMError(msg string) PingResult {
	lower := strings.ToLower(msg)

	isRateLimit := strings.Contains(lower, "429") ||
		strings.Contains(lower, "quota") ||
		strings.Contains(lower, "rate_limit") ||
		strings.Contains(lower, "resource_exhausted") ||
		strings.Contains(lower, "insufficient_quota") ||
		strings.Contains(lower, "overloaded") ||
		strings.Contains(lower, "too many requests")

	isAuth := strings.Contains(lower, "401") ||
		strings.Contains(lower, "403") ||
		strings.Contains(lower, "invalid_api_key") ||
		strings.Contains(lower, "authentication_error") ||
		strings.Contains(lower, "permission_denied") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "invalid key") ||
		strings.Contains(lower, "api key")

	switch {
	case isRateLimit:
		return PingResult{Status: "rate_limited", Message: "Rate limited or quota exceeded"}
	case isAuth:
		return PingResult{Status: "auth_error", Message: "Invalid or unauthorized API key"}
	default:
		if len(msg) > 120 {
			msg = msg[:120] + "…"
		}
		return PingResult{Status: "error", Message: msg}
	}
}
