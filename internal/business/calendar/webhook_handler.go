package calendar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type CalcomWebhookHandler struct {
	repo           *business.Repository
	secret         string
	organizerEmail string
}

func NewCalcomWebhookHandler(repo *business.Repository, secret, organizerEmail string) *CalcomWebhookHandler {
	return &CalcomWebhookHandler{repo: repo, secret: secret, organizerEmail: organizerEmail}
}

type calcomPayload struct {
	TriggerEvent string `json:"triggerEvent"`
	Payload      struct {
		UID         string `json:"uid"`
		CancelledBy string `json:"cancelledBy"`
		StartTime   string `json:"startTime"`
		RescheduleUID string `json:"rescheduleUid"`
		Attendees   []struct {
			Email  string `json:"email"`
			NoShow bool   `json:"noShow"`
		} `json:"attendees"`
	} `json:"payload"`
}

func (h *CalcomWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "erro ao ler corpo", http.StatusServiceUnavailable)
		return
	}

	// Verificação manual de assinatura: a Cal.com manda um HMAC-SHA256 do
	// corpo cru, usando o secret como chave — calculamos o mesmo hash aqui
	// e comparamos. hmac.Equal (em vez de == direto) evita "timing attacks",
	// onde alguém descobre o segredo medindo quanto tempo a comparação leva.
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	receivedSignature := r.Header.Get("x-cal-signature-256")

	if !hmac.Equal([]byte(expectedSignature), []byte(receivedSignature)) {
		log.Println("assinatura do webhook Cal.com inválida")
		http.Error(w, "assinatura inválida", http.StatusBadRequest)
		return
	}

	var event calcomPayload
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	bookingUID := event.Payload.UID
	ctx := r.Context()

	log.Printf("webhook Cal.com: evento=%s uid=%s rescheduleUid=%s cancelledBy=%q",
		event.TriggerEvent, bookingUID, event.Payload.RescheduleUID, event.Payload.CancelledBy)

	var opErr error
	switch event.TriggerEvent {
	case "BOOKING_CANCELLED":
		status := "cancelado_pelo_cliente"
		// EqualFold ignora maiúsculas/minúsculas: emails não deveriam diferir por isso
		if strings.EqualFold(event.Payload.CancelledBy, h.organizerEmail) {
			status = "cancelado_pelo_consultor"
		}
		opErr = h.repo.UpdateConsultationStatus(ctx, bookingUID, status)

	case "BOOKING_RESCHEDULED":
		startTime, err := time.Parse(time.RFC3339, event.Payload.StartTime)
		if err != nil {
			opErr = fmt.Errorf("startTime inválido %q: %w", event.Payload.StartTime, err)
			break
		}
		currentUID := event.Payload.RescheduleUID
		if currentUID == "" {
			currentUID = bookingUID
		}
		opErr = h.repo.UpdateConsultationSchedule(ctx, currentUID, bookingUID, startTime)

	case "BOOKING_NO_SHOW_UPDATED":
		clientNoShow := false
		for _, a := range event.Payload.Attendees {
			if a.NoShow {
				clientNoShow = true
			}
		}
		if clientNoShow {
			opErr = h.repo.MarkNoShowAndIncrement(ctx, bookingUID, "no_show_cliente")
		} else {
			opErr = h.repo.UpdateConsultationStatus(ctx, bookingUID, "no_show_consultor")
		}

	default:
		log.Printf("evento Cal.com ignorado: %s", event.TriggerEvent)
	}

	if opErr != nil {
		log.Printf("erro ao processar %s (uid=%s): %v", event.TriggerEvent, bookingUID, opErr)
		// "Não achei essa consultoria" não se resolve com reenvio; falha de banco sim.
		if !errors.Is(opErr, business.ErrConsultationNotFound) {
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}