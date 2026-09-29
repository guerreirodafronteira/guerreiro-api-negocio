package business

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

type OrderDetails struct {
	UserID         string
	WhatsAppNumber string
	Email          string
	ServiceSlug    string
	ServiceName    string
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

func (r *Repository) MarkOrderAsPaid(ctx context.Context, orderID, paymentIntentID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orders SET status = 'pago', stripe_payment_intent_id = $1, updated_at = now() WHERE id = $2`,
		paymentIntentID, orderID,
	)
	return err
}

func (r *Repository) UpdateUserContactInfo(ctx context.Context, orderID, email, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users
		 SET email = COALESCE(NULLIF($1, ''), email),
		     name  = COALESCE(NULLIF($2, ''), name)
		 WHERE id = (SELECT user_id FROM orders WHERE id = $3)`,
		email, name, orderID,
	)
	return err
}

func (r *Repository) GetOrderDetails(ctx context.Context, orderID string) (*OrderDetails, error) {
	var d OrderDetails
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.whatsapp_number, COALESCE(u.email, ''), s.slug, s.name
		 FROM orders o
		 JOIN users u ON u.id = o.user_id
		 JOIN services s ON s.id = o.service_id
		 WHERE o.id = $1`,
		orderID,
	).Scan(&d.UserID, &d.WhatsAppNumber, &d.Email, &d.ServiceSlug, &d.ServiceName)
	if err != nil {
		return nil, err
	}
	return &d, nil
}