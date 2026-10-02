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
    StripeWebhookSecret   string
	ResendAPIKey   		  string
	ResendFromEmail       string
	CalcomAPIKey          string
	CalcomEventTypeID     string
	CalcomWebhookSecret   string
	CalcomOrganizerEmail  string
	CalcomTimezone        string
	AppEnv                string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	appEnv := os.Getenv("APP_ENV")
	if appEnv == ""{
		appEnv = "development"
	}

	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		Port:                  os.Getenv("PORT"),
		StripeSecretKey:       os.Getenv("STRIPE_SECRET_KEY"),
		StripePriceMigramovil: os.Getenv("STRIPE_PRICE_MIGRAMOVIL"),
		StripePriceMigracion:  os.Getenv("STRIPE_PRICE_MIGRACION"),
        StripeWebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
		ResendAPIKey:          os.Getenv("RESEND_API_KEY"),
		ResendFromEmail:       os.Getenv("RESEND_FROM_EMAIL"),
		CalcomAPIKey:          os.Getenv("CALCOM_API_KEY"),
		CalcomEventTypeID:     os.Getenv("CALCOM_EVENT_TYPE_ID"),
		CalcomWebhookSecret:   os.Getenv("CALCOM_WEBHOOK_SECRET"),
		CalcomOrganizerEmail: os.Getenv("CALCOM_ORGANIZER_EMAIL"),
		CalcomTimezone:        os.Getenv("CALCOM_TIMEZONE"),
		AppEnv: appEnv,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("variável de ambiente DATABASE_URL não definida")
	}
	if cfg.StripeSecretKey == "" {
		return nil, fmt.Errorf("variável de ambiente STRIPE_SECRET_KEY não definida")
	}
	if cfg.StripeWebhookSecret == "" { 
		return nil, fmt.Errorf("variável de ambiente STRIPE_WEBHOOK_SECRET não definida")
	}
	if cfg.CalcomTimezone == "" {
		cfg.CalcomTimezone = "America/Asuncion" // ajusta se o Guerreiro estiver em outro fuso
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}