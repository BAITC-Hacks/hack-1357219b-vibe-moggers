package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	AIMode      string
	OpenAIKey   string
	OpenAIModel string
	DemoMode    bool
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("could not parse .env")
	}
	cfg := Config{
		Port:        valueOrDefault("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AIMode:      valueOrDefault("AI_MODE", "fallback"),
		OpenAIKey:   strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel: valueOrDefault("OPENAI_MODEL", "gpt-4.1-mini"),
	}
	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be 1-65535")
	}
	cfg.DemoMode, err = strconv.ParseBool(valueOrDefault("DEMO_MODE", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("DEMO_MODE must be true or false")
	}
	if cfg.AIMode != "live" && cfg.AIMode != "fallback" {
		return Config{}, fmt.Errorf("AI_MODE must be live or fallback")
	}
	if cfg.AIMode == "live" && cfg.OpenAIKey == "" {
		return Config{}, fmt.Errorf("OPENAI_API_KEY is required for AI_MODE=live")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
