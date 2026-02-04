// Package cfg provides a deadsimple interface to load configurations
package cfg

import "errors"

type Config interface {
	// Get returns ErrInvalidType if pointer type is mismatched
	Get(Key, any) error
	GetMany([]Key, ...any) error
}

type Key string

var (
	ErrInvalidArg      = errors.New("invalid arg")
	ErrMissingRequired = errors.New("missing required value")
)
