package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	InternalAPIKey string

	StripeSecretKey        string
	StripeWebhookSecret    string
	StripePriceMigramovil  string
	StripePriceMigracion   string
	StripePriceConsultoria string

	ResendAPIKey    string
	ResendFromEmail string

	CalcomAPIKey         string
	CalcomEventTypeID    string
	CalcomWebhookSecret  string
	CalcomTimezone       string
	CalcomOrganizerEmail string
}

type envVar struct {
	name  string
	value string
}

func Load() (*Config, error) {
	_ = godotenv.Load() // em produção não existe .env; as variáveis vêm dos secrets do Fly

	cfg := &Config{
		AppEnv:         os.Getenv("APP_ENV"),
		Port:           os.Getenv("PORT"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		InternalAPIKey: os.Getenv("INTERNAL_API_KEY"),

		StripeSecretKey:        os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePriceMigramovil:  os.Getenv("STRIPE_PRICE_MIGRAMOVIL"),
		StripePriceMigracion:   os.Getenv("STRIPE_PRICE_MIGRACION"),
		StripePriceConsultoria: os.Getenv("STRIPE_PRICE_CONSULTORIA"),

		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFromEmail: os.Getenv("RESEND_FROM_EMAIL"),

		CalcomAPIKey:         os.Getenv("CALCOM_API_KEY"),
		CalcomEventTypeID:    os.Getenv("CALCOM_EVENT_TYPE_ID"),
		CalcomWebhookSecret:  os.Getenv("CALCOM_WEBHOOK_SECRET"),
		CalcomTimezone:       os.Getenv("CALCOM_TIMEZONE"),
		CalcomOrganizerEmail: os.Getenv("CALCOM_ORGANIZER_EMAIL"),
	}

	// Padrões. Repara no APP_ENV: se alguém esquecer de setar, o padrão agora é
	// "production" (mais seguro: sem Swagger e com chave obrigatória). Seu .env
	// local já tem APP_ENV=development, então nada muda pra você.
	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.CalcomTimezone == "" {
		cfg.CalcomTimezone = "America/Asuncion"
	}

	required := []envVar{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"STRIPE_SECRET_KEY", cfg.StripeSecretKey},
		{"STRIPE_WEBHOOK_SECRET", cfg.StripeWebhookSecret},
		{"STRIPE_PRICE_MIGRAMOVIL", cfg.StripePriceMigramovil},
		{"STRIPE_PRICE_MIGRACION", cfg.StripePriceMigracion},
		{"STRIPE_PRICE_CONSULTORIA", cfg.StripePriceConsultoria},
		{"RESEND_API_KEY", cfg.ResendAPIKey},
		{"RESEND_FROM_EMAIL", cfg.ResendFromEmail},
		{"CALCOM_API_KEY", cfg.CalcomAPIKey},
		{"CALCOM_EVENT_TYPE_ID", cfg.CalcomEventTypeID},
		{"CALCOM_WEBHOOK_SECRET", cfg.CalcomWebhookSecret},
		{"CALCOM_ORGANIZER_EMAIL", cfg.CalcomOrganizerEmail},
	}
	// Só "development" dispensa a chave. Qualquer outro valor (inclusive um typo
	// como "prod") exige a chave, ou seja, o erro sempre cai pro lado seguro.
	if cfg.AppEnv != "development" {
		required = append(required, envVar{"INTERNAL_API_KEY", cfg.InternalAPIKey})
	}

	var missing []string
	for _, v := range required {
		if v.value == "" {
			missing = append(missing, v.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("variáveis de ambiente obrigatórias não definidas: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}