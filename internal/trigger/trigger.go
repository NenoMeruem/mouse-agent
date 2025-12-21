package trigger

// Trigger defines the interface for trigger listeners
type Trigger interface {
	// Name returns the trigger name
	Name() string

	// Start begins listening for trigger events
	Start() error

	// Stop stops the trigger listener
	Stop() error

	// IsActive returns true if trigger is currently listening
	IsActive() bool
}
