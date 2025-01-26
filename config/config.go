package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Xendit XenditConfig `json:"xendit"`
	Stripe StripeConfig `json:"stripe"`
	Server ServerConfig `json:"server"`
}

type XenditConfig struct {
	Secret string `json:"secret"`
	Public string `json:"public"`
}

type StripeConfig struct {
	Secret string `json:"secret"`
	Public string `json:"public"`
}

type ServerConfig struct {
	Port string `json:"port"`
}

// LoadConfig loads configuration from a JSON file and environment variables
func LoadConfig() (*Config, error) {
	// First load from config file
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

	// Override with environment variables if they exist
	if envSecret := os.Getenv("XENDIT_API_KEY"); envSecret != "" {
		config.Xendit.Secret = envSecret
	}
	if envSecret := os.Getenv("STRIPE_API_KEY"); envSecret != "" {
		config.Stripe.Secret = envSecret
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		config.Server.Port = envPort
	} else if config.Server.Port == "" {
		config.Server.Port = "8080" // Default port
	}

	return config, nil
}
