package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beastixq/marketplace/internal/config"
)

const validYAML = `
server:
  addr: ":8080"
database:
  dsn: "postgres://u:p@host/db"
auth:
  jwt_secret: "yaml-secret"
  jwt_ttl: "24h"
  bcrypt_cost: 10
payment:
  ttl: "15m"
  gateway_url: "http://localhost:9000"
orders:
  expiration_check_interval: "1m"
logging:
  level: "info"
  format: "json"
  file: "logs/test.log"
  console: true
  add_source: false
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("JWT_SECRET")

	cfg, err := config.Load(writeConfig(t, validYAML))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Auth.JWTSecret != "yaml-secret" {
		t.Errorf("jwt_secret: got %q", cfg.Auth.JWTSecret)
	}
	if cfg.Auth.JWTTTL.Std() != 24*time.Hour {
		t.Errorf("jwt_ttl: got %v", cfg.Auth.JWTTTL.Std())
	}
	if cfg.Payment.TTL.Std() != 15*time.Minute {
		t.Errorf("payment.ttl: got %v", cfg.Payment.TTL.Std())
	}
}

func TestLoad_EnvOverridesSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://from-env")
	t.Setenv("JWT_SECRET", "env-secret")

	cfg, err := config.Load(writeConfig(t, validYAML))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.DSN != "postgres://from-env" {
		t.Errorf("DSN env override failed: got %q", cfg.Database.DSN)
	}
	if cfg.Auth.JWTSecret != "env-secret" {
		t.Errorf("JWT secret env override failed: got %q", cfg.Auth.JWTSecret)
	}
}

func TestLoad_RejectsUnknownField(t *testing.T) {
	body := validYAML + "\nunknown_top_level: 42\n"
	_, err := config.Load(writeConfig(t, body))
	if err == nil {
		t.Fatal("expected error on unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_top_level") {
		t.Errorf("error should mention unknown field: %v", err)
	}
}

func TestLoad_RejectsBadDuration(t *testing.T) {
	body := strings.Replace(validYAML, `jwt_ttl: "24h"`, `jwt_ttl: "not-a-duration"`, 1)
	_, err := config.Load(writeConfig(t, body))
	if err == nil {
		t.Fatal("expected error on bad duration")
	}
}

func TestValidate(t *testing.T) {
	base := func() *config.Config {
		return &config.Config{
			Server:   config.ServerConfig{Addr: ":8080"},
			Database: config.DatabaseConfig{DSN: "postgres://x"},
			Auth: config.AuthConfig{
				JWTSecret:  "s",
				JWTTTL:     config.Duration(time.Hour),
				BcryptCost: 10,
			},
			Payment: config.PaymentConfig{
				TTL:        config.Duration(time.Minute),
				GatewayURL: "http://localhost",
			},
			Orders: config.OrdersConfig{
				ExpirationCheckInterval: config.Duration(time.Second),
			},
			Logging: config.LoggingConfig{
				Level:   "info",
				Format:  "json",
				File:    "logs/x.log",
				Console: true,
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr string
	}{
		{"valid", func(*config.Config) {}, ""},
		{"empty DSN", func(c *config.Config) { c.Database.DSN = "" }, "database.dsn"},
		{"empty JWT", func(c *config.Config) { c.Auth.JWTSecret = "" }, "auth.jwt_secret"},
		{"zero JWT TTL", func(c *config.Config) { c.Auth.JWTTTL = 0 }, "auth.jwt_ttl"},
		{"bcrypt below min", func(c *config.Config) { c.Auth.BcryptCost = 1 }, "auth.bcrypt_cost"},
		{"bcrypt above max", func(c *config.Config) { c.Auth.BcryptCost = 99 }, "auth.bcrypt_cost"},
		{"empty addr", func(c *config.Config) { c.Server.Addr = "" }, "server.addr"},
		{"zero payment TTL", func(c *config.Config) { c.Payment.TTL = 0 }, "payment.ttl"},
		{"empty gateway URL", func(c *config.Config) { c.Payment.GatewayURL = "" }, "payment.gateway_url"},
		{"non-absolute gateway URL", func(c *config.Config) { c.Payment.GatewayURL = "/relative" }, "payment.gateway_url"},
		{"zero orders interval", func(c *config.Config) { c.Orders.ExpirationCheckInterval = 0 }, "orders.expiration_check_interval"},
		{"bad log level", func(c *config.Config) { c.Logging.Level = "trace" }, "logging.level"},
		{"bad log format", func(c *config.Config) { c.Logging.Format = "xml" }, "logging.format"},
		{"no log sinks", func(c *config.Config) { c.Logging.File = ""; c.Logging.Console = false }, "logging:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error on missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}
