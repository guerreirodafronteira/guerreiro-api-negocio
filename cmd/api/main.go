package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/config"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/db"
	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/payment"
)

type checkResult struct {
	Name string
	OK   bool
	Err  error
}

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
	stripeClient := payment.NewStripeClient(cfg.StripeSecretKey, cfg.StripePriceMigramovil, cfg.StripePriceMigracion)
	checkoutHandler := payment.NewCheckoutHandler(repo, stripeClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(pool))
	mux.Handle("/checkout", checkoutHandler) // repara: Handle (não HandleFunc), porque checkoutHandler já é um http.Handler

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