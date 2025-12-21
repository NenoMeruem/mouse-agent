package selection

// NewProvider returns the appropriate selection provider for the current OS
func NewProvider() Provider {
	// Build tags handle OS-specific implementations automatically
	return getOSProvider()
}
