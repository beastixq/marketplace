package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const (
	envDatabaseURL = "DATABASE_URL"
	envJWTSecret   = "JWT_SECRET"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Payment  PaymentConfig  `yaml:"payment"`
	Orders   OrdersConfig   `yaml:"orders"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type AuthConfig struct {
	JWTSecret  string   `yaml:"jwt_secret"`
	JWTTTL     Duration `yaml:"jwt_ttl"`
	BcryptCost int      `yaml:"bcrypt_cost"`
}

type PaymentConfig struct {
	TTL        Duration `yaml:"ttl"`
	GatewayURL string   `yaml:"gateway_url"`
}

type OrdersConfig struct {
	ExpirationCheckInterval Duration `yaml:"expiration_check_interval"`
}

// Duration wraps time.Duration with YAML support for human-readable strings ("15m", "24h").
type Duration time.Duration

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", node.Value, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }

// Load reads YAML from path, then overlays secrets from environment variables.
// DATABASE_URL and JWT_SECRET override the YAML values when present, so that
// production secrets stay out of the file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	if v, ok := os.LookupEnv(envDatabaseURL); ok {
		cfg.Database.DSN = v
	}
	if v, ok := os.LookupEnv(envJWTSecret); ok {
		cfg.Auth.JWTSecret = v
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.DSN == "" {
		return errors.New("database.dsn must be set (use config file or DATABASE_URL env)")
	}
	if c.Auth.JWTSecret == "" {
		return errors.New("auth.jwt_secret must be set (use config file or JWT_SECRET env)")
	}
	if c.Auth.JWTTTL <= 0 {
		return errors.New("auth.jwt_ttl must be positive")
	}
	if c.Auth.BcryptCost < bcrypt.MinCost || c.Auth.BcryptCost > bcrypt.MaxCost {
		return fmt.Errorf("auth.bcrypt_cost must be in [%d, %d], got %d",
			bcrypt.MinCost, bcrypt.MaxCost, c.Auth.BcryptCost)
	}
	if c.Server.Addr == "" {
		return errors.New("server.addr must be set")
	}
	if c.Payment.TTL <= 0 {
		return errors.New("payment.ttl must be positive")
	}
	if c.Payment.GatewayURL == "" {
		return errors.New("payment.gateway_url must be set")
	}
	if u, err := url.Parse(c.Payment.GatewayURL); err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("payment.gateway_url must be absolute URL, got %q", c.Payment.GatewayURL)
	}
	if c.Orders.ExpirationCheckInterval <= 0 {
		return errors.New("orders.expiration_check_interval must be positive")
	}
	return nil
}
