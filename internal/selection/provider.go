package selection

// Provider defines interface for getting selected/clipboard text
type Provider interface {
	Get() (string, error)
	Name() string
}
