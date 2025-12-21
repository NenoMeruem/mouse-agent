package app

import (
	"fmt"

	"github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/sl/prompt-builder-agent/internal/selection"
	"github.com/sl/prompt-builder-agent/internal/storage"
)

type AppContext struct {
	PromptStore storage.PromptStore
	Builder     prompt.Builder
	Selection   selection.Provider
}

var GlobalContext *AppContext

func InitializeContext() error {
	dbPath := storage.GetDefaultDBPath()

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("cannot create prompt store: %w", err)
	}

	if err := store.Init(); err != nil {
		return fmt.Errorf("cannot initialize prompt store: %w", err)
	}

	builder := prompt.NewSimpleBuilder()
	selectionProvider := selection.NewProvider()

	GlobalContext = &AppContext{
		PromptStore: store,
		Builder:     builder,
		Selection:   selectionProvider,
	}

	return nil
}
