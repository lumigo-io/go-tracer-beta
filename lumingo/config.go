package lumingo

// Config describes the struct about the configuration
// of the wrap handler for tracer
type Config struct {
	// Enabled switch off SDK completely
	Enabled bool

	// Token is used to interact with Lumingo API
	Token string

	// Verbose whether the tracer should send all the possible information (debug mode)
	Verbose bool

	// isTest
	isTest bool
}
