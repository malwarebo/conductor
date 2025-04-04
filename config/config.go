package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Database DatabaseConfig `json:"database"`
	Stripe   StripeConfig   `json:"stripe"`
	Xendit   XenditConfig   `json:"xendit"`
	Server   ServerConfig   `json:"server"`
	Redis    RedisConfig    `json:"redis"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

type StripeConfig struct {
	Secret string `json:"secret"`
	Public string `json:"public"`
}

type XenditConfig struct {
	Secret string `json:"secret"`
	Public string `json:"public"`
}

type ServerConfig struct {
	Port string `json:"port"`
}

type RedisConfig struct {
	Host     string `json:"host" env:"REDIS_HOST" default:"localhost"`
	Port     int    `json:"port" env:"REDIS_PORT" default:"6379"`
	Password string `json:"password" env:"REDIS_PASSWORD" default:""`
	DB       int    `json:"db" env:"REDIS_DB" default:"0"`
	TTL      int    `json:"ttl" env:"REDIS_TTL" default:"86400"` // Default TTL in seconds (24 hours)
}

// LoadConfig loads configuration from a JSON file and environment variables
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Get the absolute path to the config directory
	configDir, err := filepath.Abs("config")
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "config.json")
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	// Override with environment variables if present
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		fmt.Sscanf(dbPort, "%d", &config.Database.Port)
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		config.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		config.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.Database.DBName = dbName
	}
	if dbSSLMode := os.Getenv("DB_SSLMODE"); dbSSLMode != "" {
		config.Database.SSLMode = dbSSLMode
	}
	if stripeKey := os.Getenv("STRIPE_API_KEY"); stripeKey != "" {
		config.Stripe.Secret = stripeKey
	}
	if xenditKey := os.Getenv("XENDIT_API_KEY"); xenditKey != "" {
		config.Xendit.Secret = xenditKey
	}
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}

	// Override Redis config with environment variables if present
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		fmt.Sscanf(redisPort, "%d", &config.Redis.Port)
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		config.Redis.Password = redisPass
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		fmt.Sscanf(redisDB, "%d", &config.Redis.DB)
	}
	if redisTTL := os.Getenv("REDIS_TTL"); redisTTL != "" {
		fmt.Sscanf(redisTTL, "%d", &config.Redis.TTL)
	}

	// Set defaults if not configured
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}

	return config, nil
}

func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
