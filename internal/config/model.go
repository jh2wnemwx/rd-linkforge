// Package config defines how runtime configuration is obtained. Configuration
// is environment-only and validated at startup.
package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
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

type EnvLoader struct {
	lookup func(key string) (value string, ok bool)
}

var _ Loader = (*EnvLoader)(nil)

func NewEnvLoader(lookup func(key string) (value string, ok bool)) *EnvLoader {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	return &EnvLoader{
		lookup: lookup,
	}
}

func (l *EnvLoader) Load(ctx context.Context) (Config, error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}

	var errs []error
	cfg := Config{}

	// Helper for address fields with defaults
	loadAddr := func(key, defVal string) string {
		v, ok := l.lookup(key)
		v = strings.TrimSpace(v)
		if !ok || v == "" {
			v = defVal
		}

		if err := validateAddr(key, v); err != nil {
			errs = append(errs, err)
		}

		return v
	}

	cfg.HTTPAddr = loadAddr("LF_HTTP_ADDR", ":8080")
	cfg.AdminAddr = loadAddr("LF_ADMIN_ADDR", "127.0.0.1:6060")
	cfg.TCPAddr = loadAddr("LF_TCP_ADDR", ":7070")

	// Helper for required strings (PGDSN, MongoURI)
	loadRequired := func(key string) string {
		v, ok := l.lookup(key)
		v = strings.TrimSpace(v)

		if !ok || v == "" {
			errs = append(errs, fmt.Errorf("%v: %w", key, ErrMissing))
			return ""
		}

		if key == "LF_JWT_SECRET" && len(v) < 32 {
			errs = append(errs, fmt.Errorf("%v: %w", key, ErrInvalid))
			return ""
		}

		return v
	}

	cfg.PGDSN = loadRequired("LF_PG_DSN")
	cfg.MongoURI = loadRequired("LF_MONGO_URI")
	cfg.JWTSecret = loadRequired("LF_JWT_SECRET")

	cacheVal, cacheOk := l.lookup("LF_CACHE_SIZE")
	cacheVal = strings.TrimSpace(cacheVal)
	if !cacheOk || cacheVal == "" {
		cfg.CacheSize = 100000
	} else {
		n, err := strconv.Atoi(cacheVal)
		if err != nil || n <= 0 {
			errs = append(errs, fmt.Errorf("LF_CACHE_SIZE: %w", ErrInvalid))
		} else {
			cfg.CacheSize = n
		}
	}

	logVal, logOk := l.lookup("LF_LOG_LEVEL")
	logVal = strings.TrimSpace(logVal)
	if !logOk {
		cfg.LogLevel = slog.LevelInfo
	} else {
		lvl, err := ParseLevel(logVal)
		if err != nil {
			errs = append(errs, fmt.Errorf("LF_LOG_LEVEL: %w", err))
		} else {
			cfg.LogLevel = lvl
		}
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w", ErrInvalid)
	}
}

func validateAddr(key, val string) error {
	addrStr, portStr, err := net.SplitHostPort(val)
	if err != nil {
		return fmt.Errorf("%v: %w", key, ErrInvalid)
	}

	if addrStr == "" {
		return fmt.Errorf("%v: %w", key, ErrInvalid)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%v: %w", key, ErrInvalid)
	}

	return nil
}

// LogValue implements slog.LogValuer. The credentials are shown as "[redacted]".
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_addr", c.HTTPAddr),
		slog.String("admin_addr", c.AdminAddr),
		slog.String("tcp_addr", c.TCPAddr),
		slog.String("pg_dsn", "[redacted]"),
		slog.String("mongo_uri", "[redacted]"),
		slog.String("jwt_secret", "[redacted]"),
		slog.Int("cache_size", c.CacheSize),
		slog.Any("log_level", c.LogLevel),
	)
}
