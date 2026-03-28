package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/internal/config"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"github.com/sl/prompt-builder-agent/internal/llm/claude"
	"github.com/sl/prompt-builder-agent/internal/llm/gemini"
	"github.com/sl/prompt-builder-agent/internal/llm/openai"
	"github.com/sl/prompt-builder-agent/internal/output"
	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/spf13/cobra"
)

// runPromptCommand orchestrates the full prompt execution flow:
// 1. Load prompt definition
// 2. Collect input data (selection + user variables)
// 3. Build the final prompt
// 4. Send to LLM and render response
func runPromptCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Get prompt definition
	promptDef, err := app.GlobalContext.PromptStore.Get(id)
	if err != nil {
		return fmt.Errorf("❌ prompt not found: %s", id)
	}

	// Collect input data for template substitution
	data := map[string]string{}

	// Get selection from clipboard (unless --no-select is set)
	if !noSelection {
		selectionText, err := app.GlobalContext.SelectionManager.GetSelection()
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}
		data["selection"] = selectionText
	}

	// Prompt user for any missing variables from prompt definition
	if err := collectMissingVariables(promptDef.Variables, data); err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Build the final prompt from template
	finalPrompt, err := app.GlobalContext.Builder.Build(
		promptDef.Template,
		data,
	)
	if err != nil {
		return fmt.Errorf("❌ cannot build prompt: %w", err)
	}

	// If dry-run, just preview the prompt without API call
	if dryRun {
		fmt.Printf("🔍 DRY RUN - Final Prompt:\n%s\n", finalPrompt)
		if !rawOutput {
			fmt.Printf("\n📋 Engine: %s | Variables: %v\n", promptDef.Engine, promptDef.Variables)
		}
		return nil
	}

	// Allow user to review and edit the prompt before sending
	if promptDef.Engine != "" && promptDef.Engine != "none" {
		editedPrompt, confirmed, err := PromptEditor(finalPrompt, promptDef.Engine)
		if err != nil {
			return fmt.Errorf("❌ editor error: %w", err)
		}
		if !confirmed {
			fmt.Println("❌ Cancelled by user (exiting app)")
			// immediately terminate the entire application instead of
			// continuing execution. this ensures that pressing cancel in
			// the prompt editor quits the program as requested.
			os.Exit(0)
		}
		finalPrompt = editedPrompt
	}

	// Send to LLM if engine is specified
	if promptDef.Engine != "" && promptDef.Engine != "none" {
		return runWithLLM(finalPrompt, promptDef.Engine)
	}

	// Fallback: show prompt without LLM call
	fmt.Printf("🤖 Prompt:\n%s\n", finalPrompt)
	return nil
}

// collectMissingVariables prompts user for any variables not yet provided.
// It tries environment variables first, then asks user interactively.
func collectMissingVariables(requiredVars []string, data map[string]string) error {
	reader := bufio.NewReader(os.Stdin)

	for _, varName := range requiredVars {
		if _, ok := data[varName]; ok {
			continue // Already have this variable from selection
		}

		// Try environment variable first
		envKey := "PROMPT_" + strings.ToUpper(varName)
		if envValue := os.Getenv(envKey); envValue != "" {
			data[varName] = envValue
			continue
		}

		// Skip selection variable if --no-select was used
		if noSelection && varName == "selection" {
			continue
		}

		// Ask user interactively
		fmt.Printf("Enter %s: ", varName)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("missing required variable: %s", varName)
		}
		data[varName] = value
	}

	// Validate all required variables are present
	return promptlib.ValidateVariables(requiredVars, data)
}

// runWithLLM sends the prompt to an LLM engine and renders the streamed response.
// It handles setting up the correct LLM client, managing context cancellation,
// and rendering output according to user preferences.
func runWithLLM(prompt, engine string) error {
	// Initialize LLM manager and register available engines
	llmMgr := llm.NewManager()
	registerAvailableEngines(llmMgr)

	// Get the LLM client for requested engine
	client, err := llmMgr.Get(engine)
	if err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Create cancellable context (Ctrl+C support)
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	// Create LLM request
	req := llm.Request{
		Prompt: prompt,
		Model:  config.GetEngineModel(engine),
	}

	// Start streaming from LLM
	fmt.Printf("🚀 Streaming from %s...\n", engine)
	ch, err := client.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("❌ failed to start streaming: %w", err)
	}

	// Render the output stream
	renderer := output.NewFactory(config.GetUIOutput(), engine).CreateRenderer()
	if err := renderer.RenderStream(ch); err != nil {
		return fmt.Errorf("❌ render error: %w", err)
	}

	return nil
}

// registerAvailableEngines registers LLM engines that have API keys configured.
// It checks both environment variables and config file for API keys.
// Priority: environment variable > config file
func registerAvailableEngines(mgr *llm.Manager) {
	// Register OpenAI if API key is available
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" && config.AppConfig != nil {
		if cfg, ok := config.AppConfig.Engines["openai"]; ok {
			openaiKey = cfg.APIKey
		}
	}
	if openaiKey != "" {
		model := config.GetEngineModel("openai")
		timeout := config.GetEngineTimeout("openai")
		mgr.Register("openai", openai.NewClient(openaiKey, model, timeout))
	}

	// Register Gemini if API key is available
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" && config.AppConfig != nil {
		if cfg, ok := config.AppConfig.Engines["gemini"]; ok {
			geminiKey = cfg.APIKey
		}
	}
	if geminiKey != "" {
		model := config.GetEngineModel("gemini")
		timeout := config.GetEngineTimeout("gemini")
		mgr.Register("gemini", gemini.NewClient(geminiKey, model, timeout))
	}

	// Register Claude if API key is available
	claudeKey := os.Getenv("ANTHROPIC_API_KEY")
	if claudeKey == "" && config.AppConfig != nil {
		if cfg, ok := config.AppConfig.Engines["claude"]; ok {
			claudeKey = cfg.APIKey
		}
	}
	if claudeKey != "" {
		model := config.GetEngineModel("claude")
		timeout := config.GetEngineTimeout("claude")
		mgr.Register("claude", claude.NewClient(claudeKey, model, timeout))
	}
}
