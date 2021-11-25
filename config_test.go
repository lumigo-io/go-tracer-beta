package lumigotracer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidationMissingToken(t *testing.T) {
	assert.Error(t, ErrInvalidToken, loadConfig(Config{}))
}

func TestConfigEnvVariables(t *testing.T) {
	os.Setenv("LUMIGO_TOKEN", "token")
	os.Setenv("LUMIGO_DEBUG", "true")
	os.Setenv("LUMIGO_ENABLED", "false")

	err := loadConfig(Config{})
	assert.NoError(t, err)
	assert.Equal(t, "token", cfg.Token)
	assert.Equal(t, true, cfg.debug)
	assert.Equal(t, false, cfg.enabled)

	os.Unsetenv("LUMIGO_TOKEN")
	os.Unsetenv("LUMIGO_DEBUG")
	os.Unsetenv("LUMIGO_ENABLED")
}

func TestConfigEnabledByDefault(t *testing.T) {
	os.Setenv("LUMIGO_TOKEN", "token")

	err := loadConfig(Config{})
	assert.NoError(t, err)
	assert.Equal(t, "token", cfg.Token)
	assert.Equal(t, false, cfg.debug)
	assert.Equal(t, true, cfg.enabled)

	os.Unsetenv("LUMIGO_TOKEN")
}
