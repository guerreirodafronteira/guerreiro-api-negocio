package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL           string
	Port                  string
	StripeSecretKey       string
	StripePriceMigramovil string
	StripePriceMigracion  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		Port:                  os.Getenv("PORT"),
		StripeSecretKey:       os.Getenv("STRIPE_SECRET_KEY"),
		StripePriceMigramovil: os.Getenv("STRIPE_PRICE_MIGRAMOVIL"),
		StripePriceMigracion:  os.Getenv("STRIPE_PRICE_MIGRACION"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("variável de ambiente DATABASE_URL não definida")
	}
	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("variável de ambiente STRIPE_SECRET_KEY não definida")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}