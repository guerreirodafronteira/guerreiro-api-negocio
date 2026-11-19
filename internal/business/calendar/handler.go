package calendar

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type ConsultationHandler struct {
	repo   *business.Repository
	calcom *CalcomClient
}

func NewConsultationHandler(repo *business.Repository, calcom *CalcomClient) *ConsultationHandler {
	return &ConsultationHandler{repo: repo, calcom: calcom}
}

type slotOption struct {
	StartISO string `json:"start_iso"`
	Label    string `json:"label"` // ex: "06/10 13:00", pronto pra exibir no bot
}

// GetAvailableSlots
// @Summary      Lista horários livres para consultoria
// @Tags         consultation
// @Produce      json
// @Param        order_id query string true "ID do pedido"
// @Success      200 {object} map[string][]slotOption
// @Router       /consultoria/horarios [get]
func (h *ConsultationHandler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id é obrigatório", http.StatusBadRequest)
		return
	}

	details, err := h.repo.GetOrderDetails(r.Context(), orderID)
	if err != nil {
		http.Error(w, "pedido não encontrado", http.StatusNotFound)
		return
	}
	if details.ServiceSlug != "consultoria" {
		http.Error(w, "esse pedido não é de consultoria", http.StatusBadRequest)
		return
	}
	if details.Status != "pago" {
	http.Error(w, "pagamento ainda não confirmado", http.StatusPaymentRequired)
	return
}

	now := time.Now()
	slots, err := h.calcom.GetAvailableSlots(now, now.AddDate(0, 0, 14))
	if err != nil {
		log.Printf("erro ao buscar horários: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	loc, err := time.LoadLocation("America/Asuncion")
	if err != nil {
		log.Printf("fuso horário indisponível, usando UTC: %v", err)
		loc = time.UTC
	}

	const maxOptions = 9
	if len(slots) > maxOptions {
		slots = slots[:maxOptions]
	}

	options := make([]slotOption, 0, len(slots))
	labels := make([]string, 0, len(slots))
	values := make([]string, 0, len(slots))
	for _, s := range slots {
		opt := slotOption{StartISO: s.Format(time.RFC3339), Label: s.In(loc).Format("02/01 15:04")}
		options = append(options, opt)
		labels = append(labels, opt.Label)
		values = append(values, opt.StartISO)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"slots": options, "labels": labels, "values": values})
}

type scheduleRequest struct {
	OrderID  string `json:"order_id"`
	StartISO string `json:"start_iso"`
}

// CreateBooking
// @Summary      Confirma o agendamento da consultoria
// @Tags         consultation
// @Accept       json
// @Produce      json
// @Param        request body scheduleRequest true "Dados do agendamento"
// @Success      200 {object} map[string]string
// @Router       /consultoria/agendar [post]
func (h *ConsultationHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartISO)
	if err != nil {
		http.Error(w, "start_iso inválido", http.StatusBadRequest)
		return
	}

	details, err := h.repo.GetOrderDetails(r.Context(), req.OrderID)
	if err != nil {
		http.Error(w, "pedido não encontrado", http.StatusNotFound)
		return
	}

	attendeeName := details.Name // fallback simples
	if attendeeName == "" {
	attendeeName = details.Email
	}
	
	if details.Email == "" {
		http.Error(w, "cliente ainda não tem email cadastrado", http.StatusBadRequest)
		return
	}

	bookingUID, err := h.calcom.CreateBooking(startTime, attendeeName, details.Email)
	if err != nil {
		log.Printf("erro ao criar reserva no Cal.com: %v", err)
		http.Error(w, "não foi possível agendar esse horário — talvez já tenha sido reservado", http.StatusConflict)
		return
	}

	if err := h.repo.ConfirmConsultationBooking(r.Context(), req.OrderID, bookingUID, startTime); err != nil {
		log.Printf("erro ao salvar agendamento: %v", err)
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	loc, err := time.LoadLocation("America/Asuncion")
	if err != nil {
		loc = time.UTC
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "agendado",
		"start_iso": req.StartISO,
		"label":     startTime.In(loc).Format("02/01 às 15:04"),
	})
}