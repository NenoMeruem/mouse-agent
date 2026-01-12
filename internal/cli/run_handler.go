package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/internal/config"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"github.com/sl/prompt-builder-agent/internal/llm/gemini"
	"github.com/sl/prompt-builder-agent/internal/llm/openai"
	"github.com/sl/prompt-builder-agent/internal/output"
	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/spf13/cobra"
)

func runPromptCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Get prompt definition
	promptDef, err := app.GlobalContext.PromptStore.Get(id)
	if err != nil {
		return fmt.Errorf("❌ prompt not found: %s", id)
	}

	// Build data map for template
	data := map[string]string{}

	// Get selection from clipboard (unless --no-select is set)
	if !noSelection {
		selectionText, err := app.GlobalContext.SelectionManager.GetSelection()
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}
		data["selection"] = selectionText
	}

	// Collect missing variables from environment or user input
	reader := bufio.NewReader(os.Stdin)
	for _, varName := range promptDef.Variables {
		if _, ok := data[varName]; !ok {
			// Try environment variable first
			envKey := "PROMPT_" + strings.ToUpper(varName)
			if envValue := os.Getenv(envKey); envValue != "" {
				data[varName] = envValue
			} else if noSelection && varName == "selection" {
				// Skip if --no-select and variable is selection
				continue
			} else {
				// Ask user interactively
				fmt.Printf("Enter %s: ", varName)
				value, _ := reader.ReadString('\n')
				value = strings.TrimSpace(value)
				if value == "" {
					return fmt.Errorf("❌ missing required variable: %s", varName)
				}
				data[varName] = value
			}
		}
	}

	// Validate all required variables are present
	if err := promptlib.ValidateVariables(promptDef.Variables, data); err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Build final prompt
	finalPrompt, err := app.GlobalContext.Builder.Build(
		promptDef.Template,
		data,
	)
	if err != nil {
		return fmt.Errorf("❌ cannot build prompt: %w", err)
	}

	// If dry-run, just show the prompt
	if dryRun {
		fmt.Printf("🔍 DRY RUN - Final Prompt:\n%s\n", finalPrompt)
		if !rawOutput {
			fmt.Printf("\n📋 Engine: %s | Variables: %v\n", promptDef.Engine, promptDef.Variables)
		}
		return nil
	}

	// Stream to LLM if engine is specified
	if promptDef.Engine != "" && promptDef.Engine != "none" {
		return runWithLLM(finalPrompt, promptDef.Engine)
	}

	// Fallback: show prompt without LLM
	fmt.Printf("🤖 Prompt:\n%s\n", finalPrompt)
	return nil
}

// runWithLLM sends the prompt to an LLM with streaming output
func runWithLLM(prompt, engine string) error {
	// Initialize LLM manager
	llmMgr := llm.NewManager()

	// Register providers based on configuration
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		model := getConfigValue("openai_model", "gpt-4-mini")
		timeout := getTimeoutFromConfig("openai")
		fmt.Fprintf(os.Stderr, "DEBUG: OpenAI timeout=%v\n", timeout)
		llmMgr.Register("openai", openai.NewClient(apiKey, model, timeout))
	}

	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		model := getConfigValue("gemini_model", "gemini-2.5-flash-lite")
		timeout := getTimeoutFromConfig("gemini")
		fmt.Fprintf(os.Stderr, "DEBUG: Gemini timeout=%v, apiKey=%v\n", timeout, apiKey[:10]+"...")
		llmMgr.Register("gemini", gemini.NewClient(apiKey, model, timeout))
	}

	// Get the LLM client
	client, err := llmMgr.Get(engine)
	if err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Create cancellable context
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	// Create LLM request
	req := llm.Request{
		Prompt: prompt,
		Model:  getConfigValueFromEngine(engine, "model"),
	}

	// Stream from LLM
	fmt.Printf("🚀 Streaming from %s...\n", engine)
	ch, err := client.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("❌ failed to start streaming: %w", err)
	}

	// Render output
	outputType := getConfigValue("ui_output", "stdout")
	factory := output.NewFactory(outputType)
	renderer := factory.CreateRenderer()

	if err := renderer.RenderStream(ch); err != nil {
		return fmt.Errorf("❌ render error: %w", err)
	}

	return nil
}

// Helper functions to get config values
func getConfigValue(key, defaultValue string) string {
	if config.AppConfig != nil {
		switch key {
		case "openai_model":
			if cfg, ok := config.AppConfig.Engines["openai"]; ok && cfg.Model != "" {
				return cfg.Model
			}
		case "gemini_model":
			if cfg, ok := config.AppConfig.Engines["gemini"]; ok && cfg.Model != "" {
				return cfg.Model
			}
		case "ui_output":
			if config.AppConfig.UI.Output != "" {
				return config.AppConfig.UI.Output
			}
		}
	}
	return defaultValue
}

func getConfigValueFromEngine(engine, key string) string {
	if config.AppConfig != nil {
		if cfg, ok := config.AppConfig.Engines[engine]; ok {
			if key == "model" && cfg.Model != "" {
				return cfg.Model
			}
		}
	}

	// Return engine-specific defaults
	switch engine {
	case "openai":
		return "gpt-4-mini"
	case "gemini":
		return "gemini-2.5-flash-lite"
	default:
		return "gpt-4-mini"
	}
}

// getTimeoutFromConfig returns timeout for given engine from config
// Falls back to 120s if not configured
func getTimeoutFromConfig(engine string) time.Duration {
	if config.AppConfig != nil {
		if cfg, ok := config.AppConfig.Engines[engine]; ok {
			if cfg.Timeout > 0 {
				return cfg.Timeout
			}
		}
	}
	// Default to 120 seconds
	return 120 * time.Second
}
