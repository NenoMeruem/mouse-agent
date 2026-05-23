package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/meruem/promptly/internal/app"
	"github.com/meruem/promptly/internal/llm"
	"github.com/meruem/promptly/internal/logger"
	"github.com/meruem/promptly/pkg/models"
	"github.com/spf13/cobra"
)

var (
	chatMessagesJSON string
	chatEngine       string
	chatSessionID    string
	chatTurnIndex    int
	chatPromptID     string
)

func init() {
	chatCmd.Flags().StringVar(&chatMessagesJSON, "messages", "", "JSON array of {role,content} messages (required)")
	chatCmd.Flags().StringVar(&chatEngine, "engine", "", "Engine to use (required)")
	chatCmd.Flags().BoolVar(&rawOutput, "raw", false, "Raw stdout output — no decorators (for Tauri)")
	chatCmd.Flags().StringVar(&chatSessionID, "session-id", "", "Conversation session ID — links turns in history")
	chatCmd.Flags().IntVar(&chatTurnIndex, "turn-index", 1, "Turn index within the session (1-based for follow-ups)")
	chatCmd.Flags().StringVar(&chatPromptID, "prompt-id", "", "Original recipe ID for history grouping")
	rootCmd.AddCommand(chatCmd)
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Continue a conversation with an LLM",
	Long:  "Send a multi-turn message history to an LLM and stream the response.",
	RunE:  runChatCommand,
}

func runChatCommand(cmd *cobra.Command, args []string) error {
	if chatMessagesJSON == "" {
		return fmt.Errorf("❌ --messages is required")
	}
	if chatEngine == "" {
		return fmt.Errorf("❌ --engine is required")
	}

	var messages []llm.Message
	if err := json.Unmarshal([]byte(chatMessagesJSON), &messages); err != nil {
		return fmt.Errorf("❌ invalid messages JSON: %w", err)
	}
	if len(messages) == 0 {
		return fmt.Errorf("❌ messages array is empty")
	}

	// The last user message is what we're sending now
	var userInput string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userInput = messages[i].Content
			break
		}
	}

	req := llm.Request{Messages: messages}
	response, durationMs, err := runWithLLMRequest(req, chatEngine)
	if err != nil {
		return err
	}

	// Save follow-up turn to history (non-fatal)
	if app.GlobalContext != nil && app.GlobalContext.HistoryStore != nil && response != "" {
		record := &models.RunRecord{
			ID:          uuid.New().String(),
			SessionID:   chatSessionID,
			TurnIndex:   chatTurnIndex,
			PromptID:    chatPromptID,
			Engine:      chatEngine,
			InputText:   userInput,
			FinalPrompt: userInput,
			Response:    response,
			DurationMs:  durationMs,
			CreatedAt:   time.Now(),
		}
		if err := app.GlobalContext.HistoryStore.Append(record); err != nil {
			logger.Warn("failed to save chat turn to history", "turn", chatTurnIndex, "error", err)
		}
	}

	return nil
}
