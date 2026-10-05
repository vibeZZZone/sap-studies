package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	LogLevel    string

	BonusCashbackPercent int
	BonusSpendLimitPct   int

	SeedOnStartup bool
	HTTPTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  env("PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		LogLevel:              env("LOG_LEVEL", "info"),
		BonusCashbackPercent: 5,
		BonusSpendLimitPct:   50,
		SeedOnStartup:         true,
		HTTPTimeout:           15 * time.Second,
	}

	if p, ok := os.LookupEnv("BONUS_CASHBACK_PERCENT"); ok {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || v > 100 {
			return Config{}, fmt.Errorf("BONUS_CASHBACK_PERCENT must be an integer in [0,100], got %q", p)
		}
		cfg.BonusCashbackPercent = v
	}
	if p, ok := os.LookupEnv("BONUS_SPEND_LIMIT_PERCENT"); ok {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || v > 100 {
			return Config{}, fmt.Errorf("BONUS_SPEND_LIMIT_PERCENT must be an integer in [0,100], got %q", p)
		}
		cfg.BonusSpendLimitPct = v
	}
	if s, ok := os.LookupEnv("SEED_ON_STARTUP"); ok {
		v, err := strconv.ParseBool(s)
		if err != nil {
			return Config{}, fmt.Errorf("SEED_ON_STARTUP must be a boolean, got %q", s)
		}
		cfg.SeedOnStartup = v
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}