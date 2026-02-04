// Package env provides a deadsimple interface to load environment variables
package env

import (
	"fmt"
	"os"
	"reflect"

	"github.com/benjamonnguyen/deadsimple/cfg"
	"github.com/joho/godotenv"
)

type Entry struct {
	Key     cfg.Key
	Default string
	// Required is ignored if Default is provided
	Required bool
}

func NewConfig(src string, entries ...Entry) (cfg.Config, error) {
	kvs := make(map[cfg.Key]string)

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
			return nil, fmt.Errorf("key: %s: %w", entry.Key, cfg.ErrMissingRequired)
		}

		kvs[entry.Key] = value
	}

	return &config{kvs: kvs}, nil
}

type config struct {
	kvs map[cfg.Key]string
}

func (c *config) Get(k cfg.Key, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("expected string pointer: %w", cfg.ErrInvalidArg)
	}

	if rv.Elem().Kind() != reflect.String {
		return fmt.Errorf("expected string pointer: %w", cfg.ErrInvalidArg)
	}

	val, ok := c.kvs[k]
	if !ok {
		return nil // Key not found, leave v unchanged
	}

	rv.Elem().SetString(val)
	return nil
}

func (c *config) GetMany(ks []cfg.Key, vs ...any) error {
	if len(ks) != len(vs) {
		return fmt.Errorf("provide string pointer for each key: %w", cfg.ErrInvalidArg)
	}

	for i, v := range vs {
		if err := c.Get(ks[i], v); err != nil {
			return err
		}
	}

	return nil
}
