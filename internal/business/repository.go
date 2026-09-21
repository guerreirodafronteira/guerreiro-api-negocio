package business

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindOrCreateUserByWhatsApp(ctx context.Context, whatsappNumber string) (string, error) {
	var userID string

	err := r.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE whatsapp_number = $1`,
		whatsappNumber,
	).Scan(&userID)

	if err == nil {
		return userID, nil
	}

	err = r.pool.QueryRow(ctx,
		`INSERT INTO users (whatsapp_number) VALUES ($1) RETURNING id`,
		whatsappNumber,
	).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (r *Repository) GetServiceIDBySlug(ctx context.Context, slug string) (string, error) {
	var serviceID string
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM services WHERE slug = $1 AND active = true`,
		slug,
	).Scan(&serviceID)
	return serviceID, err
}

func (r *Repository) CreateOrder(ctx context.Context, userID, serviceID string, amountCents int64) (string, error) {
	var orderID string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO orders (user_id, service_id, status, amount_cents)
		 VALUES ($1, $2, 'pendente', $3) RETURNING id`,
		userID, serviceID, amountCents,
	).Scan(&orderID)
	return orderID, err
}

func (r *Repository) AttachStripeSession(ctx context.Context, orderID, sessionID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orders SET stripe_checkout_session_id = $1, updated_at = now() WHERE id = $2`,
		sessionID, orderID,
	)
	return err
}