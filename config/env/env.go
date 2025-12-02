// Package env provides a dead simple interface to load environment variables
package env

import (
	"fmt"
	"os"
	"reflect"

	"github.com/benjamonnguyen/deadsimple/config"
	"github.com/joho/godotenv"
)

type Entry struct {
	Key     config.Key
	Default string
	// Required is ignored if Default is provided
	Required bool
}

func NewConfig(src string, entries ...Entry) (config.Config, error) {
	kvs := make(map[config.Key]string)

	var fromSrc map[string]string
	if src != "" {
		m, err := godotenv.Read(src)
		if err != nil {
			return nil, err
		}
		fromSrc = m
	}

	for _, entry := range entries {
		// Get environment variable value (highest priority)
		value := os.Getenv(string(entry.Key))

		// If no env var, check file values
		if value == "" && src != "" {
			if fileValue, exists := fromSrc[string(entry.Key)]; exists {
				value = fileValue
			}
		}

		// If still no value, use default
		if value == "" {
			value = entry.Default
		}

		// Check if required field is missing
		if entry.Required && value == "" {
			return nil, fmt.Errorf("key: %s: %w", entry.Key, config.ErrMissingRequired)
		}

		kvs[entry.Key] = value
	}

	return &cfg{kvs: kvs}, nil
}

type cfg struct {
	kvs map[config.Key]string
}

func (c *cfg) Get(k config.Key, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("expected string pointer: %w", config.ErrInvalidArg)
	}

	if rv.Elem().Kind() != reflect.String {
		return fmt.Errorf("expected string pointer: %w", config.ErrInvalidArg)
	}

	val, ok := c.kvs[k]
	if !ok {
		return nil // Key not found, leave v unchanged
	}

	rv.Elem().SetString(val)
	return nil
}

func (c *cfg) GetMany(ks []config.Key, vs ...any) error {
	if len(ks) != len(vs) {
		return fmt.Errorf("provide string pointer for each key: %w", config.ErrInvalidArg)
	}

	for i, v := range vs {
		if err := c.Get(ks[i], v); err != nil {
			return err
		}
	}

	return nil
}
