package lumigotracer

import (
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Config describes the struct about the configuration
// of the wrap handler for tracer
type Config struct {
	// enabled switch off SDK completely
	enabled bool

	// Token is used to interact with Lumigo API
	Token string

	// debug log everything
	debug bool

	// tracerProvider to use a dynamic tracer provider for private usage only
	tracerProvider *trace.TracerProvider

	// PrintStdout prints in stdout
	PrintStdout bool
}

// cfg it's a public empty config
var cfg Config

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
	viper.SetDefault("Enabled", true)
	viper.SetDefault("Debug", false)

	recoverWithLogs()
}

func loadConfig(conf Config) error {
	cfg.Token = viper.GetString("Token")
	if cfg.Token == "" {
		cfg.Token = conf.Token
	}
	cfg.enabled = viper.GetBool("Enabled")
	cfg.debug = viper.GetBool("Debug")
	cfg.PrintStdout = conf.PrintStdout
	return cfg.validate()
}
