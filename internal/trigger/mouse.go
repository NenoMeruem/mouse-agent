package trigger

// MouseTrigger defines interface for mouse-based triggers
// This is Phase 4 design - actual implementation comes in Phase 5+
type MouseTrigger interface {
	Trigger

	// GetMouseButton returns which button triggers (middle, right, etc.)
	GetMouseButton() string

	// GetAction returns the action type (context_menu, direct_run, etc.)
	GetAction() string
}

// MouseTriggerConfig holds configuration for mouse triggers
type MouseTriggerConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Button        string `mapstructure:"button"` // middle, right
	Action        string `mapstructure:"action"` // context_menu, direct_run
	DefaultPrompt string `mapstructure:"default_prompt"`
}

// Note: Real implementation requires OS-specific code:
// macOS: Karabiner or CGEventTap (cgo)
// Linux: X11 event listening
// Windows: Mouse hook DLL
//
// Phase 4 just defines the interface to prepare for future implementation
