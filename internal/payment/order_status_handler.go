package payment

import (
	"encoding/json"
	"net/http"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type OrderStatusHandler struct{ repo *business.Repository }

func NewOrderStatusHandler(repo *business.Repository) *OrderStatusHandler {
	return &OrderStatusHandler{repo: repo}
}

func (h *OrderStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	details, err := h.repo.GetOrderDetails(r.Context(), r.URL.Query().Get("order_id"))
	if err != nil {
		http.Error(w, "pedido não encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       details.Status,
		"service_slug": details.ServiceSlug,
	})
}