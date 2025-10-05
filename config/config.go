package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Environment string           `json:"environment"`
	Database    DatabaseConfig   `json:"database"`
	Stripe      StripeConfig     `json:"stripe"`
	Xendit      XenditConfig     `json:"xendit"`
	Server      ServerConfig     `json:"server"`
	Redis       RedisConfig      `json:"redis"`
	OpenAI      OpenAIConfig     `json:"openai"`
	Security    SecurityConfig   `json:"security"`
	Monitoring  MonitoringConfig `json:"monitoring"`
}

type DatabaseConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	User         string        `json:"user"`
	Password     string        `json:"password"`
	DBName       string        `json:"dbname"`
	SSLMode      string        `json:"sslmode"`
	MaxOpenConns int           `json:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns"`
	MaxLifetime  time.Duration `json:"max_lifetime"`
	MaxIdleTime  time.Duration `json:"max_idle_time"`
	ReplicaDSNs  []string      `json:"replica_dsns"`
}

type StripeConfig struct {
	Secret        string `json:"secret"`
	Public        string `json:"public"`
	WebhookSecret string `json:"webhook_secret"`
}

type XenditConfig struct {
	Secret        string `json:"secret"`
	Public        string `json:"public"`
	WebhookSecret string `json:"webhook_secret"`
}

type OpenAIConfig struct {
	APIKey string `json:"api_key"`
}

type ServerConfig struct {
	Port           string        `json:"port"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	MaxHeaderBytes int           `json:"max_header_bytes"`
	EnableTLS      bool          `json:"enable_tls"`
	TLSCertFile    string        `json:"tls_cert_file"`
	TLSKeyFile     string        `json:"tls_key_file"`
}

type RedisConfig struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Password string        `json:"password"`
	DB       int           `json:"db"`
	TTL      time.Duration `json:"ttl"`
	PoolSize int           `json:"pool_size"`
	MinIdle  int           `json:"min_idle"`
}

type SecurityConfig struct {
	JWTSecret        string        `json:"jwt_secret"`
	JWTExpiration    time.Duration `json:"jwt_expiration"`
	EncryptionKey    string        `json:"encryption_key"`
	WebhookSecret    string        `json:"webhook_secret"`
	RateLimitEnabled bool          `json:"rate_limit_enabled"`
	RateLimitRPS     float64       `json:"rate_limit_rps"`
	RateLimitBurst   int           `json:"rate_limit_burst"`
}

type MonitoringConfig struct {
	Enabled         bool   `json:"enabled"`
	MetricsPort     string `json:"metrics_port"`
	HealthCheckPort string `json:"health_check_port"`
	AlertingEnabled bool   `json:"alerting_enabled"`
	LogLevel        string `json:"log_level"`
	LogFormat       string `json:"log_format"`
	EnableTracing   bool   `json:"enable_tracing"`
	TracingEndpoint string `json:"tracing_endpoint"`
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	config.Environment = env

	configDir, err := filepath.Abs("config")
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
	}

	config.loadFromEnv()

	config.setEnvironmentDefaults()

	return config, nil
}

func (c *Config) loadFromEnv() {
	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := fmt.Sscanf(port, "%d", &c.Database.Port); err == nil {
			c.Database.Port = p
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		c.Database.DBName = dbname
	}
	if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}

	if stripeSecret := os.Getenv("STRIPE_SECRET"); stripeSecret != "" {
		c.Stripe.Secret = stripeSecret
	}
	if stripePublic := os.Getenv("STRIPE_PUBLIC"); stripePublic != "" {
		c.Stripe.Public = stripePublic
	}
	if stripeWebhook := os.Getenv("STRIPE_WEBHOOK_SECRET"); stripeWebhook != "" {
		c.Stripe.WebhookSecret = stripeWebhook
	}

	if xenditSecret := os.Getenv("XENDIT_SECRET"); xenditSecret != "" {
		c.Xendit.Secret = xenditSecret
	}
	if xenditPublic := os.Getenv("XENDIT_PUBLIC"); xenditPublic != "" {
		c.Xendit.Public = xenditPublic
	}
	if xenditWebhook := os.Getenv("XENDIT_WEBHOOK_SECRET"); xenditWebhook != "" {
		c.Xendit.WebhookSecret = xenditWebhook
	}

	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		c.OpenAI.APIKey = openaiKey
	}

	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		c.Server.Port = serverPort
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		c.Security.JWTSecret = jwtSecret
	}
	if encryptionKey := os.Getenv("ENCRYPTION_KEY"); encryptionKey != "" {
		c.Security.EncryptionKey = encryptionKey
	}
	if webhookSecret := os.Getenv("WEBHOOK_SECRET"); webhookSecret != "" {
		c.Security.WebhookSecret = webhookSecret
	}
}

func (c *Config) setEnvironmentDefaults() {
	switch c.Environment {
	case "production":
		c.setProductionDefaults()
	case "staging":
		c.setStagingDefaults()
	default: // development
		c.setDevelopmentDefaults()
	}
}

func (c *Config) setDevelopmentDefaults() {
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Redis.TTL == 0 {
		c.Redis.TTL = time.Hour
	}
	if c.Security.RateLimitRPS == 0 {
		c.Security.RateLimitRPS = 1000.0
	}
	if c.Security.RateLimitBurst == 0 {
		c.Security.RateLimitBurst = 2000
	}
}

func (c *Config) setStagingDefaults() {
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 500
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 50
	}
	if c.Redis.TTL == 0 {
		c.Redis.TTL = 12 * time.Hour
	}
	if c.Security.RateLimitRPS == 0 {
		c.Security.RateLimitRPS = 500.0
	}
	if c.Security.RateLimitBurst == 0 {
		c.Security.RateLimitBurst = 1000
	}
}

func (c *Config) setProductionDefaults() {
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 1000
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 100
	}
	if c.Database.MaxLifetime == 0 {
		c.Database.MaxLifetime = time.Hour
	}
	if c.Database.MaxIdleTime == 0 {
		c.Database.MaxIdleTime = 10 * time.Minute
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 15 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 15 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Redis.TTL == 0 {
		c.Redis.TTL = 24 * time.Hour
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 100
	}
	if c.Redis.MinIdle == 0 {
		c.Redis.MinIdle = 10
	}
	if c.Security.RateLimitRPS == 0 {
		c.Security.RateLimitRPS = 100.0
	}
	if c.Security.RateLimitBurst == 0 {
		c.Security.RateLimitBurst = 200
	}
}

func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Stripe.Secret == "" {
		return fmt.Errorf("Stripe secret key is required")
	}
	if c.Xendit.Secret == "" {
		return fmt.Errorf("Xendit secret key is required")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	return nil
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

func (c *Config) GetRedisURL() string {
	return fmt.Sprintf("redis://%s:%d", c.Redis.Host, c.Redis.Port)
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsStaging() bool {
	return c.Environment == "staging"
}
