package lumigotracer

import (
	"fmt"

	"github.com/spf13/viper"
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

	// PrintStdout prints in stdout
	PrintStdout bool
}

// cfg it's a public empty config
var cfg Config

// validate runs a validation to the required fields
// for this Config struct
func (cfg Config) validate() error { // nolint
	fmt.Println("In cfg.validate")
	if cfg.Token == "" {
		fmt.Println("Validte failed")
		return ErrInvalidToken
	}
	fmt.Println("Validate succeeded")
	return nil
}

// init not really used right now
func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("lumigo")
	viper.SetDefault("Enabled", true)
	viper.SetDefault("Debug", false)
}

func loadConfig(conf Config) error {
	defer recoverWithLogs()

	cfg.Token = viper.GetString("Token")
	fmt.Println("cfg.Token:", cfg.Token)
	if cfg.Token == "" {
		cfg.Token = conf.Token
		fmt.Println("cfg.Token in if:", cfg.Token)
	}
	cfg.enabled = viper.GetBool("Enabled")
	fmt.Println("cfg.enabled:", cfg.enabled)
	cfg.debug = viper.GetBool("Debug")
	fmt.Println("cfg.debug:", cfg.debug)
	cfg.PrintStdout = conf.PrintStdout
	fmt.Println("cfg.PrintStdout:", cfg.PrintStdout)
	return cfg.validate()
}
