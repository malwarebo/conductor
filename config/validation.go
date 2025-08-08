package config

import (
	"fmt"
	"strings"
)

func (c *Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}

	return nil
}

func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

func (c *ServerConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port is required")
	}
	return nil
}

func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("port is required")
	}
	return nil
}

func (c *Config) GetProviderConfig(provider string) map[string]string {
	switch strings.ToLower(provider) {
	case "stripe":
		return map[string]string{
			"secret": c.Stripe.Secret,
			"public": c.Stripe.Public,
		}
	case "xendit":
		return map[string]string{
			"secret": c.Xendit.Secret,
			"public": c.Xendit.Public,
		}
	default:
		return nil
	}
}
