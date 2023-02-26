package config

import "os"

type Config struct {
    XenditSecretKey string
}

func NewConfig() *Config {
    return &Config{
        XenditSecretKey: getEnv("XENDIT_SECRET_KEY", ""),
    }
}

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}
