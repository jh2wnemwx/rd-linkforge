package config

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

const (
	testDSN    = "postgres://lf:pg-password@db:5432/lf"
	testMongo  = "mongodb://lf:mongo-password@mongo:27017"
	testSecret = "0123456789abcdef0123456789abcdef" // exactly 32 bytes
)

// requiredEnv holds only the variables that have no default.
func requiredEnv() map[string]string {
	return map[string]string{
		"LF_PG_DSN":     testDSN,
		"LF_MONGO_URI":  testMongo,
		"LF_JWT_SECRET": testSecret,
	}
}

// validEnv sets every variable to a valid value, so that a test can change
// one of them and see exactly one problem.
func validEnv() map[string]string {
	env := requiredEnv()
	env["LF_HTTP_ADDR"] = "0.0.0.0:8080"
	env["LF_ADMIN_ADDR"] = "127.0.0.1:6060"
	env["LF_TCP_ADDR"] = "0.0.0.0:7070"
	env["LF_CACHE_SIZE"] = "1000"
	env["LF_LOG_LEVEL"] = "info"
	return env
}

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func load(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	return NewEnvLoader(lookupFrom(env)).Load(context.Background())
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(t, requiredEnv())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		HTTPAddr:  ":8080",
		AdminAddr: "127.0.0.1:6060",
		TCPAddr:   ":7070",
		PGDSN:     testDSN,
		MongoURI:  testMongo,
		JWTSecret: testSecret,
		CacheSize: 100000,
		LogLevel:  slog.LevelInfo,
	}
	if cfg != want {
		t.Errorf("got  %#v\nwant %#v", cfg, want)
	}
}

func TestLoadBlankMeansDefault(t *testing.T) {
	env := requiredEnv()
	for _, key := range []string{"LF_HTTP_ADDR", "LF_ADMIN_ADDR", "LF_TCP_ADDR", "LF_CACHE_SIZE", "LF_LOG_LEVEL"} {
		env[key] = "   "
	}
	cfg, err := load(t, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdminAddr != "127.0.0.1:6060" || cfg.CacheSize != 100000 || cfg.LogLevel != slog.LevelInfo {
		t.Errorf("blank values did not fall back to defaults: %#v", cfg)
	}
}

func TestLoadOverridesAreTrimmed(t *testing.T) {
	env := map[string]string{
		"LF_HTTP_ADDR":  "  0.0.0.0:9000 ",
		"LF_ADMIN_ADDR": "\t127.0.0.1:9001\n",
		"LF_TCP_ADDR":   " localhost:9002 ",
		"LF_PG_DSN":     "  " + testDSN + "  ",
		"LF_MONGO_URI":  " " + testMongo + " ",
		"LF_JWT_SECRET": "  " + testSecret + "  ",
		"LF_CACHE_SIZE": "  42 ",
		"LF_LOG_LEVEL":  " DEBUG ",
	}
	cfg, err := load(t, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		HTTPAddr:  "0.0.0.0:9000",
		AdminAddr: "127.0.0.1:9001",
		TCPAddr:   "localhost:9002",
		PGDSN:     testDSN,
		MongoURI:  testMongo,
		JWTSecret: testSecret,
		CacheSize: 42,
		LogLevel:  slog.LevelDebug,
	}
	if cfg != want {
		t.Errorf("got  %#v\nwant %#v", cfg, want)
	}
}

func TestLoadRules(t *testing.T) {
	const unset = "\x00unset" // marks a variable to remove from the environment

	tests := []struct {
		name    string
		key     string
		value   string
		wantErr error
	}{
		{"missing PG DSN", "LF_PG_DSN", unset, ErrMissing},
		{"blank PG DSN", "LF_PG_DSN", "   ", ErrMissing},
		{"missing Mongo URI", "LF_MONGO_URI", unset, ErrMissing},
		{"missing JWT secret", "LF_JWT_SECRET", unset, ErrMissing},
		{"JWT secret too short", "LF_JWT_SECRET", testSecret[:31], ErrInvalid},
		{"address without port", "LF_HTTP_ADDR", "localhost", ErrInvalid},
		{"bad port", "LF_ADMIN_ADDR", "127.0.0.1:http", ErrInvalid},
		{"port zero", "LF_TCP_ADDR", "0.0.0.0:0", ErrInvalid},
		{"port out of range", "LF_HTTP_ADDR", "0.0.0.0:65536", ErrInvalid},
		{"bad cache size", "LF_CACHE_SIZE", "lots", ErrInvalid},
		{"zero cache size", "LF_CACHE_SIZE", "0", ErrInvalid},
		{"negative cache size", "LF_CACHE_SIZE", "-5", ErrInvalid},
		{"bad log level", "LF_LOG_LEVEL", "loud", ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnv()
			if tt.value == unset {
				delete(env, tt.key)
			} else {
				env[tt.key] = tt.value
			}

			cfg, err := load(t, env)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Errorf("err = %q, want it to name %s", err, tt.key)
			}
			if cfg != (Config{}) {
				t.Errorf("got config %#v with an error, want the zero Config", cfg)
			}
		})
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	env := map[string]string{
		"LF_HTTP_ADDR":  "nonsense",
		"LF_JWT_SECRET": "short",
		"LF_CACHE_SIZE": "0",
		"LF_LOG_LEVEL":  "loud",
	}
	_, err := load(t, env)
	if !errors.Is(err, ErrMissing) || !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want both ErrMissing and ErrInvalid", err)
	}
	for _, key := range []string{"LF_HTTP_ADDR", "LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET", "LF_CACHE_SIZE", "LF_LOG_LEVEL"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("err does not name %s:\n%v", key, err)
		}
	}
}

func TestLoadErrorHidesSecret(t *testing.T) {
	const secret = "hunter2-is-a-secret" // too short, so it is rejected
	env := validEnv()
	env["LF_JWT_SECRET"] = secret

	_, err := load(t, env)
	if err == nil {
		t.Fatal("Load accepted a short secret")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("err leaks the secret: %v", err)
	}
}

func TestLoadCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := NewEnvLoader(lookupFrom(validEnv())).Load(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if cfg != (Config{}) {
		t.Errorf("got config %#v, want the zero Config", cfg)
	}
}

func TestConfigLogValueRedacts(t *testing.T) {
	cfg, err := load(t, validEnv())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("loaded", "config", cfg)
	out := buf.String()

	for _, field := range []string{"config.pg_dsn=[redacted]", "config.mongo_uri=[redacted]", "config.jwt_secret=[redacted]"} {
		if !strings.Contains(out, field) {
			t.Errorf("log does not contain %s:\n%s", field, out)
		}
	}
	for _, secret := range []string{testDSN, testMongo, testSecret, "pg-password", "mongo-password"} {
		if strings.Contains(out, secret) {
			t.Errorf("log leaks %q:\n%s", secret, out)
		}
	}
	if !strings.Contains(out, "config.http_addr=0.0.0.0:8080") || !strings.Contains(out, "config.cache_size=1000") {
		t.Errorf("log is missing the non-secret fields:\n%s", out)
	}
}
