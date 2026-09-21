package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/price"
)

type StripeClient struct {
	priceIDByService map[string]string
}

func NewStripeClient(secretKey string, priceMigramovil, priceMigracion string) *StripeClient {
	stripe.Key = secretKey

	return &StripeClient{
		priceIDByService: map[string]string{
			"migramovil": priceMigramovil,
			"migracion":  priceMigracion,
		},
	}
}

// GetPriceAmountCents busca o valor do preço DIRETO na Stripe, em vez de
// duplicarmos esse número no nosso .env — a Stripe é a "fonte da verdade"
// do preço, nosso banco só guarda o que foi cobrado em cada pedido específico.
func (c *StripeClient) GetPriceAmountCents(serviceSlug string) (int64, error) {
	priceID, ok := c.priceIDByService[serviceSlug]
	if !ok {
		return 0, fmt.Errorf("serviço desconhecido: %s", serviceSlug)
	}

	p, err := price.Get(priceID, nil)
	if err != nil {
		return 0, err
	}

	return p.UnitAmount, nil
}

func (c *StripeClient) CreateCheckoutSession(serviceSlug, orderID string) (*stripe.CheckoutSession, error) {
	priceID, ok := c.priceIDByService[serviceSlug]
	if !ok {
		return nil, fmt.Errorf("serviço desconhecido: %s", serviceSlug)
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(orderID),
		SuccessURL:        stripe.String("https://guerreirodafronteira.com/pagamento-confirmado"),
		CancelURL:         stripe.String("https://guerreirodafronteira.com/pagamento-cancelado"),
	}

	return session.New(params)
}