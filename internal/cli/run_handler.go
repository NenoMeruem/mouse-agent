package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/meruem/prompt-builder-agent/internal/config"
	"github.com/meruem/prompt-builder-agent/internal/llm"
	"github.com/meruem/prompt-builder-agent/internal/llm/claude"
	"github.com/meruem/prompt-builder-agent/internal/llm/gemini"
	"github.com/meruem/prompt-builder-agent/internal/llm/openai"
	"github.com/meruem/prompt-builder-agent/internal/logger"
	"github.com/meruem/prompt-builder-agent/internal/output"
	promptlib "github.com/meruem/prompt-builder-agent/internal/prompt"
	"github.com/meruem/prompt-builder-agent/pkg/models"
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
		logger.Error("prompt not found", "id", id, "error", err)
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
	} else {
		// When --no-select, check if there's data piped on stdin
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			// stdin is a pipe, read from it
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("❌ cannot read stdin: %w", err)
			}
			data["selection"] = strings.TrimSpace(string(bytes))
		}
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

	// Inject tone/length/complexity param suffixes into the prompt
	paramValues := map[string]string{}
	if runTone != "" {
		paramValues["tone"] = runTone
	}
	if runLength != "" {
		paramValues["length"] = runLength
	}
	if runComplexity != "" {
		paramValues["complexity"] = runComplexity
	}
	if len(paramValues) > 0 {
		finalPrompt = promptlib.InjectParams(finalPrompt, paramValues)
	}

	// If dry-run, just preview the prompt without API call
	if dryRun {
		fmt.Printf("🔍 DRY RUN - Final Prompt:\n%s\n", finalPrompt)
		if !rawOutput {
			fmt.Printf("\n📋 Engine: %s | Variables: %v\n", promptDef.Engine, promptDef.Variables)
		}
		return nil
	}

	// Allow user to review and edit the prompt before sending (only when --edit flag is set)
	if editPrompt {
		editedPrompt, confirmed, err := PromptEditor(finalPrompt, promptDef.Engine)
		if err != nil {
			return fmt.Errorf("❌ editor error: %w", err)
		}
		if !confirmed {
			fmt.Println("❌ Cancelled by user (exiting app)")
			os.Exit(0)
		}
		finalPrompt = editedPrompt
	}

	// Resolve engine: --engine flag overrides recipe's engine
	engine := promptDef.Engine
	if runEngine != "" {
		engine = runEngine
	}

	// Send to LLM if engine is specified
	if engine != "" && engine != "none" {
		logger.Info("running prompt", "id", id, "engine", engine)
		response, durationMs, err := runWithLLM(finalPrompt, engine)
		if err != nil {
			logger.Error("LLM run failed", "id", id, "engine", engine, "error", err)
			return err
		}
		logger.Info("prompt completed", "id", id, "engine", engine, "duration_ms", durationMs)

		// Save to history (non-fatal)
		if app.GlobalContext.HistoryStore != nil {
			record := &models.RunRecord{
				ID:          uuid.New().String(),
				PromptID:    promptDef.ID,
				Engine:      engine,
				InputText:   data["selection"],
				FinalPrompt: finalPrompt,
				Response:    response,
				DurationMs:  durationMs,
				CreatedAt:   time.Now(),
			}
			if err := app.GlobalContext.HistoryStore.Append(record); err != nil {
				logger.Warn("failed to save history", "error", err)
			}
		}
		return nil
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
// Returns the full response text, duration in milliseconds, and any error.
func runWithLLM(prompt, engine string) (string, int64, error) {
	// Initialize LLM manager and register available engines
	llmMgr := llm.NewManager()
	registerAvailableEngines(llmMgr)

	// Get the LLM client for requested engine
	client, err := llmMgr.Get(engine)
	if err != nil {
		return "", 0, fmt.Errorf("❌ %w", err)
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

	// Start streaming from LLM (suppress status line in raw/Tauri mode)
	if !rawOutput {
		fmt.Printf("🚀 Streaming from %s...\n", engine)
	}
	ch, err := client.Stream(ctx, req)
	if err != nil {
		logger.Error("stream start failed", "engine", engine, "error", err)
		return "", 0, fmt.Errorf("❌ failed to start streaming: %w", err)
	}

	// Tee the channel: capture response while streaming to renderer
	var responseBuilder strings.Builder
	startTime := time.Now()

	teeCh := make(chan llm.Chunk, 10)
	go func() {
		defer close(teeCh)
		for chunk := range ch {
			if chunk.Text != "" {
				responseBuilder.WriteString(chunk.Text)
			}
			select {
			case teeCh <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Render the output stream — raw mode: no decorators (for Tauri/piped use)
	var renderer output.StreamRenderer
	if rawOutput {
		renderer = output.NewRawStdoutStreamRenderer()
	} else {
		renderer = output.NewFactory(config.GetUIOutput(), engine).CreateRenderer()
	}
	if err := renderer.RenderStream(teeCh); err != nil {
		return "", 0, fmt.Errorf("❌ render error: %w", err)
	}

	return responseBuilder.String(), time.Since(startTime).Milliseconds(), nil
}

// registerAvailableEngines registers LLM engines that have API keys configured.
// It checks both environment variables and config file for API keys.
// Priority: environment variable > config file
func registerAvailableEngines(mgr *llm.Manager) {
	// Register OpenAI if API key is available
	// Priority: environment variable > config file (with env: prefix support)
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		openaiKey = config.GetEngineAPIKey("openai")
	}
	if openaiKey != "" {
		model := config.GetEngineModel("openai")
		timeout := config.GetEngineTimeout("openai")
		mgr.Register("openai", openai.NewClient(openaiKey, model, timeout))
	}

	// Register Gemini if API key is available
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		geminiKey = config.GetEngineAPIKey("gemini")
	}
	if geminiKey != "" {
		model := config.GetEngineModel("gemini")
		timeout := config.GetEngineTimeout("gemini")
		mgr.Register("gemini", gemini.NewClient(geminiKey, model, timeout))
	}

	// Register Claude if API key is available
	claudeKey := os.Getenv("ANTHROPIC_API_KEY")
	if claudeKey == "" {
		claudeKey = config.GetEngineAPIKey("claude")
	}
	if claudeKey != "" {
		model := config.GetEngineModel("claude")
		timeout := config.GetEngineTimeout("claude")
		mgr.Register("claude", claude.NewClient(claudeKey, model, timeout))
	}
}
