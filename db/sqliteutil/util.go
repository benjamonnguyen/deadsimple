// Package sqliteutil
package sqliteutil

import (
	"strings"
)

type Scannable interface {
	Scan(...any) error
}

func GenerateParameters(n int) string {
	if n == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("(?")
	for range n - 1 {
		sb.WriteString(",?")
	}

	sb.WriteString(")")
	return sb.String()
}
