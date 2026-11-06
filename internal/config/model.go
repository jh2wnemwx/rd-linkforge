// Package config defines how runtime configuration is obtained. Configuration
// is environment-only and validated at startup.
package config

import (
	"errors"
	"log/slog"
	"os"
)

var (
	ErrMissing = errors.New("config: required variable is not set")
	ErrInvalid = errors.New("config: invalid value")
)

// Config is the fully resolved runtime configuration.
type Config struct {
	HTTPAddr  string
	AdminAddr string
	TCPAddr   string
	PGDSN     string
	MongoURI  string
	JWTSecret string
	CacheSize int
	LogLevel  slog.Level
}
