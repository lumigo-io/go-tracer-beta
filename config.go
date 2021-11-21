package lumigo

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config describes the struct about the configuration
// of the wrap handler for tracer
type Config struct {
	// Enabled switch off SDK completely
	Enabled bool

	// Token is used to interact with Lumigo API
	Token string

	// ServiceName the name of the service to trace
	ServiceName string

	// Endpoint the endpoint of the service
	Endpoint string

	// EnableThreadSafe ndicates that calls may be executed in threads. This flag
	// creates a single parent to all the span in the current interpreter.
	EnableThreadSafe bool `mapstructure:"enable_thread_safe"`

	// Verbose whether the tracer should send all the possible information (debug mode)
	Verbose bool

	// PrintStdout
	PrintStdout bool
}

// validate runs a validation to the required fields
// for this Config struct
func (cfg Config) validate() error { // nolint
	if cfg.Token == "" {
		return ErrInvalidToken
	}
	return nil
}

// init not really used right now
func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("lumigo")

	defaults := map[string]interface{}{
		"enabled":            true,
		"verbose":            false,
		"endpoint":           "lumigo-wrapper-collector.golumigo.com:4317",
		"service_name":       "",
		"enable_thread_safe": false,
		"token":              "",
	}

	for key, value := range defaults {
		viper.SetDefault(key, value)
	}
}

// load will get the environment variables and bind them in
// config struct
func load() (Config, error) { // nolint
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "failed to load")
	}
	if err := cfg.validate(); err != nil {
		return Config{}, errors.Wrap(err, "failed to validate the config")
	}

	return cfg, nil
}
