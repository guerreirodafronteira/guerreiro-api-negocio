package payment

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type WebhookHandler struct {
	repo          *business.Repository
	webhookSecret string
}

func NewWebhookHandler(repo *business.Repository, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{repo: repo, webhookSecret: webhookSecret}
}

// ServeHTTP recebe e processa eventos de webhook da Stripe.
// @Summary      Webhook da Stripe
// @Description  Recebe eventos da Stripe (ex: checkout.session.completed) e atualiza o pedido correspondente
// @Tags         payment
// @Accept       json
// @Produce      json
// @Success      200 {string} string "ok"
// @Failure      400 {string} string "assinatura inválida"
// @Failure      500 {string} string "erro interno"
// @Router       /webhooks/stripe [post]
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Limite de tamanho do corpo — proteção básica contra payload absurdo
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "erro ao ler corpo da requisição", http.StatusServiceUnavailable)
		return
	}

	// A verificação de assinatura PRECISA usar os bytes crus, exatamente como
	// chegaram — por isso lemos com io.ReadAll em vez de json.NewDecoder
	// direto (que já consumiria/interpretaria o body antes da checagem).
	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, h.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("assinatura do webhook inválida: %v", err)
		http.Error(w, "assinatura inválida", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("erro ao decodificar sessão: %v", err)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}

		orderID := session.ClientReferenceID
		if orderID == "" {
		log.Printf("evento %s sem client_reference_id — ignorando (provavelmente evento de teste sintético)", event.ID)
		w.WriteHeader(http.StatusOK) // responde OK mesmo assim, pra Stripe não ficar retentando
		return
		}
		paymentIntentID := ""
		if session.PaymentIntent != nil {
			paymentIntentID = session.PaymentIntent.ID
		}

		if err := h.repo.MarkOrderAsPaid(r.Context(), orderID, paymentIntentID); err != nil {
			log.Printf("erro ao marcar pedido %s como pago: %v", orderID, err)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}

		log.Printf("pedido %s confirmado como pago", orderID)

	default:
		// Não tratamos esse tipo de evento ainda — só logamos e respondemos OK,
		// pra Stripe não ficar tentando reenviar o mesmo evento sem necessidade.
		log.Printf("evento recebido e ignorado: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}