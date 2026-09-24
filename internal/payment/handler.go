package payment

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type CheckoutHandler struct {
	repo   *business.Repository
	stripe *StripeClient
}

func NewCheckoutHandler(repo *business.Repository, stripe *StripeClient) *CheckoutHandler {
	return &CheckoutHandler{repo: repo, stripe: stripe}
}

type checkoutRequest struct {
	WhatsAppNumber string `json:"whatsapp_number"`
	ServiceSlug    string `json:"service_slug"`
}

type checkoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// ServeHTTP faz o CheckoutHandler satisfazer a interface http.Handler sozinho —
// diferente do healthHandler (que precisava de http.HandlerFunc explícito),
// qualquer struct com esse método já PODE ser registrada direto no mux.
// ServeHTTP cria uma sessão de pagamento na Stripe para o serviço escolhido.
// @Summary      Cria checkout de pagamento
// @Description  Recebe o número de WhatsApp e o serviço escolhido, cria o pedido e retorna a URL de pagamento da Stripe
// @Tags         payment
// @Accept       json
// @Produce      json
// @Param        request body checkoutRequest true "Dados do checkout"
// @Success      200 {object} checkoutResponse
// @Failure      400 {string} string "requisição inválida"
// @Failure      500 {string} string "erro interno"
// @Router       /checkout [post]
func (h *CheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	if req.WhatsAppNumber == "" || req.ServiceSlug == "" {
		http.Error(w, "whatsapp_number e service_slug são obrigatórios", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, err := h.repo.FindOrCreateUserByWhatsApp(ctx, req.WhatsAppNumber)
	if err != nil {
		log.Printf("erro ao buscar/criar usuário: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	serviceID, err := h.repo.GetServiceIDBySlug(ctx, req.ServiceSlug)
	if err != nil {
		http.Error(w, "serviço inválido", http.StatusBadRequest)
		return
	}

	amountCents, err := h.stripe.GetPriceAmountCents(req.ServiceSlug)
	if err != nil {
		log.Printf("erro ao buscar preço na Stripe: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	orderID, err := h.repo.CreateOrder(ctx, userID, serviceID, amountCents)
	if err != nil {
		log.Printf("erro ao criar pedido: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	session, err := h.stripe.CreateCheckoutSession(req.ServiceSlug, orderID)
	if err != nil {
		log.Printf("erro ao criar checkout session: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	if err := h.repo.AttachStripeSession(ctx, orderID, session.ID); err != nil {
		log.Printf("erro ao vincular sessão ao pedido: %v", err)
		// não retornamos erro pro cliente aqui — o pagamento já pode ser feito,
		// esse dado é só pra rastreabilidade; logamos e seguimos
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkoutResponse{CheckoutURL: session.URL})
}