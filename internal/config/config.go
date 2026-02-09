package config

import (
	"os"
)

type Config struct {
	DatabasePath string
	Port         string
	Environment  string
	LocaleDir    string
}

func Load() *Config {
	return &Config{
		DatabasePath: getEnv("DATABASE_PATH", "./data/subvault.db"),
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("GIN_MODE", "debug"),
		LocaleDir:    getEnv("LOCALE_DIR", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
