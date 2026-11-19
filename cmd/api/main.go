package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	_ "time/tzdata"

	_ "github.com/guerreirodafronteira/guerreiro-api-negocio/docs"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business/calendar"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/config"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/db"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/middleware"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/notify"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/payment"
)

type checkResult struct {
	Name string
	OK   bool
	Err  error
}

// @title Guerreiro da Fronteira - API
// @version 1.0
// @description API de negócio: pagamentos, pedidos e consultoria
// @host localhost:8080
// @BasePath /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	repo := business.NewRepository(pool)
	stripeClient := payment.NewStripeClient(cfg.StripeSecretKey, map[string]string{
	"migramovil":  cfg.StripePriceMigramovil,
	"migracion":   cfg.StripePriceMigracion,
	"consultoria": cfg.StripePriceConsultoria,
		})
	checkoutHandler := payment.NewCheckoutHandler(repo, stripeClient)
	orderStatusHandler := payment.NewOrderStatusHandler(repo)
	emailClient := notify.NewResendClient(cfg.ResendAPIKey, cfg.ResendFromEmail)
	whatsappNotifier := notify.NewWhatsAppNotifier()
	webhookHandler := payment.NewWebhookHandler(repo, cfg.StripeWebhookSecret, emailClient, whatsappNotifier)
	calcomClient := calendar.NewCalcomClient(cfg.CalcomAPIKey, cfg.CalcomEventTypeID, cfg.CalcomTimezone)
	calcomWebhookHandler := calendar.NewCalcomWebhookHandler(repo, cfg.CalcomWebhookSecret, cfg.CalcomOrganizerEmail)
	consultationHandler := calendar.NewConsultationHandler(repo, calcomClient)

	protect := func(h http.Handler) http.Handler {
	return middleware.RequireAPIKey(cfg.InternalAPIKey, h)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(pool))
	mux.Handle("/webhooks/stripe", webhookHandler)       // protegido pela assinatura da Stripe
	mux.Handle("/webhooks/calcom", calcomWebhookHandler) // protegido pela assinatura do Cal.com

	// Rotas chamadas pelo Typebot: exigem X-API-Key
	mux.Handle("/checkout", protect(checkoutHandler))
	mux.Handle("/pedidos/status", protect(orderStatusHandler))
	mux.Handle("/consultoria/horarios", protect(http.HandlerFunc(consultationHandler.GetAvailableSlots)))
	mux.Handle("/consultoria/agendar", protect(http.HandlerFunc(consultationHandler.CreateBooking)))
	if cfg.AppEnv == "development" {
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	log.Println("Swagger disponível em /swagger/index.html")
}

	log.Printf("servidor rodando na porta %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := []struct {
			name string
			fn   func(context.Context) error
		}{
			{"database", func(ctx context.Context) error { return pool.Ping(ctx) }},
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		results := make(chan checkResult, len(checks))
		var wg sync.WaitGroup

		for _, c := range checks {
			wg.Add(1)
			go func(name string, fn func(context.Context) error) {
				defer wg.Done()
				err := fn(ctx)
				results <- checkResult{Name: name, OK: err == nil, Err: err}
			}(c.name, c.fn)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		allOK := true
		report := make(map[string]string)
		for res := range results {
			if res.OK {
				report[res.Name] = "ok"
			} else {
				report[res.Name] = res.Err.Error()
				allOK = false
			}
		}

		status := http.StatusOK
		if !allOK {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(report)
	}
}