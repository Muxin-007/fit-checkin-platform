package config

import (
	"fmt"
	"strings"

	commonConfig "platform/common/config"
)

type Config struct {
	Port         uint16                 `yaml:"port"`
	RouterPrefix string                 `yaml:"router-prefix"`
	Env          string                 `default:"develop" yaml:"env"`
	Log          commonConfig.LogConfig `yaml:"log"`
}

// Validate implement Loader interface
func (c *Config) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if !strings.HasPrefix(c.RouterPrefix, "/") {
		return fmt.Errorf("router-prefix must start with /")
	}
	if c.Log.Module == "" {
		c.Log.Module = "platform"
	}
	return nil
}

// LoadConfig from file
func LoadConfig(path string) (*Config, error) {
	return commonConfig.Load[*Config](path)
}
