package config

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"

	"platform/common/tools"
)

// Loader config loader interface
type Loader interface {
	Validate() error
}

// Load from YAML file and set default value
// T must be a pointer to a struct
func Load[T Loader](path string) (T, error) {
	var zero T

	// Get type of T
	tType := reflect.TypeOf(zero)
	if tType.Kind() != reflect.Pointer {
		return zero, fmt.Errorf("generic type T must be a pointer to a struct")
	}

	// Allocate the underlying struct
	// tType.Elem() is the struct type
	// reflect.New creates a pointer to the struct
	cfgVal := reflect.New(tType.Elem())
	cfg := cfgVal.Interface().(T)

	// Set default value
	if err := tools.SetDefault(cfg); err != nil {
		return zero, fmt.Errorf("set default failed: %w", err)
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("read config file failed: %w", err)
	}

	// Environment variables are expanded before parsing so production secrets do
	// not need to be committed to the repository.
	expanded := os.ExpandEnv(string(data))
	if err = yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return zero, fmt.Errorf("unmarshal yaml failed: %w", err)
	}

	// Validate config
	if err = cfg.Validate(); err != nil {
		return zero, fmt.Errorf("validate config failed: %w", err)
	}

	return cfg, nil
}
