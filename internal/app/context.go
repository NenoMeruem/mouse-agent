// Package app handles application-wide initialization and context management.
package app

import (
	"fmt"

	"github.com/meruem/prompt-builder-agent/internal/config"
	"github.com/meruem/prompt-builder-agent/internal/logger"
	"github.com/meruem/prompt-builder-agent/internal/prompt"
	"github.com/meruem/prompt-builder-agent/internal/selection"
	"github.com/meruem/prompt-builder-agent/internal/storage"
)

// AppContext holds the core dependencies needed throughout the application.
// It is initialized once at startup and accessed globally.
type AppContext struct {
	PromptStore      storage.PromptStore  // Handles prompt persistence
	HistoryStore     storage.HistoryStore // Handles run history persistence
	Builder          prompt.Builder       // Handles template variable substitution
	SelectionManager *selection.Manager   // Handles clipboard/selection access
	Config           *config.Config       // Holds application configuration
}

// GlobalContext is the application-wide context initialized at startup.
var GlobalContext *AppContext

// InitializeContext sets up the application context by initializing all core
// components: storage, prompt builder, selection manager, and configuration.
// This should be called once at application startup.
func InitializeContext() error {
	dbPath := storage.GetDefaultDBPath()

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("cannot create prompt store: %w", err)
	}

	// Run one-time migration from JSON if needed (non-fatal).
	jsonPath := storage.GetDefaultJSONPath()
	if err := storage.MigrateFromJSON(store, jsonPath); err != nil {
		logger.Warn("JSON migration failed", "error", err)
		fmt.Printf("warning: JSON migration failed: %v\n", err)
	}

	historyStore, err := storage.NewSQLiteHistoryStore(store.DB())
	if err != nil {
		return fmt.Errorf("cannot create history store: %w", err)
	}

	builder := prompt.NewSimpleBuilder()
	selectionProvider := selection.NewProvider()
	selectionManager := selection.NewManager(selectionProvider)

	// Load config (ignore error if not initialized).
	cfg := &config.Config{}
	_ = config.Load()
	if config.AppConfig != nil {
		cfg = config.AppConfig
	}

	GlobalContext = &AppContext{
		PromptStore:      store,
		HistoryStore:     historyStore,
		Builder:          builder,
		SelectionManager: selectionManager,
		Config:           cfg,
	}

	return nil
}
